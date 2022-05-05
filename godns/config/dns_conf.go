package config

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type ConfigInstance struct {
	Nameservers    []string `yaml:"nameservers"`
	Host           string   `yaml:"host"`
	Port           int      `yaml:"port"`
	UpdateLivetime bool     `yaml:"update-in-livetime"`
}

func (conf *ConfigInstance) parse(data []byte) error {
	return yaml.Unmarshal(data, conf)
}

func UseConfig(configPath *string) *ConfigInstance {

	if _, err := os.Stat(*configPath); err != nil {
		fmt.Print("\033[31mCan't find config file!\033[0m")
		os.Exit(0)
	}

	data, err := ioutil.ReadFile(*configPath)
	var configInst *ConfigInstance = &ConfigInstance{}

	if err != nil {
		fmt.Print("\033[31mConfig file is not a correct YAML file!\033[0m")
		os.Exit(0)
	}

	if err := configInst.parse(data); err != nil {
		fmt.Print("\033[31mError occured during parsing YAML config\033[0m")
		os.Exit(0)
	}

	if configInst.UpdateLivetime {
		go CreateWatcher(configPath, configInst)
	}

	return configInst
}
