# Architecture Document
RDB Archiver

## Overview
RDB Archiver is a small service that watches a Redis or Valkey RDB file and creates snapshot 
copies whenever the file changes. It also applies retention rules, exposes health endpoints, 
and provides Prometheus metrics. The design is simple: `detect -> enqueue -> process -> store -> clean up`.

This document explains the internal structure and how the main parts work together.

## Main Components

### Snapshot Watcher 

The watcher monitors the source directory for changes to the primary file (`dump.rdb`)

It supports two modes:
- `fsnotify` - real‑time file system events
- `polling` - periodic checks when `fsnotify` is not supported

The watcher uses `fsprobe` to test if `fsnotify` works on the current file system.
Some network file systems do not support `fsnotify` correctly, so `polling` is used instead.
Also, if the `config.yaml` file is a symlink, polling will be used for detecting the changes. 

When a new or updated RDB file is detected, the watcher:
- validates the snapshot
- creates a Snapshot object
- sends a job to the mailbox

### Mailbox

The mailbox is a small internal queue. It connects the watcher and the worker.

The mailbox:
- receives jobs from the watcher
- stores only the latest job (older ones may be overwritten)
- delivers jobs to the worker

The mailbox is a best‑effort queue. It holds maximum one job at a time.
If the worker is busy and a new snapshot arrives, the existing job in the queue is replaced with the newer one.
This ensures the worker always processes the most recent snapshot and avoids unnecessary backlog.

Metrics track how many jobs were enqueued, dequeued, or overwritten.

### Worker

The worker processes each snapshot job:
- copies the primary RDB file
- copies auxiliary files
- writes them into a timestamped directory
- compresses files
- updates metrics

The worker also handles retries if file operations fail.

### File System Layer

The `internal/fs` package provides:
- safe file copying
- retries with backoff
- compression
- atomic renames
- directory helpers

This layer hides OS differences (Linux vs Windows).

### Retention Engine

The retention engine removes old snapshots based on:
- `lastCount` (keep last N snapshots)
- cron‑based rules (hourly, daily, weekly, etc.)

Each rule has:
- a cron schedule
- a count (how many to keep)

Retention runs periodically and logs what it removes.

### Configuration System

The config file is YAML. Any value can use environment variables:

```yaml
subDir: "$(HOSTNAME)"
destination: "$(BACKUP_ROOT)"
```

The service can reload the config at runtime.
When running in Kubernetes with a ConfigMap, polling mode is recommended.

### Health Server

A small HTTP server exposes:
- `/live` - process is running
- `/ready` - service is ready to handle snapshots

The port is configurable.

### Prometheus Metrics

The service exposes metrics for:
- build info
- watcher activity
- mailbox queue
- worker processing
- retention cycles

These help operators understand how often snapshots are created, how long they take, and whether anything fails.

### Destination Layout

Snapshots are stored under:

```
<root>/<subDir>/snapshots/<timestamp>.tar.zst
```

Example:
```
/backup/my-pod/snapshots/
  2026-03-02T08-45-13.tar.zst
```

The timestamp is the modify moment of the RDB file.

### Redis and Valkey Compatibility

Redis and Valkey both write RDB files in the same format and with the same file names.
The archiver does not parse the RDB file; it only copies it.
Therefore, it works with both systems without any special handling.

### Data Flow
```
  +----------+-----------+
  |   Snapshot Watcher   |
  |  (fsnotify or poll)  |
  +----------+-----------+
             |
             v
      +------+------+
      |   Mailbox   |
      +------+------+
             |
             v
      +------+------+
      |   Worker    |
      +------+------+
             |
             v
  +----------+-----------+
  | Destination Storage  |
  +----------+-----------+
             |
             v
  +----------+-----------+
  |   Retention Engine   |
  +----------------------+
```

### Why this architecture

The design is intentionally simple:
- the watcher only detects changes.
- the mailbox decouples detection from processing.
- the worker does all file operations.
- retention is separate and runs on its own schedule.
- metrics and health endpoints make it observable.
- config reload avoids restarts in Kubernetes.

This separation keeps each part small and easy to reason about.

### Summary

RDB Archiver is built around a clear pipeline:
1. detect new RDB
2. enqueue job
3. process and store snapshot
4. apply retention rules

It works with Redis and Valkey, supports multiple file‑watching modes,
and is designed to run well in Kubernetes or on a single machine.