package logging

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
)

// Log constants
const (
	ALL   = 1<<32 - 1
	TRACE = 600
	DEBUG = 500
	INFO  = 400
	WARN  = 300
	ERROR = 200
	FATAL = 100
	OFF   = 0
)

type Formatter func(string, ...interface{}) string

// TODO: Consider changing the API of this module to have several utilities
// for generating a LogConfig and a simple constructor that takes a LogConfig.
// Modifying anything but the the root logger's config or even the root
// logger's config after initialization adds a great deal of complexity with
// very little utility.

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
	EnvPrefixConfig(prefix string) error
	Level() int
}

type LogConfig struct {
	Loggers map[string]*LogConfig `json:"loggers"`
	Level   *string               `json:"level"`
}

type logFormatter struct {
	parent            *logFormatter
	isSubLabledLogger bool
	label             string
	level             int
	logConfig         LogConfig
}

func levelLabel(level int) string {
	if level >= TRACE {
		return "TRACE"
	} else if level >= DEBUG {
		return "DEBUG"
	} else if level >= INFO {
		return "INFO"
	} else if level >= WARN {
		return "WARN"
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

// findConfig searches for the labeled config by checking in the parent loggers'
// configs until one is found or we run our of parent loggers.
func findConfig(logger *logFormatter, label string, isSubLabel bool) (*LogConfig, bool) {
	lookupkey := label

	if isSubLabel {
		lookupkey = fmt.Sprintf("%s.%s", logger.label, label)
	}

	if cfg, ok := configForLabel(logger.logConfig, lookupkey); ok {
		return cfg, true
	}

	if nil == logger.parent {
		return nil, false
	}

	return findConfig(logger.parent, lookupkey, logger.isSubLabledLogger)
}

func configForLabel(config LogConfig, label string) (*LogConfig, bool) {
	parts := strings.Split(label, ".")
	if len(parts) == 0 {
		return nil, false
	}

	head := parts[0]
	childConfig, ok := config.Loggers[head]
	if !ok {
		return nil, false
	}
	// If we have a nil config named exactly for what we were looking for
	// it's probably _very_ deliberate; we will return (nil, true).
	if len(parts) == 1 {
		return childConfig, true
	}
	if childConfig == nil {
		return nil, false
	}

	return configForLabel(*childConfig, strings.Join(parts[1:], "."))
}

func (logger *logFormatter) LoadConfig(config LogConfig) error {
	var level int
	if nil != logger.parent {
		log.Println(
			fmt.Sprintf(
				"WARNING (Deprecated): it is not advised to call LoadConfig() on child logger `%s`",
				logger.label,
			),
		)
	}

	// Set the config first
	logger.logConfig = config

	// NOTE: LoadConfig() has some surprising behavior. If you load a config
	// (as JSON) that looks like:
	// { "level": "ERROR",
	//   "loggers": {
	//     "main": {
	//       "level": "INFO",
	//       "loggers": {
	//         "test": {
	//           "level": "FATAL"
	//         }
	//       }
	//     }
	//   }
	// }
	// and your logger's label is "main", it's log level should be INFO, but
	// any loggers created with logger.New() - since they don't have an entry
	// under "loggers" - should get the root log level of ERROR.

	if namedConfig, ok := findConfig(logger, logger.label, logger.isSubLabledLogger); ok {
		// If we've got it in the namedConfig, assign the log level explicitly
		level = labelLevel(*namedConfig.Level)
	} else if config.Level != nil {
		// If we've got it, assign the log level explicitly
		level = labelLevel(*config.Level)
	} else if nil != logger.parent {
		// Default to the parent's log level if we have not set one
		// Not passing in a log level means "reset to default"
		level = logger.parent.level
	} else {
		// If we also don't have a parent, use the default of INFO
		// Not passing in a log level means "reset to default"
		level = INFO
	}

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

func (logger *logFormatter) EnvPrefixConfig(prefix string) error {
	cfg := make(map[string]interface{})

	for _, envpair := range os.Environ() {
		fullprefix := fmt.Sprintf("%s_", prefix)
		if strings.HasPrefix(envpair, fullprefix) {
			envsplit := strings.Split(envpair, "=")
			envname, envvalue := envsplit[0], envsplit[1]

			envkeys := strings.Split(strings.TrimPrefix(envname, fullprefix), "__")
			lvlCfg := cfg
			for i, k := range envkeys {
				// Convert k from ENV_CASE to camelCase
				key := strings.Replace(
					strings.Join(
						strings.Split(
							strings.Title(
								strings.ToLower(
									strings.ReplaceAll(k, "_", " "),
								),
							),
							" ",
						),
						"",
					),
					string(k[0]),
					strings.ToLower(string(k[0])),
					1,
				)

				if i == len(envkeys)-1 {
					// Set the value
					// Parse things that look like JSON
					if []rune(envvalue)[0] == []rune("{")[0] {
						v := make(map[string]interface{})
						err := json.Unmarshal([]byte(envvalue), &v)
						if err == nil {
							lvlCfg[key] = v
							continue
						}
						log.Println(fmt.Sprintf("Unable to parse %s as JSON. %s", envname, err))
					}

					// Fallback to just setting the value
					lvlCfg[key] = envvalue
				} else {
					// descend in to the child object
					if _, ok := lvlCfg[key]; !ok {
						lvlCfg[key] = make(map[string]interface{})
					}
					lvlCfg = lvlCfg[key].(map[string]interface{})
				}
			}
		}
	}

	config, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	log.Println(fmt.Sprintf("JSON config from Env: %s", config))

	return logger.JsonConfig(config)
}

// New has a confusing API because it allows setting the log level
// and the log config - which may contradict one another.
func New(label string, level int, logConfig *LogConfig) *logFormatter {
	if logConfig == nil {
		configLevel := levelLabel(level)
		logConfig = &LogConfig{
			Level: &configLevel,
		}
	}

	logger := &logFormatter{
		parent: nil,
		label:  label,
		level:  level,
	}

	logger.LoadConfig(*logConfig)
	if logger.level != level {
		log.Println(
			fmt.Sprintf(
				"WARNING: level passed for logger `%s` directly does not match level in config that was also passed",
				logger.label,
			),
		)
	}

	return logger
}

func (logger *logFormatter) Level() int {
	return logger.level
}

func (logger *logFormatter) New(label string) Logger {
	config, ok := findConfig(logger, label, false)
	if nil == config {
		config = &LogConfig{}
	}

	var level int
	if ok && config.Level != nil {
		level = labelLevel(*config.Level)
	} else {
		level = logger.level
	}

	return &logFormatter{
		parent:    logger,
		label:     label,
		level:     level,
		logConfig: *config,
	}
}

func (logger *logFormatter) Method(methodName string) Logger {
	config, ok := findConfig(logger, methodName, true)
	if nil == config {
		config = &LogConfig{}
	}

	var level int
	if ok && config.Level != nil {
		level = labelLevel(*config.Level)
	} else {
		level = logger.level
	}

	return &logFormatter{
		parent:            logger,
		isSubLabledLogger: true,
		label:             fmt.Sprintf("%s.%s", logger.label, methodName),
		level:             level,
		logConfig:         *config,
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
	return "\x1b[90;1m" + fmt.Sprintf(format, args...) + "\033[0m"
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
