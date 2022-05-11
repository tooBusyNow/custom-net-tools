package config

import (
	"context"
	"fmt"
	"os"
	"time"
)

func watchFile(configLoader *ConfigHandler, ctx context.Context) error {
	initialStat, err := os.Stat(configLoader.configPath)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			stat, err := os.Stat(configLoader.configPath)
			if err != nil {
				return err
			}
			if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
				fmt.Print("\033[36mConfig file was changed\n\033[0m")
				initialStat = stat
				configLoader.Reload(ctx)
			}
			time.Sleep(time.Second / 2)
		}
	}
}
