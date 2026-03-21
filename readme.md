# RDB Archiver

## What this tool does

RDB Archiver watches a Redis or Valkey RDB file (usually `dump.rdb`).
When the file changes, it creates a snapshot (a backup copy).
It can also copy extra files like `nodes.conf`.

The tool keeps old snapshots using retention rules (for example: keep the last 6, keep 7 daily, keep 4 weekly).

It works on Linux and Windows. You can run it on your machine, in Docker, or in Kubernetes.

It supports config hot reload, so no Redis/Valkey node restart is needed if changes in config are needed.

The service is small, fast, and has a very low memory footprint.
It focuses on doing one thing well: detecting new RDB files and archiving them reliably.

## Works with Redis and Valkey

Redis and Valkey both write RDB files in the same format and with the same file names.
This tool works with both without any changes.

> **Important:** If you run Redis or Valkey in cluster mode, include `nodes.conf` in `auxNames`,
as it contains the sharding topology. A restore will fail if the topology in the destination
does not match the source.

## How it detects new snapshots

The tool can watch the RDB file in two ways:
- **fsnotify** (real‑time file change events)
- **polling** (check the file every few seconds)

If `auto` is used and `fsnotify` is not supported by the file system, the app will fall back to `polling`

## Where snapshots are stored
You choose a destination folder. Inside it, the app creates:

```
<root>/<subDir>/snapshots/<timestamp>/
```

You can use environment variables in the config, for example:
```yaml
subDir: "$(HOSTNAME)"
```

Any config value can use `$(ENV_NAME)`.

## Retention rules

You can keep:
- the last N snapshots
- snapshots based on cron rules (hourly, daily, weekly, etc.)

Old snapshots are removed automatically.

## Configuration file
The tool uses a YAML config file.
Example:
```yaml
source:
  path: "/data"
  primaryName: "dump.rdb"
  auxNames:
  - "nodes.conf"
  watchMode: "fsnotify"

destination:
  root: "/backup"
  subDir: "$(HOSTNAME)"
  snapshotSubdir: "snapshots"
  retention:
    lastCount: 6
    rules:
    - name: "daily"
      cron: "0 0 * * *"
      count: 7
```

## Health endpoints

The tool exposes two HTTP endpoints:
- `/live` - shows if the process is running
- `/ready` - shows if the process is ready to work

The port is set in the config (default: 8080).

## Prometheus metrics

The tool can expose Prometheus metrics if enabled. Metrics include:
- build info
- watcher events
- snapshots processed
- worker job counts
- bytes written
- retention activity

## Running in Kubernetes

You mount:
- `/data` - where Redis or Valkey writes dump.rdb
- `/backup` - where snapshots are stored
- `/config` - the config file (usually from a ConfigMap)

## More Info

You can find more info in [Architecture Document](docs/architecture.md) and [Technical Document](docs/technical.md).
A full fledged Kubernetes statefulset yaml snippet can be found in [Kubernetes Statefulset Snippet](docs/k8s.snip.md) 

## Summary

**RDB Archiver** is a small app that:
- watches Redis/Valkey RDB files
- creates snapshots
- keeps snapshots based on rules
- exposes health and metrics endpoints
- works well in Kubernetes
- supports config hot reload
- supports environment variables in config