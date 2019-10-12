package gologsgo_test

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"testing"

	logs "github.com/big-squid/go-logs-go"
)

const logEnv = "LOG_CONFIG"

func TestNew(test *testing.T) {
	cfg := logs.RootLogConfig{
		Label: "testnew",
		Level: logs.All,
	}
	// Make sure the constructor works.
	logger := logs.New(&cfg)

	// The default LogHandler uses log.Output, so we can call
	// log.SetOutput to capture our log messages in a bytes.Buffer
	// Redirect output to a custom writer so we can verify log messages get
	// through, are formatted as expected, and are omitted when the message's
	// level is lower than the logger's level.
	var buffer bytes.Buffer
	writer := bufio.NewWriter(&buffer)
	log.SetOutput(writer)
	// Turn off date and time logging for our test - otherwise logs strings
	// change with the time
	flags := log.Flags()
	defer func() {
		// Restore flags (although test is ending)
		log.SetFlags(flags)
	}()
	log.SetFlags(0)

	expectedAllOut := `TRACE [testnew]: A trace log message
DEBUG [testnew]: A debug log message
INFO [testnew]: An info log message
WARN [testnew]: A warn log message
ERROR [testnew]: A error log message
`

	// Run everything to make sure no errors occur.
	logger.Trace("A trace log message")
	logger.Debug("A debug log message")
	logger.Info("An info log message")
	logger.Warn("A warn log message")
	logger.Error("A error log message")

	if logger.Level() != logs.All {
		test.Error("Expected log level to be All for `testnew`")
	}

	writer.Flush()
	actualAllOut := buffer.String()
	if actualAllOut != expectedAllOut {
		test.Errorf("Did not receive expected log messages for `testnew`:\n%s\nShould be:\n%s", actualAllOut, expectedAllOut)
	}

	// reset the buffer for a second test
	buffer.Reset()

	// Make sure the constructor works with defaults.
	defaultLogger := logs.New(&logs.RootLogConfig{})
	if defaultLogger.Level() != logs.Info {
		test.Error("Expected log level to be Info for default root logger")
	}

	// Run everything to make sure no errors occur.
	// We should not see the Trace and Debug messages.
	expectedInfoOut := `INFO: An info log message
WARN: A warn log message
ERROR: A error log message
`

	defaultLogger.Trace("A trace log message")
	defaultLogger.Debug("A debug log message")
	defaultLogger.Info("An info log message")
	defaultLogger.Warn("A warn log message")
	defaultLogger.Error("A error log message")

	writer.Flush()
	actualInfoOut := buffer.String()
	if actualInfoOut != expectedInfoOut {
		test.Errorf("Did not receive expected log messages for default root logger:\n%s\nShould be:\n%s", actualInfoOut, expectedInfoOut)
	}
}

// This will test that the root config is honored.
func TestConfigA(test *testing.T) {
	jsonCfg, err := logs.JsonConfig([]byte(`
	{ "level": "INFO",
	  "label": "main"
	}
`))
	if nil != err {
		test.Errorf("Error preparing RootLogConfig with logging.JsonConfig(): %s", err)
	}
	logger := logs.New(jsonCfg)

	if logger.Level() != logs.Info {
		test.Error("Expected log level to be INFO for `main`")
	}

	logger = logger.ChildLogger("test")
	if logger.Level() != logs.Info {
		test.Error("Expected log level to be INFO for `main.test`")
	}
}

func TestConfigB(test *testing.T) {
	jsonCfg, err := logs.JsonConfig([]byte(`
	{ "level": "ERROR",
      "loggers": {
        "main": {
          "level": "INFO",
          "loggers": {
            "test": {
              "level": "DEBUG"
            }
          }
        }
      }
    }
`))
	if nil != err {
		test.Errorf("Error preparing RootLogConfig with logging.JsonConfig(): %s", err)
	}
	rootLogger := logs.New(jsonCfg)
	if rootLogger.Level() != logs.Error {
		test.Error("Expected log level to be INFO for `main`")
	}

	mainLogger := rootLogger.ChildLogger("main")
	if mainLogger.Level() != logs.Info {
		test.Error("Expected log level to be INFO for `main`")
	}

	testChildLogger := mainLogger.ChildLogger("test")
	if testChildLogger.Level() != logs.Debug {
		test.Error("Expected log level to be DEBUG for `main.test`")
	}
}

func TestEnvPrefixConfig(test *testing.T) {

	os.Setenv("LOGGER_TEST_LEVEL", "TRACE")
	os.Setenv("LOGGER_TEST_LABEL", "main")
	os.Setenv("LOGGER_TEST_LOGGERS__CHILD__LEVEL", "DEBUG")
	os.Setenv("LOGGER_TEST_LOGGERS__CHILD__GRANDCHILD__LEVEL", "INFO")
	os.Setenv("LOGGER_TEST_LOGGERS__JSON_CHILD", `{
		"level": "WARN",
		"loggers": {
			"grandchild": {
				"level": "ERROR"
			}
		}
	}`)
	defer func() {
		os.Unsetenv("LOGGER_TEST_LEVEL")
		os.Unsetenv("LOGGER_TEST_LOGGERS__CHILD__LEVEL")
		os.Unsetenv("LOGGER_TEST_LOGGERS__CHILD__GRANDCHILD__LEVEL")
		os.Unsetenv("LOGGER_TEST_LOGGERS__JSON_CHILD")
	}()

	envCfg, err := logs.EnvPrefixConfig("LOGGER_TEST")
	if nil != err {
		test.Errorf("Error preparing RootLogConfig with logs.EnvPrefixConfig(): %s", err)
	}
	rootLogger := logs.New(envCfg)

	if rootLogger.Level() != logs.Trace {
		test.Error("Expected log level to be TRACE for `main`")
	}

	child := rootLogger.ChildLogger("child")
	if child.Level() != logs.Debug {
		test.Error("Expected log level to be DEBUG for `main.child`")
	}

	grandchild := child.ChildLogger("grandchild")
	if grandchild.Level() != logs.Info {
		grandchild.Error("Expected log level to be Info for `main.child.grandchild`")
	}

	jsonchild := rootLogger.ChildLogger("jsonChild")
	if jsonchild.Level() != logs.Warn {
		test.Error("Expected log level to be WARN for `main.jsonChild`")
	}

	jsongrandchild := jsonchild.ChildLogger("grandchild")
	if jsongrandchild.Level() != logs.Error {
		test.Error("Expected log level to be ERROR for `main.jsonChild.grandchild`")
	}
}