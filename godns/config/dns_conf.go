package config

import (
	"context"
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

func (conf *ConfigInstance) parseConfig(data []byte) error {
	return yaml.Unmarshal(data, conf)
}

func LoadDNSConfig(configPath *string, updateChan chan bool,
	doneChan chan bool, ctx context.Context) *ConfigInstance {

	var configInst *ConfigInstance = &ConfigInstance{}

	if configInst, err := getValidatedConfig(configPath, configInst); err == nil {
		if configInst.UpdateLivetime {
			go CreateWatcher(configPath, configInst, updateChan, doneChan, ctx)
		}
		return configInst
	} else {
		os.Exit(0)
	}
	return nil
}

func ReloadDNSConfig(configPath *string, configInst *ConfigInstance) (*ConfigInstance, error) {
	var err error
	configInst, err = getValidatedConfig(configPath, configInst)
	if err != nil {
		return nil, err
	}
	return configInst, nil
}

func getValidatedConfig(configPath *string, configInst *ConfigInstance) (*ConfigInstance, error) {

	if _, err := os.Stat(*configPath); err != nil {
		fmt.Print("\033[31mCan't find config file!\033[0m")
		return nil, err
	}

	data, err := ioutil.ReadFile(*configPath)

	if err != nil {
		fmt.Print("\033[31mConfig file is not a correct YAML file!\033[0m")
		return nil, err
	}

	if err := configInst.parseConfig(data); err != nil {
		fmt.Print("\033[31mError occured during parsing YAML config\033[0m")
		return nil, err
	}

	return configInst, nil
}
