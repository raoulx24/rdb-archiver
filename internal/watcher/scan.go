package watcher

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"
)

// contains the directory scanning logic that detects new or updated RDB files.

func (w *Watcher) scan(ctx context.Context, seen map[string]time.Time) {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		log.Printf("watcher: failed to read dir %s: %v", w.dir, err)
		return
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := e.Name()
		if filepath.Ext(name) != ".rdb" {
			continue
		}

		full := filepath.Join(w.dir, name)

		info, err := os.Stat(full)
		if err != nil {
			log.Printf("watcher: stat failed for %s: %v", full, err)
			continue
		}

		mod := info.ModTime()
		last, ok := seen[full]

		if !ok || mod.After(last) {
			seen[full] = mod
			w.enqueue(ctx, full, mod)
		}
	}
}
