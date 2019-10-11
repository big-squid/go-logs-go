package gologsgo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
)

var defaultLeveledLogHandler LeveledLogHandler
var LogLevels orderedLogLevels

func init() {
	LogLevels = orderedLogLevels{
		order: []LogLevel{
			All,
			Trace,
			Debug,
			Info,
			Warn,
			Error,
			Off,
		},
		labels: map[LogLevel]string{
			All:   "ALL",
			Trace: "TRACE",
			Debug: "DEBUG",
			Info:  "INFO",
			Warn:  "WARN",
			Error: "ERROR",
			Off:   "OFF",
		},
	}
	defaultLeveledLogHandler = LeveledLogHandler{
		Format:     "%s [%s]: %s",
		RootFormat: "%s: %s",
		Levels: map[LogLevel]Formatter{
			Trace: greyString,
			Debug: greyString,
			Info:  color.WhiteString,
			Warn:  color.YellowString,
			Error: color.RedString,
		},
	}
}

type LogLevel int

func (ll *LogLevel) UnmarshalJSON(b []byte) error {
	var i interface{}
	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}

	switch i.(type) {
	case int:
		ord := i.(int)
		if ord > 0 && ord < len(LogLevels.order) {
			*ll = LogLevel(ord)
		}
		return nil
	case string:
		label := strings.ToUpper(i.(string))
		level, ok := LogLevels.Level(label)
		if ok {
			*ll = level
			return nil
		}
	case nil:
		*ll = NotSet
	default:
		// Do nothing. We'll be returning an error
	}

	return fmt.Errorf("Invalid JSON value for LogLevel %s", i)
}

// Log Constants
// NotSet is literally our "zero value"
// NOTE: go does _not_ recommend using ALL_CAPS for constants, as these
// would always be exported (see https://stackoverflow.com/questions/22688906/go-naming-conventions-for-const)
const (
	NotSet LogLevel = iota
	All
	Trace
	Debug
	Info
	Warn
	Error
	Off
)

// log levels should have an inherent limited set and order that the set of all ints does not provide
// By declaring a struct we get:
//  - the order in the struct is the order of the level (assuming linting does not reorder)
//  - a namespace for methods
//  - a place to hide the memoization of ordinals
// We still don't get the type checking of the limited set of possible values
// that a proper enum would provide, but hopefully in practice that won't be too
// problematic.
type orderedLogLevels struct {
	order         []LogLevel
	labels        map[LogLevel]string
	indexCache    map[LogLevel]int
	ordinalsCache map[string]LogLevel
}

func (ll *orderedLogLevels) Label(level LogLevel) string {
	return ll.labels[level]
}

func (ll *orderedLogLevels) Level(label string) (LogLevel, bool) {
	if nil == ll.ordinalsCache {
		ll.ordinalsCache = make(map[string]LogLevel)
	}

	lvl, ok := ll.ordinalsCache[label]
	if ok {
		return lvl, true
	}

	for k, v := range ll.labels {
		if v == label {
			ll.ordinalsCache[label] = k
			return k, true
		}
	}

	return NotSet, false
}

func (ll *orderedLogLevels) Index(level LogLevel) (int, bool) {
	if nil == ll.indexCache {
		ll.indexCache = make(map[LogLevel]int)
	}

	i, ok := ll.indexCache[level]
	if ok {
		return i, true
	}

	for i, v := range ll.order {
		if v == level {
			ll.indexCache[level] = i
			return i, true
		}
	}

	return 0, false
}

func (ll *orderedLogLevels) Next(level LogLevel) (LogLevel, bool) {
	idx, ok := ll.Index(level)
	if ok {
		next := idx + 1
		if next < len(ll.order) {
			return ll.order[next], true
		}
	}
	return NotSet, false
}

func (ll *orderedLogLevels) Previous(level LogLevel) (LogLevel, bool) {
	idx, ok := ll.Index(level)
	if ok {
		prev := idx - 1
		if prev > 0 {
			return ll.order[prev], true
		}
	}
	return NotSet, false
}

// LogMessage structs will be passed by the logger to the configured LogHandler
// each time a logging function (`.Trace()`, `.Debug()`,`.Info()`,`.Warn()`,
// `.Error()`) is called.
type LogMessage struct {
	Level      LogLevel
	LevelLabel string
	Logger     string
	Message    string
}

// LogHandler receives a LogMessage and ensures it is properly written to the logs.
// Most consumers of this package will want to use the DefaultLogHandler to write
// color coded log messages to stdout with timestamps.
type LogHandler func(LogMessage)

// DefaultLogHandler is a LogHandler that writes color coded log messages to stdout with
// UTC timestamps.
func DefaultLogHandler(msg LogMessage) {
	defaultLeveledLogHandler.LogHandler(msg)
}

type Formatter func(string, ...interface{}) string

type LeveledLogHandler struct {
	Format     string
	RootFormat string
	Levels     map[LogLevel]Formatter
}

func (h *LeveledLogHandler) LogHandler(msg LogMessage) {
	var levelFn Formatter
	lvl := msg.Level
	for {
		levelFn := h.Levels[lvl]
		if nil != levelFn {
			break
		}
		prev, ok := LogLevels.Previous(lvl)
		if !ok {
			break
		}
		lvl = prev
	}

	if nil == levelFn {
		// TODO: find the Formatter for the next lower log level
		levelFn = fmt.Sprintf
	}

	if len(h.RootFormat) > 0 && len(msg.Logger) == 0 {
		log.Println(levelFn(
			h.RootFormat,
			strings.ToUpper(msg.LevelLabel),
			msg.Message,
		))
		return
	}

	log.Println(levelFn(
		h.Format,
		strings.ToUpper(msg.LevelLabel),
		msg.Logger,
		msg.Message,
	))
}

// greyString is a private method supporting the DefaultLogHandler
func greyString(format string, args ...interface{}) string {
	return "\x1b[90;1m" + fmt.Sprintf(format, args...) + "\033[0m"
}

type RootLogConfig struct {
	Loggers map[string]*LogConfig `json:"loggers"`
	Level   LogLevel              `json:"level"`
	Label   string                `json:"label"`
	// Don't try to Marshall/Unmarshall a function
	LogHandler LogHandler `json:"-"`
}

type LogConfig struct {
	Loggers map[string]*LogConfig `json:"loggers"`
	Level   LogLevel              `json:"level"`
}

// JsonConfig creates a RootLogConfig from JSON data
func JsonConfig(data []byte) (*RootLogConfig, error) {
	config := RootLogConfig{}
	err := json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// FileConfig reads a file path and creates a RootLogConfig from it's JSON data
func FileConfig(configFile string) (*RootLogConfig, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	return JsonConfig(data)
}

// PathEnvConfig gets a file path from the specified environment variable, reads it's contents
// and creates a RootLogConfig from it's JSON data
func PathEnvConfig(env string) (*RootLogConfig, error) {
	return FileConfig(os.Getenv(env))
}

// EnvPrefixConfig finds all the environment variables that start with a specified prefix
// and uses them to build a RootLogConfig. After the prefix, a single underscore ("_")
// is treated as a word seperator. Two successive underscores ("__") are treated as
// a struct seperator - the left side is the parent struct, the right is a field name.
func EnvPrefixConfig(prefix string) (*RootLogConfig, error) {
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
		return nil, err
	}

	return JsonConfig(config)
}

// TODO: Implement a NamedConfig method that takes defaults, searches for files in the
// current working directory, etc/, and ~/, uses environment vairables, and parses CLI args

// TODO: Implement an optional channel as part of the RootLogConfig on which to receive updated
// RootLogConfig instances so log levels can be updated via Redis or some other means that
// didn't entail a restart. This enables turning on debug or trace level logging for a code path
// that is exhibiting errors.

// TODO: implement a PackageLogger interface that allows targetting a logger in configuration by package name

// Logger is the primary structure in this package. It supplies the log level functions.
// A Logger only has a `parent` if it was created by Logger.ChildLogger(). If so, it's
// `logConfig` will be a reference to it's config from the parent - the only place it
// can get a config.
type Logger struct {
	parent     *Logger
	logConfig  *LogConfig
	label      string
	logHandler LogHandler
}

// New returns a new root Logger
func New(logConfig *RootLogConfig) *Logger {
	if logConfig == nil {
		logConfig = &RootLogConfig{}
	}

	if logConfig.Level == NotSet {
		// Default to the INFO log level
		logConfig.Level = Info
	}

	if len(logConfig.Label) < 1 {
		// Explicitly default the Label to the empty string
		logConfig.Label = ""
	}

	if logConfig.LogHandler == nil {
		// Default to the INFO log level
		logConfig.LogHandler = DefaultLogHandler
	}

	logger := &Logger{
		parent: nil,
		logConfig: &LogConfig{
			Loggers: logConfig.Loggers,
			Level:   logConfig.Level,
		},
		label:      logConfig.Label,
		logHandler: logConfig.LogHandler,
	}

	return logger
}

func (logger *Logger) Level() LogLevel {
	return logger.logConfig.Level
}

func (logger *Logger) Label() string {
	return logger.label
}

func (logger *Logger) ChildLogger(name string) *Logger {
	if len(name) < 1 {
		panic(fmt.Errorf("Child loggers require a name"))
	}

	if strings.Contains(name, ".") {
		panic(fmt.Errorf("Child logger name should not contain '.'"))
	}

	config, ok := logger.logConfig.Loggers[name]
	if !ok || nil == config {
		config = &LogConfig{}
	}

	if config.Level == NotSet {
		config.Level = logger.logConfig.Level
	}

	parts := []string{}
	if len(logger.label) > 1 {
		parts = append(parts, logger.label)
	}
	parts = append(parts, name)
	label := strings.Join(parts, ".")

	return &Logger{
		parent:     logger,
		logConfig:  config,
		label:      label,
		logHandler: logger.logHandler,
	}
}

// log is a private method that supports all of the exported log level
// methods
func (logger *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < logger.Level() {
		return
	}

	msg := fmt.Sprintf(format, args...)
	logger.logHandler(LogMessage{
		Level:      level,
		LevelLabel: LogLevels.Label(level),
		Logger:     logger.Label(),
		Message:    msg,
	})
}

// Trace logs a message at the TRACE level
func (logger *Logger) Trace(format string, args ...interface{}) {
	logger.log(Trace, format, args...)
}

// Debug logs a message at the DEBUG level
func (logger *Logger) Debug(format string, args ...interface{}) {
	logger.log(Debug, format, args...)
}

// Info logs a message at the INFO level
func (logger *Logger) Info(format string, args ...interface{}) {
	logger.log(Info, format, args...)
}

// Warn logs a message at the WARN level
func (logger *Logger) Warn(format string, args ...interface{}) {
	logger.log(Warn, format, args...)
}

// Error logs a message at the ERROR level
func (logger *Logger) Error(format string, args ...interface{}) {
	logger.log(Error, format, args...)
}
