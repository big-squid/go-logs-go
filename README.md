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

## Examples

```go
package main

import (
  "github.com/big-squid/go-logging"
)

var logger logging.Logger

func init() {
  logger = logging.New("main", logging.DEBUG, nil)
  logger.EnvConfig("LOG_CONFIG")
}

func counter() {
  log := logger.New("counter")
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
