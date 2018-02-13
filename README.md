## Installation

`gb vendor fetch github.com/big-squid/go-logging`

## Examples

```go
package main

import (
  "github.com/big-squid/logging"
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
