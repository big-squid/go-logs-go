package logging

import (
  "bufio"
  "strings"
  "bytes"
  "testing"
  "log"
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
