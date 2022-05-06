package main

import (
	"context"
	"fmt"
	"godns/cmd/utils"
	"godns/config"
)

func main() {

	updateChan := make(chan bool, 1)
	doneChan := make(chan bool)

	ctx, shutdown := context.WithCancel(context.Background())

	configPath := utils.GetConfigPath()
	configInst := config.LoadDNSConfig(configPath, updateChan, doneChan, ctx)

	fmt.Print("\033[32mDNS Server is up and running\n\033[0m")

	if configInst.UpdateLivetime {
		go func() {
			for {
				select {

				case <-updateChan:
					newConfigInst, err := config.ReloadDNSConfig(configPath, configInst)
					configInst = newConfigInst

					if err != nil {
						shutdown()
						return
					}
					fmt.Print("Finished ", configInst)

				case <-doneChan:
					return
				}
			}
		}()
	}
	<-doneChan
}
