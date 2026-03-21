# Technical Guide
RDB Archiver

This document explains how to build, run, configure, and deploy RDB Archiver. It also includes full Prometheus metrics, Kubernetes wiring, and Redis/Valkey notes.

## Requirements
-  Go 1.25 or newer
- Redis or Valkey producing an RDB file (dump.rdb)
- Linux or Windows
- Optional: Docker, Kubernetes, Prometheus

The tool does not parse RDB files. It only copies them, so it works with both Redis and Valkey.

## Getting the Source Code
```sh
git clone https://github.com/raoulx24/rdb-archiver.git
cd rdb-archiver
go mod tidy
```

## Building the Binary

```sh
go build ./cmd/rdb-archiver
```

This produces a binary named rdb-archiver (or rdb-archiver.exe on Windows).

## Running Locally
   Create or copy the sample config file `config.yaml` in `config` folder. Run the service:

```sh
./rdb-archiver
```

> * 

## Configuration Details

The config file is YAML. Any value can use environment variables:

```yaml
subDir: "$(HOSTNAME)"
destination: "$(BACKUP_ROOT)"
```

The service expands `$(ENV_NAME)` at runtime.

Below is a full explanation of each section.

### `source`
```yaml
source:
  path: "/data"
  primaryName: "dump.rdb"
  auxNames: ["nodes.conf"]
  watchMode: "fsnotify"
```

- `path` - directory where Redis/Valkey writes the RDB file
- `primaryName` - usually `dump.rdb`
- `auxNames` - extra files to copy
- `watchMode` - `auto`, `fsnotify`, or `poll`

> **Important** `nodes.conf` should be included when archiving Redis/Valkey cluster nodes.
It stores the cluster topology (node IDs, slots, replication). A restore
may fail or start in an inconsistent state if `nodes.conf` is missing or
different.


### `destination`
```yaml
destination:
  root: "/backup"
  subDir: "$(HOSTNAME)"
  snapshotSubdir: "snapshots"
  retention:
    lastCount: 6
    removeUnknownFolders: true
    rules:
    - name: "daily"
      cron: "0 0 * * *"
      count: 7
```
- `root` - base folder for snapshots
- `subDir` - usually the pod name or hostname
- `snapshotSubdir` - folder where snapshots go
- `retention` - rules for keeping/deleting old snapshots

Retention supports:
- `lastCount` - keep last N snapshots
- cron rules - keep N snapshots per schedule

### `watchFS`

```yaml
watchFS:
  fsnotify:
    debounceWindow: "200ms"
  pool:
    interval: "5s"
    stabilityWindow: "200ms"
```
- `debounceWindow` - to protect for notify events storms
- `stabilityWindow` - to protect against file changes

### `fs`
```yaml
fs:
  maxRetries: 7
  retryBase: "50ms"
  retryDurationCap: "1s"
  compressionLevel: 2
```
File system behavior:
- `maxRetries`, `retryBase`, `retryDurationCap` are for retrying policies
- `compressionLevel` is for `zst` files

### `logging`

```yaml
logging:
  level: "info"
  format: "json"
```
- `level` - can be `debug`, `info`, `warn`, `error`
- `format` - can be `json`, `text`

Use log format
- `json` when structured logs are needed and a log shipper (for example `fluentd`) sends them to a system like `Elasticsearch`
- `text` when human‑readable logs are preferred (local debugging, simple setups)

### `health`

```yaml
health:
  port: 8080
```
Endpoints:
- `/live`
- `/ready`

### `prometheus`

```yaml
prometheus:
  enabled: true
  port: 9090
  histogramBuckets: [0.1, 0.5, 1, 5, 10]
```

### `configReload`

```yaml
configReload:
  enabled: true
  method: "poll"
```
Polling is recommended when using a `ConfigMap` in Kubernetes.

## Directory Layout of Snapshots

Snapshots are stored as:
```
<root>/<subDir>/snapshots/<timestamp>.tar.zst
```
Example:
```
/backup/my-pod/snapshots/
  2026-03-02T08-45-13.tar.zst
  2026-03-02T08-47-35.tar.zst
  [...]
/backup/my-pod/daily/
  2026-03-01T00-01-02Z.tar.zst
  2026-03-02T00-03-04Z.tar.zst
  [...]
```

## Docker Usage

Run the container:
```sh
docker run \
-v /path/to/rdb:/data \
-v /path/to/backups:/backup \
-v /path/to/config.yaml:/config/config.yaml \
ghcr.io/raoulx24/rdb-archiver:latest
```

## Kubernetes Deployment

You need three mounts:
- `/data` - where Redis/Valkey writes dump.rdb
- `/backup` - where snapshots are stored
- `/config` - the config file

Example container section:
```yaml
containers:
- name: rdb-archiver
  image: ghcr.io/raoulx24/rdb-archiver:0.0.1-35
  env:
  - name: HOSTNAME
    valueFrom:
      fieldRef:
        apiVersion: v1
        fieldPath: metadata.name
  volumeMounts:
  - mountPath: /data
    name: redis-data
  - mountPath: /backup
    name: backup-volume
  - mountPath: /config
    name: rdb-archiver-config
[...]
volumes:
- name: rdb-archiver-config
  configMap:
    name: rdb-archiver-configmap
```

Full yaml snip in [Kubernetes Statefulset Snippet](technical.md)

## Prometheus Metrics (Full List)

| package         | metric name                                         | description                                  |
|-----------------|-----------------------------------------------------|----------------------------------------------|
| buildInfo       | rdb_archiver_build_info                             | build metadata (version, commit, build_date) |
| mailbox         | rdb_archiver_mailbox_jobs_enqueued_total            | jobs added to mailbox                        |
| mailbox         | rdb_archiver_mailbox_jobs_overwritten_total         | jobs overwritten when mailbox full           |
| mailbox         | rdb_archiver_mailbox_jobs_dequeued_total            | jobs removed from mailbox                    |
| retention       | rdb_archiver_retention_runs_total                   | retention cycles executed                    |
| retention       | rdb_archiver_retention_snapshots_processed_total    | snapshots processed per rule/outcome         |
| retention       | rdb_archiver_retention_deletions_total              | snapshots deleted per rule                   |
| snapshotwatcher | rdb_archiver_watcher_events_received_total          | filesystem events received                   |
| snapshotwatcher | rdb_archiver_watcher_snapshots_parsed_total         | valid snapshots detected                     |
| snapshotwatcher | rdb_archiver_watcher_invalid_snapshots_total        | invalid or unreadable snapshots              |
| snapshotwatcher | rdb_archiver_watcher_jobs_enqueued_total            | jobs sent to mailbox                         |
| worker          | rdb_archiver_worker_jobs_processed_total            | jobs processed successfully                  |
| worker          | rdb_archiver_worker_jobs_failed_total               | jobs that failed                             |
| worker          | rdb_archiver_worker_jobs_retried_total              | jobs retried                                 |
| worker          | rdb_archiver_worker_job_processing_duration_seconds | histogram of job processing time             |
| worker          | rdb_archiver_worker_bytes_written_total             | total bytes written                          |
| worker          | rdb_archiver_worker_destination_total_bytes         | destination total size                       |
| worker          | rdb_archiver_worker_destination_free_bytes          | destination free size                        |

## Troubleshooting

### fsnotify does not work

Some file systems (especially network storage) do not support fsnotify. Use:
```yaml
watchMode: "poll"
```
### No snapshots created
Check:
- correct `source.path`
- correct `primaryName`
- Redis/Valkey actually writes `dump.rdb`
- permissions on `/data`

### Retention not running
- Check cron expressions.
- Check logs for retention cycles.

### Slow snapshot creation
Check:
- network storage latency
- compression level
- retry settings

## Summary

This guide covers:
- building the binary
- running locally
- configuration
- Docker usage
- Kubernetes deployment
- full Prometheus metrics
- troubleshooting

RDB Archiver is simple to operate and works with both Redis and Valkey because
it only copies RDB files and does not parse them.