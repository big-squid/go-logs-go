package logging

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

const logEnv = "LOG_CONFIG"

func TestNew(test *testing.T) {
	// Make sure the constructor works just fine.
	logger := New("main", DEBUG, nil)

	// Just run everything to make sure no errors occur.
	logger.Info("info")
	logger.Debug("debug")
	logger.Warn("warn")
	logger.Error("error")
}

// This will test that the root config is honored.
func TestConfigA(test *testing.T) {
	var logger Logger
	logger = New("main", DEBUG, nil)
	logger.JsonConfig([]byte(`
    { "level": "INFO"
    }
  `))
	if logger.Level() != INFO {
		test.Error("Expected log level to be INFO for `main`")
	}

	logger = logger.New("main.test")
	if logger.Level() != INFO {
		test.Error("Expected log level to be INFO for `main.test`")
	}
}

func TestConfigB(test *testing.T) {
	var logger Logger
	logger = New("main", DEBUG, nil)
	logger.JsonConfig([]byte(`
    { "level": "ERROR",
      "loggers": {
        "main": {
          "level": "INFO",
          "loggers": {
            "test": {
              "level": "FATAL"
            }
          }
        }
      }
    }
  `))

	if logger.Level() != INFO {
		test.Error("Expected log level to be INFO for `main`")
	}

	logger = logger.New("main.test")
	if logger.Level() != FATAL {
		test.Error("Expected log level to be FATAL for `main.test`")
	}
}

func TestEnvPrefixConfig(test *testing.T) {

	os.Setenv("LOGGER_TEST_LEVEL", "INFO")
	os.Setenv("LOGGER_TEST_LOGGERS__CHILD__LEVEL", "DEBUG")
	os.Setenv("LOGGER_TEST_LOGGERS__CHILD__GRANDCHILD__LEVEL", "TRACE")
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

	var logger Logger
	logger = New("main", DEBUG, nil)
	logger.EnvPrefixConfig("LOGGER_TEST")

	if logger.Level() != INFO {
		test.Error("Expected log level to be INFO for `main`")
	}

	child := logger.New("child")
	if child.Level() != DEBUG {
		test.Error("Expected log level to be DEBUG for `main.child`")
	}

	grandchild := child.New("grandchild")
	if grandchild.Level() != TRACE {
		grandchild.Error("Expected log level to be TRACE for `main.child.grandchild`")
	}

	jsonchild := logger.New("jsonChild")
	if jsonchild.Level() != WARN {
		test.Error("Expected log level to be WARN for `main.jsonChild`")
	}

	jsongrandchild := jsonchild.New("grandchild")
	if jsongrandchild.Level() != ERROR {
		test.Error("Expected log level to be ERROR for `main.jsonChild.grandchild`")
	}
}

func TestLogLevel(test *testing.T) {
	logger := New("main", INFO, nil)

	var buffer bytes.Buffer
	writer := bufio.NewWriter(&buffer)
	log.SetOutput(writer)
	// Redirect output to this stream.
	logger.Info("hello")
	writer.Flush()

	if !strings.Contains(buffer.String(), "hello") {
		test.Errorf("Expected log message `%s` to contain `hello`", buffer.String())
	}

	buffer.Reset()
	logger.Debug("debug msg")
	writer.Flush()

	if strings.Contains(buffer.String(), "debug msg") {
		test.Errorf("Expected log message `%s` to omit `debug msg` but the message still got through.", buffer.String())
	}
}
