package config

import (
	"context"
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v2"
)

type ConfigInstance struct {
	Nameservers    []string `yaml:"nameservers"`
	Host           string   `yaml:"host"`
	Port           int      `yaml:"port"`
	UpdateLivetime bool     `yaml:"update-in-livetime"`
}

type ConfigHandler struct {
	configPath string
	configInst *ConfigInstance
	mu         sync.RWMutex
}

func NewConfigHandler(path string, ctx context.Context) *ConfigHandler {
	handler := &ConfigHandler{configPath: path}
	err := handler.Load(ctx)
	if err != nil {
		fmt.Print("\033[31mCan't create config loader due to internal error\033[0m")
		os.Exit(0)
	}
	return handler
}

func (handler *ConfigHandler) Load(ctx context.Context) error {
	config, err := loadConfigFile(handler, ctx)
	if err != nil {
		return err
	}
	handler.mu.Lock()
	handler.configInst = config
	handler.mu.Unlock()

	if config.UpdateLivetime {
		go watchFile(handler, ctx)
	}
	return nil
}

func (handler *ConfigHandler) Reload(ctx context.Context) error {

	newConfig, err := loadConfigFile(handler, ctx)
	if err != nil {
		return err
	}

	handler.mu.Lock()
	handler.configInst = newConfig
	handler.mu.Unlock()

	return nil
}

func (handler *ConfigHandler) Get() *ConfigInstance {
	handler.mu.RLock()
	defer handler.mu.RUnlock()
	return handler.configInst
}

func loadConfigFile(handler *ConfigHandler, ctx context.Context) (*ConfigInstance, error) {
	data, err := os.ReadFile(handler.configPath)
	config := ConfigInstance{}

	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &config)

	if err != nil {
		return nil, err
	}
	return &config, nil
}
