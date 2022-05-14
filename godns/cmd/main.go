package main

import (
	"context"
	"fmt"
	. "godns/cache"
	. "godns/cmd/utils"
	. "godns/config"
	. "godns/server"
	"time"
)

func main() {

	mainContext, shutdown := context.WithCancel(context.Background())

	go StartShutdownHandler(shutdown)

	var configPath string = GetConfigPath()

	var configHandler *ConfigHandler = NewConfigHandler(configPath, mainContext)
	var cache *Cache = NewCache(time.Minute*configHandler.Get().CacheExpiration,
		time.Minute*configHandler.Get().CacheCleanup)

	fmt.Println(time.Minute * configHandler.Get().CacheExpiration)

	go StartServer(configHandler, mainContext, cache)

	for {
		select {
		case <-mainContext.Done():
			time.Sleep(time.Second * 2)
			fmt.Print("\033[32mDNS Server was stopped\n\033[0m")
			return

		default:
			if configHandler.NeedRestart {
				time.Sleep(time.Second * 2)
				fmt.Print("\033[36mServer is now using updated config\n\033[0m")
				configHandler.NeedRestart = false
				go StartServer(configHandler, mainContext, cache)
			}
			time.Sleep(time.Second)
		}
	}
}
