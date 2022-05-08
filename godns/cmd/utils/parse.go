package utils

import (
	"flag"
)

func GetConfigPath() string {
	var configPath string
	flag.StringVar(&configPath, "c", "../config/conf.yaml", "path to yaml config file")

	flag.Parse()
	return configPath
}
