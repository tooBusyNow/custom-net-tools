package main

import (
	"fmt"
	"godns/cmd/utils"
	"godns/config"
)

func main() {

	configPath := utils.GetConfigPath()
	config := config.UseConfig(configPath)

	fmt.Print("\033[32mDNS Server is up and running\n\033[0m")
	fmt.Printf("%+v\n", config)

	for {

	}
}
