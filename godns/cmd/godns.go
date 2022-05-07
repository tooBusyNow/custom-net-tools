package main

import (
	"context"
	"fmt"
	"godns/cmd/utils"
	"godns/config"
	"time"
)

func main() {

	ctx, shutdown := context.WithCancel(context.Background())
	go utils.HandleSignals(shutdown)

	var configPath string = utils.GetConfigPath()
	var configLoader *config.ConfigHandler = config.NewConfigHandler(configPath, ctx)

	fmt.Print("\033[32mDNS Server is up and running\n\033[0m")

	for {
		select {
		case <-ctx.Done():
			time.Sleep(time.Second * 2)
			fmt.Print("\033[32mDNS Server was stopped\n\033[0m")
			return

		default:
			fmt.Println(configLoader.Get())
			time.Sleep(time.Second)
		}
	}
}
