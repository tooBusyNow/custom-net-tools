package config

import (
	"fmt"
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
)

func CreateWatcher(configPath *string, configInst *ConfigInstance) {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	defer watcher.Close()

	if err != nil {
		fmt.Print("\033[31mCan't start new Watcher instance\033[0m")
		os.Exit(0)
	}

	watcher.Add(*configPath)

	for {
		select {

		case event := <-watcher.Events:

			if event.Op&fsnotify.Write == fsnotify.Write {
				fmt.Print("Config was updated, start reloading\n")
				UseConfig(configPath)
				fmt.Print("Finised reloading\n")
				return
			}

			if !configInst.UpdateLivetime {
				return
			}

		case err := <-watcher.Errors:
			log.Println("error:", err)

		}
	}
}
