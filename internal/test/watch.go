package test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

var watchDirs = []string{"tests", "compositions", "functions", "crds"}

// Watch calls runFn immediately, then re-calls it with the changed file path
// whenever any file under the watched directories changes (debounced 300ms).
// It blocks until the process is interrupted.
func Watch(runFn func(changedFile string), out io.Writer) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer watcher.Close()

	for _, d := range watchDirs {
		if err := watchRecursive(watcher, d); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("watch %s: %w", d, err)
		}
	}

	runFn("")
	fmt.Fprintln(out, "\nWatching for changes… (Ctrl+C to stop)")

	var debounce <-chan time.Time
	var changedFile string

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) || event.Has(fsnotify.Remove) {
				changedFile = event.Name
				debounce = time.After(300 * time.Millisecond)
				if event.Has(fsnotify.Create) {
					if fi, err := os.Stat(event.Name); err == nil && fi.IsDir() {
						_ = watchRecursive(watcher, event.Name)
					}
				}
			}
		case <-debounce:
			clearScreen(out)
			runFn(changedFile)
			fmt.Fprintln(out, "\nWatching for changes… (Ctrl+C to stop)")
			debounce = nil
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(out, "watcher error: %v\n", err)
		}
	}
}

func watchRecursive(watcher *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
}

func clearScreen(out io.Writer) {
	fmt.Fprint(out, "\033[2J\033[H")
}
