package logging

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// Log constants
const (
  ALL = 1 << 32 - 1
  TRACE = 600
	DEBUG = 500
	INFO  = 400
	WARN  = 300
	ERROR = 200
	FATAL = 100
  OFF = 0
)

type Formatter func(string, ...interface{}) string

type Logger interface {
	New(label string) Logger
	Method(methodName string) Logger
	Log(formatter Formatter, severity int, format string, args ...interface{})
  Trace(format string, args ...interface{})
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})
  LoadConfig(config LogConfig) error
  FileConfig(logFile string) error
  JsonConfig(data []byte) error
  EnvConfig(env string) error
	Level() int
}

type LogConfig struct {
	Loggers map[string]*LogConfig `json:"loggers"`
	Level   *string               `json:"level"`
}

type logFormatter struct {
	label     string
	level     int
	logConfig LogConfig
}

func levelLabel(level int) string {
  if level >= TRACE {
    return "TRACE"
  } else if level >= DEBUG {
    return "DEBUG"
  } else if level >= INFO {
		return "INFO "
	} else if level >= WARN {
		return "WARN "
	} else if level >= ERROR {
		return "ERROR"
  } else if level >= FATAL {
    return "FATAL"
	} else {
		return "OFF"
	}
}

func labelLevel(label string) int {
	switch label {
  case "ALL":
    return ALL
  case "TRACE":
    return TRACE
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
  case "OFF":
    return OFF
	default:
		panic(fmt.Sprintf("Unknown label `%s`", label))
	}
}

func configFor(config LogConfig, label string) *LogConfig {
	parts := strings.Split(label, ".")
	if len(parts) == 0 {
		return nil
	}

	head := parts[0]
	childConfig, ok := config.Loggers[head]
	if !ok {
		return nil
	}
	if childConfig == nil {
		return nil
	}
	if len(parts) == 1 {
		return childConfig
	}

	return configFor(*childConfig, strings.Join(parts[1:], "."))
}

func configLogLevel(defaultLevel int, config LogConfig, label string) int {
	// Dig through the config until we find the level
	childConfig := configFor(config, label)
	if childConfig == nil {
		if config.Level != nil {
			return labelLevel(*config.Level)
		}
		return defaultLevel
	}

	if childConfig.Level != nil {
		return labelLevel(*childConfig.Level)
	}
	return defaultLevel
}

func (logger *logFormatter) LoadConfig(config LogConfig) error {
  var level int
  if config.Level != nil {
    level = labelLevel(*config.Level)
  }

  level = configLogLevel(level, config, logger.label)
  logger.logConfig = config
  logger.level = level

  return nil
}

func (logger *logFormatter) JsonConfig(data []byte) error {
  config := LogConfig{}
  err := json.Unmarshal(data, &config)
  if err != nil {
    return err
  }

  return logger.LoadConfig(config)
}

func (logger *logFormatter) FileConfig(configFile string) error {
  data, err := ioutil.ReadFile(configFile)
  if err != nil {
    return err
  }

  return logger.JsonConfig(data)
}

func (logger *logFormatter) EnvConfig(env string) error {
  return logger.FileConfig(os.Getenv(env))
}

func New(label string, level int, logConfig *LogConfig) *logFormatter {
  if logConfig == nil {
    configLevel := levelLabel(level)
    logConfig = &LogConfig{
      Level: &configLevel,
    }
  }

	return &logFormatter{
		label: label,
		level: level,
    logConfig: *logConfig,
	}
}

func (logger *logFormatter) Level() int {
	return logger.level
}

func (logger *logFormatter) New(label string) Logger {
  label = fmt.Sprintf("%s.%s", logger.label, label)
	level := configLogLevel(logger.level, logger.logConfig, label)

	return &logFormatter{
		label:     label,
		level:     level,
		logConfig: logger.logConfig,
	}
}

func (logger *logFormatter) Method(methodName string) Logger {
	return &logFormatter{
    label: fmt.Sprintf("%s.%s", logger.label, methodName),
		level: logger.level,
    logConfig: logger.logConfig,
	}
}

func (logger *logFormatter) Log(formatter Formatter, severity int, format string, args ...interface{}) {
	if severity > logger.level {
		return
	}

	levelLabel := levelLabel(severity)
	msg := fmt.Sprintf("%s [%s]: %s", levelLabel, logger.label, fmt.Sprintf(format, args...))
	if formatter != nil {
		msg = formatter(msg)
	}
	log.Println(msg)
}

func greyString(format string, args ...interface{}) string {
	return "\x1b[90;1m" + fmt.Sprintf(format, args...) + "\x1b[0m"
}

func (logger *logFormatter) Trace(format string, args ...interface{}) {
  logger.Log(greyString, TRACE, format, args...)
}

func (logger *logFormatter) Debug(format string, args ...interface{}) {
	logger.Log(greyString, DEBUG, format, args...)
}

func (logger *logFormatter) Info(format string, args ...interface{}) {
	logger.Log(color.WhiteString, INFO, format, args...)
}

func (logger *logFormatter) Warn(format string, args ...interface{}) {
	logger.Log(color.YellowString, WARN, format, args...)
}

func (logger *logFormatter) Error(format string, args ...interface{}) {
	logger.Log(color.RedString, ERROR, format, args...)
}

func (logger *logFormatter) Fatal(format string, args ...interface{}) {
	logger.Log(color.RedString, FATAL, format, args...)
	panic("FATAL")
}
