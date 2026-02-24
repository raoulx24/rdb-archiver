## Worker should not return only the RDB path
   Right now:

```go
return finalPath, nil
```
But the worker is conceptually producing a snapshot event, not a single file.

Even if you postpone copying nodes.conf, the worker should return a struct, not a string.

Clean future‑proof shape:
```go
type SnapshotResult struct {
Timestamp time.Time
RDBPath   string
NodesPath string // empty for now
}
```
Then:

```go
return SnapshotResult{Timestamp: ts, RDBPath: finalPath}, nil
```
This makes the worker’s output meaningful and avoids future refactors.

## Worker should not parse timestamps inline
Even if you delay nodes.conf, timestamp extraction should be centralized.

Right now, timestamp parsing is scattered across retention and worker code.
Move it into a helper like:

```go
func ParseSnapshotTimestamp(path string) (time.Time, error)
```
This keeps the worker pipeline clean.

## Retention should not re‑implement timestamp parsing
Your retention engine currently re‑parses filenames.
Once you centralize timestamp parsing, retention becomes simpler and less error‑prone.

## Worker should log snapshot creation
Right now, the worker silently copies files.

Add:

```go
w.log.Info("snapshot archived", "path", finalPath)
```
This helps debugging and observability.

## Worker should validate the source file
Before copying:

```go
if _, err := os.Stat(srcPath); err != nil {
return "", fmt.Errorf("source missing: %w", err)
}
```
This prevents weird partial states.

## Worker should ensure atomicity
You already use tmp → rename, which is good.

But you should also:

ensure tmp files are cleaned up on failure

ensure rename overwrites safely (Windows vs Linux differences)

Your FS abstraction may already handle this, but it’s worth verifying.

## Retention should ignore unknown files
If someone drops a stray file into the archive folder, retention should skip it gracefully.

## Watcher should debounce events properly
You already have a debounce window in config.
Make sure your watcher uses it to avoid duplicate jobs.

If you want to explore how the watcher should behave, you can tap watcher design.

## Main.go should pass sourceDir + nodesFilename into Worker
Even if you postpone copying nodes.conf, the worker should know where nodes.conf lives.

Add fields:

```go
sourceDir     string
nodesFilename string
```
And pass them in `worker.New()`.

This avoids another refactor later.

## Retention should run after each snapshot
Your worker should call:

```go
w.retention.Apply(ctx, w.archiveDir, finalPath)
```
This ensures pruning happens automatically.

## Summary: What’s left to do (even without nodes.conf copy)

Here’s the prioritized list:

Add sourceDir + nodesFilename to Worker

Return a SnapshotResult struct instead of a string

Centralize timestamp parsing

Add logging

Add source file validation

Ensure tmp cleanup on failure

Harden retention against stray files

Ensure watcher debounce is used

Call retention after snapshot creation

These changes make your system clean, predictable, and ready for the eventual nodes.conf pairing.

## config structure

```yaml
source:
  path: /data
  rdb_filename: dump.rdb
  nodes_conf_filename: nodes.conf

destination:
  root: /dest
  snapshot_subdir: last

hostname:
  override_env: HOSTNAME_OVERRIDE
  fallback: auto

watch:
  mode: auto
  poll_interval: 5s
  debounce_window: 500ms

retention:
  last_count: 5
  rules:
    - name: daily
      cron: "0 0 * * *"
      count: 7
    - name: weekly
      cron: "0 0 * * 0"
      count: 4

logging:
  level: info
  format: json

reload:
  enabled: true
  method: fsnotify
```