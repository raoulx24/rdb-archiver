package snapshot

import (
	"os"
	"path/filepath"
)

// detect builds and enqueues a snapshot if the primary file changed.
func (w *Watcher) detect() {
	w.log.Debug("entering Worker.Handle()")
	w.mu.RLock()
	dir := w.dir
	primary := w.primaryName
	aux := append([]string(nil), w.auxNames...)
	last := w.lastModTime
	w.mu.RUnlock()

	//if !w.isPrimaryFileStable() {
	//	return
	//}

	path := filepath.Join(dir, primary)

	info, err := os.Stat(path)
	if err != nil {
		return
	}

	mod := info.ModTime()
	if !mod.After(last) {
		return
	}

	snap := Snapshot{
		Dir:     dir,
		Primary: FromFileInfo(path, info),
		Aux:     w.loadAux(dir, aux),
	}

	w.mu.Lock()
	w.lastModTime = mod
	w.mu.Unlock()

	w.log.Debug("mama", "cucu", snap)

	w.mb.Put(Job{Snap: snap})
}
