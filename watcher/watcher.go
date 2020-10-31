package watcher

import (
	"fmt"
	"github.com/radovskyb/watcher"
	"harl/model"
	"harl/utils"
	"log"
	"path"
	"strings"
	"time"
)

type FileWatch struct {
	W      *watcher.Watcher
	Event  chan model.Envent
	config *model.Watch
}

func New(config *model.Watch) *FileWatch {
	return &FileWatch{
		W:      watcher.New(),
		config: config,
		Event:  make(chan model.Envent),
	}
}

func (fw *FileWatch) Watcher() {
	w := watcher.New()
	defer w.Close()
	w.FilterOps(watcher.Write, watcher.Remove)
	go func() {
		for {
			select {
			case event := <-w.Event:
				fmt.Println(event) // Print the event's info.
				if event.IsDir() {
					continue
				}
				if !utils.IncludesString(fw.config.Includes, path.Ext(event.Path)) {
					continue
				}
				if strings.HasSuffix(event.Path, ".hap") {
					fw.Event <- model.Envent{
						Action: "reload",
						Data:   map[string]string{"bin": event.Path},
					}
				} else {
					fw.Event <- model.Envent{
						Action: "build",
						Data:   nil,
					}
				}

			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()
	// Watch test_folder recursively for changes.
	if err := w.AddRecursive(fw.config.Project); err != nil {
		log.Fatalln(err)
	}

	for _, exclude := range fw.config.Excludes {
		err := w.Ignore(fmt.Sprintf("%s/%s", fw.config.Project, exclude))
		if err != nil {
			fmt.Errorf("exclude files error: %w", err)
		}
	}

	w.IgnoreHiddenFiles(true)

	delay, _ := time.ParseDuration(fmt.Sprintf("%dms", fw.config.Delay))
	// Start the watching process - it'll check for changes every 100ms.
	if err := w.Start(delay); err != nil {
		log.Fatalln(err)
	}
}
