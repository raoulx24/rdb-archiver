// Package fsprobe checks whether fsnotify works reliably for a directory.
// It performs a real create+rename test to ensure events are delivered.
package fsprobe

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Result reports whether fsnotify is usable and why.
type Result struct {
	FsnotifySupported bool   // true if events are delivered
	Reason            string // explanation when unsupported
}

// Probe tests whether fsnotify reliably reports rename events in dir.
func Probe(dir string) Result {
	st, err := os.Stat(dir)
	if err != nil {
		return Result{false, fmt.Sprintf("stat failed: %v", err)}
	}
	if !st.IsDir() {
		return Result{false, "not a directory"}
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return Result{false, fmt.Sprintf("fsnotify unavailable: %v", err)}
	}
	defer w.Close()

	if err := w.Add(dir); err != nil {
		return Result{false, fmt.Sprintf("cannot watch directory: %v", err)}
	}

	tmp := filepath.Join(dir, ".fsprobe_tmp")
	final := filepath.Join(dir, ".fsprobe_final")

	// Create temp file.
	if f, err := os.Create(tmp); err == nil {
		f.Close()
	} else {
		return Result{false, fmt.Sprintf("cannot create temp file: %v", err)}
	}

	// Rename temp → final to trigger a rename event.
	if err := os.Rename(tmp, final); err != nil {
		os.Remove(tmp)
		return Result{false, fmt.Sprintf("rename failed: %v", err)}
	}
	defer os.Remove(final)

	// Wait briefly for events.
	timeout := time.After(200 * time.Millisecond)
	for {
		select {
		case ev := <-w.Events:
			if ev.Op&(fsnotify.Rename|fsnotify.Create|fsnotify.Write) != 0 {
				return Result{true, ""}
			}
		case <-timeout:
			return Result{false, "no events received (rename not reported)"}
		}
	}
}
