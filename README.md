## Installation

As outlined in [https://blog.golang.org/using-go-modules](https://blog.golang.org/using-go-modules) add

```go
import (
  ...
  "github.com/big-squid/go-logging"
  ...
)
```
to your project and run `go mod tidy` or a build.

## Usage

###### Hard-coded Log Configuration

```go
package main

import (
  logging "github.com/big-squid/go-logging"
)

var logger logging.Logger

func init() {
  logger = logging.New(logging.RootLogConfig{
		Label: "main",
    Level: logging.Debug,
    Loggers: {
      "counter": &LogConfig{
        Level: logging.Info
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

This example demonstrates the basic usage of a `go-logging` logger:

1. A Logger is created using `logger.New()`
2. `logger.New()` takes a `*RootLogConfig{}` that specifies
  + a log `Level` indicating which log messages should be written to the logs
  + an optional `Label` to include in all log messages
  + optional child logger configuration `Loggers` in a `map[string]*LogConfig`
3. Child loggers may be created using `logger.ChildLogger()`, which requires a name. The name will be used to:
  + create a label, by appending it to the parent logger's label
  + find the child logger's configuration in it's parent logger's `Logger's` map. The logger's `Level` _may_ be supplied in this configuration. If not, the parent logger's level will be used.
4. Loggers export log level functions for logging at a particular level. Log level functions exist for `Trace()`, `Debug()`, `Info()`, `Warn()`, `Error()`, and `Fatal()`. Each of these will generate a log message at the log level that matches their name _if_ the logger's level is less than or equal to that level.

**NOTE**: There is no exported generic `Log()` method. You must use a log level.
**NOTE**: `Fatal()` will call `panic("FATAL")` after logging it's message. This is an extreme measure - though it can be caught and does allow deferred code to run. It's use is generally discouraged.

### Ways to get a RootLogConfig

It's very unlikely that you actually want to hard code your log configuration. `go-logging` provides several methods for retrieving a configuration from outside the code:

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

logger := logging.New(cfg)
```

`JsonConfig()` takes JSON as a `[]byte` and Marshalls it in to a `*RootLogConfig`. It is used by the other configuration functions.

#### FileConfig

`FileConfig()` reads a file path and creates a `*RootLogConfig{}` from it's json data

```go
cfg, err := logging.FileConfig("./log-config.json")
if nil != err {
  panic(err)
}

logger := logging.New(cfg)
```

#### PathEnvConfig
`PathEnvConfig()` gets a file path from the specified environment variable, reads it's contents and creates a `*RootLogConfig` from it's json data

```go
cfg, err := logging.PathEnvConfig("LOG_CONFIG_PATH")
if nil != err {
  panic(err)
}

logger := logging.New(cfg)
```

#### EnvPrefixConfig
`EnvPrefixConfig()` finds all of the environment variables that start with a specified prefix and uses them to build a `*RootLogConfig{}`. After the prefix, a single underscore (`"_"`) is treated as a word separator. Two successive underscores (`"__"`) are treated as a struct separator - the left side is the name of the parent struct, the right is a field name. Environment variables that appear to be JSON (because they start with a curly brace - `"{"`) will attempt to be parsed as JSON.

```go
// LOG_CONFIG_LEVEL="ERROR"
// LOG_CONFIG_LABEL="_root_"
// LOG_CONFIG_LOGGERS__CHILD__LEVEL="DEBUG"
// LOG_CONFIG_LOGGERS__CHILD__GRANDCHILD__LEVEL="TRACE"
// LOG_CONFIG_LOGGERS__JSON_CHILD="{\"level\": \"WARN\", \"loggers\": {\"grandchild\": {\"level\": \"ERROR\"}}}"
  
cfg, err := logging.EnvPrefixConfig("LOG_CONFIG")
if nil != err {
  panic(err)
}

logger := logging.New(cfg)
```

### Advanced Usage

It is possible to further customize the logs written by a `go-logging` logger as well as where and how they are written by specifying a `LogHandler` function. For now, interested parties should review the implementation of the `DefaultLogHandler` in the source code.

#### Shhh! Don't tell anyone!

Of particular note, the `DefaultLogHandler` uses [`log.PrintLn()`](https://golang.org/pkg/log/#Println) to write log messages. (This is how our log output contains timestamps without them being included in the `LogMessage` struct.) Using the base `"log"` package in this way allows the user to make use of [`log.SetOutput()`](https://golang.org/pkg/log/#SetOutput) and [`log.SetFlags()`](https://golang.org/pkg/log/#SetFlags) if desired - though this is considered a private implementation detail. Users who want backward compatibility guarantees should implement their own LogHandler instead.
