package utils

import (
	"context"
	"fmt"
	"os"
	"os/signal"
)

func StartShutdownHandler(shutdown context.CancelFunc) {
	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, os.Interrupt)
	for {
		sig := <-signalChan
		switch sig {
		case os.Interrupt:
			fmt.Print("\033[32mStarted graceful shutdown...\n\033[0m")
			shutdown()
			return
		}
	}
}
