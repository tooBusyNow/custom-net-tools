package config

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
)

func CreateWatcher(configPath *string, configInst *ConfigInstance,
	updateChan chan bool, doneChan chan bool, ctx context.Context) {

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

		case <-ctx.Done():
			fmt.Print("Shutdown\n")
			doneChan <- true
			return

		case event := <-watcher.Events:

			if event.Op&fsnotify.Write == fsnotify.Write {
				fmt.Print("Config was updated, start reloading\n")
				updateChan <- true
			}
			if !configInst.UpdateLivetime {
				return
			}

		case err := <-watcher.Errors:
			log.Println("error:", err)
		}

	}
}
