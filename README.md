# Go Logs Go

A leveled logger for go with targeted configuration.

## Installation

As outlined in [https://blog.golang.org/using-go-modules](https://blog.golang.org/using-go-modules) add

```go
import (
  ...
  "github.com/big-squid/go-logs-go"
  ...
)
```
to your project and run `go mod tidy` or a build.

## Usage

###### Hard-coded Log Configuration

```go
package main

import (
  logs "github.com/big-squid/go-logs-go"
)

var logger logs.Logger

func init() {
  logger = logs.New(logs.RootLogConfig{
		Label: "main",
    Level: logs.Debug,
    Loggers: {
      "counter": &LogConfig{
        Level: logs.Info
      },
    }
	}
}

func counter() {
  log := logger.ChildLogger("counter")
  for i := 0; i < 5; i++ {
    log.Warn("Loop %v", i)
  }
  log.Fatal("done")
}

func main() {
  log := logger
  log.Trace("This will only appear if the level is TRACE")
  log.Debug("Debug msg")
  log.Info("Info msg")

  counter()
}
```

This example demonstrates the basic usage of a `go-logs-go` logger:

1. A root Logger is created using `logs.New()`
2. `logs.New()` takes a `*RootLogConfig{}` that specifies
  + a log `Level` indicating which log messages should be written to the logs
  + an optional `Label` to include in all log messages
  + optional child logger configuration `Loggers` in a `map[string]*LogConfig`
3. Child loggers may be created using `logger.ChildLogger()`, which requires a name. The name will be used to:
  + create a label, by appending it to the parent logger's label
  + find the child logger's configuration in it's parent logger's `Logger's` map. The logger's `Level` _may_ be supplied in this configuration. If not, the parent logger's level will be used.
4. Loggers export log level functions for logging at a particular level. Log level functions exist for `Trace()`, `Debug()`, `Info()`, `Warn()`, and `Error()`. Each of these will generate a log message at the log level that matches their name _if_ the logger's level is less than or equal to that level.

### Config-only Log Levels

Astute observes will notice 3 `LogLevel` constants that do not map to a log level function. These are use only for configuration. They are:

1. `logs.All` - this indicates that **all** log messages should be written to the logs
2. `logs.Off` - this indicates that **no** log messages should be written to the logs
3. `logs.NotSet` - this indicates that a log level has not been set for a given logger and the logger should inherit it's parent's log level or use the default `logs.Info` if no parent exists. This log level is the "zero value" for the `LogLevel` constants.

### Ways to get a RootLogConfig

It's very unlikely that you actually want to hard code your log configuration. `go-logging` provides several methods for retrieving a configuration from outside the code. Log levels should be set using the case insensitive string equivalent of the constant name.

#### JsonConfig

```go
cfg, err := logging.JsonConfig([]byte(`
	{ "level": "ERROR",
      "loggers": {
        "main": {
          "level": "INFO",
          "loggers": {
            "child": {
              "level": "DEBUG"
            }
          }
        }
      }
    }
`))

if nil != err {
  panic(err)
}

logger := logs.New(cfg)
```

`JsonConfig()` takes JSON as a `[]byte` and Marshalls it in to a `*RootLogConfig`. It is used by the other configuration functions.

#### FileConfig

`FileConfig()` reads a file path and creates a `*RootLogConfig{}` from it's json data

```go
cfg, err := logs.FileConfig("./log-config.json")
if nil != err {
  panic(err)
}

logger := logs.New(cfg)
```

#### PathEnvConfig
`PathEnvConfig()` gets a file path from the specified environment variable, reads it's contents and creates a `*RootLogConfig` from it's json data

```go
cfg, err := logs.PathEnvConfig("LOG_CONFIG_PATH")
if nil != err {
  panic(err)
}

logger := logs.New(cfg)
```

#### EnvPrefixConfig
`EnvPrefixConfig()` finds all of the environment variables that start with a specified prefix and uses them to build a `*RootLogConfig{}`. After the prefix, a single underscore (`"_"`) is treated as a word separator. Two successive underscores (`"__"`) are treated as a struct separator - the left side is the name of the parent struct, the right is a field name. Environment variables that appear to be JSON (because they start with a curly brace - `"{"`) will attempt to be parsed as JSON.

```go
// LOG_CONFIG_LEVEL="ERROR"
// LOG_CONFIG_LABEL="_root_"
// LOG_CONFIG_LOGGERS__CHILD__LEVEL="DEBUG"
// LOG_CONFIG_LOGGERS__CHILD__GRANDCHILD__LEVEL="TRACE"
// LOG_CONFIG_LOGGERS__JSON_CHILD="{\"level\": \"WARN\", \"loggers\": {\"grandchild\": {\"level\": \"ERROR\"}}}"
  
cfg, err := logs.EnvPrefixConfig("LOG_CONFIG")
if nil != err {
  panic(err)
}

logger := logs.New(cfg)
```

### Advanced Usage

It is possible to further customize the logs written by a `go-logs-go` logger as well as where and how they are written by specifying a `LogHandler` function. For now, interested parties should review the implementation of the `DefaultLogHandler` in the source code.

#### Shhh! Don't tell anyone!

Of particular note, the `DefaultLogHandler` uses [`log.PrintLn()`](https://golang.org/pkg/log/#Println) to write log messages. (This is how our log output contains timestamps without them being included in the `LogMessage` struct.) Using the base `"log"` package in this way allows the user to make use of [`log.SetOutput()`](https://golang.org/pkg/log/#SetOutput) and [`log.SetFlags()`](https://golang.org/pkg/log/#SetFlags) if desired - though this is considered a private implementation detail. Users who want backward compatibility guarantees should implement their own `LogHandler` instead.
