rdb‑archiver Architecture (Condensed)
Your application is a pipeline that turns raw Redis/Valkey state files into archived snapshot events, then applies retention rules to keep the archive clean and bounded.

It has four major subsystems:

Watcher

Queue

Worker

Retention Engine

Everything else (config, logging, filesystem abstraction) supports these four.

1. Watcher — Detects New Snapshots
   The watcher monitors the Redis/Valkey data directory for changes to:

dump.rdb

nodes.conf (later)

It can operate in two modes:

fsnotify (event‑driven)

polling (fallback)

When it detects a new RDB write, it debounces events to avoid duplicates, then enqueues a job:

Code
watcher → queue.Push(job{srcPath: /data/dump.rdb})
Constraints:

Must never copy files itself

Must never block

Must be idempotent (multiple events collapse into one job)

2. Queue — Decouples Watcher from Worker
   A simple in‑memory FIFO queue:

Watcher pushes jobs

Worker pops jobs

Ensures backpressure and isolation

This keeps the watcher fast and the worker single‑threaded and predictable.

3. Worker — Produces Snapshot Events
   The worker pipeline is the heart of the system.

It takes a job and:

Extracts the timestamp from the RDB filename

Copies the RDB into the archive directory using atomic tmp→rename

(Later) Copies nodes.conf into a matching timestamped filename

Returns a SnapshotResult

Triggers retention

This is where the snapshot event is created — a pair of files with the same timestamp.

Constraints:

Must be atomic (tmp → rename)

Must never partially write snapshots

Must not parse timestamps in multiple places

Must call retention after each snapshot

Must use the filesystem abstraction (OSFS)

4. Retention Engine — Keeps Archive Bounded
   The retention rules are simple and folder‑based:

Each rule:

yaml
- name: daily
  count: 7
  Means:

Code
archive/daily/ → keep last 7 snapshot events
A snapshot event is:

dump-<ts>.rdb

nodes-<ts>.config (optional for now)

Retention:

Scans the folder

Groups files by timestamp

Sorts newest → oldest

Keeps the first N

Deletes the rest (both files)

Constraints:

Must ignore stray files

Must not assume nodes.conf exists

Must be deterministic

Must not block the worker for long

📦 Data Flow (End‑to‑End)
```Code
Redis writes dump.rdb
↓
Watcher detects change
↓
Watcher enqueues job
↓
Queue buffers job
↓
Worker pops job
↓
Worker copies dump.rdb → archive/<rule>/dump-<ts>.rdb
↓
Worker (later) copies nodes.conf → archive/<rule>/nodes-<ts>.config
↓
Worker triggers retention
↓
Retention prunes old snapshot events
```
This is a one‑way, append‑then‑prune pipeline.

🔒 Main Constraints (The Rules of the System)
1. Watcher must never copy files
   It only detects and enqueues.

2. Worker must produce complete snapshot events
   Even if nodes.conf is postponed, the worker is the only place where pairing will happen.

3. Retention must operate on folders, not global state
   Each rule corresponds to a folder.

4. Timestamps must be extracted once, consistently
   Centralized timestamp parsing avoids drift.

5. All file operations must be atomic
   tmp → rename ensures no partial snapshots.

6. The archive must remain monotonic
   Snapshots only ever move forward in time; retention deletes old ones.

7. The system must be resilient to restarts
   Queue is in‑memory, but watcher will re‑detect changes.

his is the “executive‑summary but actually useful” version of the system you’ve built.

🧱 1. High‑Level Architecture Overview
Your application is a four‑stage pipeline:

Code
Watcher → Queue → Worker → Retention
Each stage has a single responsibility and hands off to the next.
This keeps the system predictable, testable, and easy to extend.

Key concepts you can explore anytime:

Watcher design

Worker pipeline

Snapshot events

Retention rules

🔍 2. Detailed Data Flow (End‑to‑End)
Code
Redis writes dump.rdb
↓
Watcher detects change (fsnotify or polling)
↓
Watcher debounces events
↓
Watcher enqueues a job into the queue
↓
Worker pops job
↓
Worker extracts timestamp from dump.rdb
↓
Worker copies dump.rdb → archive/<rule>/dump-<ts>.rdb
↓
Worker (later) copies nodes.conf → archive/<rule>/nodes-<ts>.config
↓
Worker returns SnapshotResult
↓
Worker triggers retention
↓
Retention scans archive/<rule>/
↓
Retention keeps last N snapshot events
↓
Retention deletes older events
This is a one‑directional, append‑then‑prune pipeline.

🧩 3. Subsystem Responsibilities
3.1 Watcher
The watcher:

Monitors the source directory (dump.rdb)

Uses fsnotify or polling

Debounces rapid events

Pushes jobs into the queue

Constraints:

Must never copy files

Must never block

Must be idempotent

Must not parse timestamps

3.2 Queue
A simple FIFO job queue:

Watcher pushes jobs

Worker pops jobs

Constraints:

Must not drop jobs

Must not block watcher

Must serialize worker execution

3.3 Worker
The worker pipeline is the heart of the system.

It:

Receives a job with the path to dump.rdb

Extracts timestamp from filename

Copies RDB atomically (tmp → rename)

(Later) Copies nodes.conf with same timestamp

Produces a SnapshotResult

Triggers retention

Constraints:

Must be atomic

Must not partially write snapshots

Must centralize timestamp parsing

Must use filesystem abstraction

Must call retention after each snapshot

3.4 Retention Engine
The retention rules are folder‑based:

Example rule:

yaml
- name: daily
  count: 7
  Meaning:

Code
archive/daily/ → keep last 7 snapshot events
A snapshot event is:

dump-<ts>.rdb

nodes-<ts>.config (optional for now)

Retention:

Scans folder

Groups files by timestamp

Sorts newest → oldest

Keeps first N

Deletes the rest

Constraints:

Must ignore stray files

Must not assume nodes.conf exists

Must be deterministic

Must not block worker too long

🗂 4. Directory Structure
Code
/data/
dump.rdb
nodes.conf

/archive/
daily/
dump-2025-02-24T23-59-00.rdb
nodes-2025-02-24T23-59-00.config
...
weekly/
...
monthly/
...
Each retention rule corresponds to a folder.

🧠 5. Core Data Model
Snapshot Event
A logical snapshot consists of:

RDB file

nodes.conf file (optional for now)

Shared timestamp

Represented as:

go
type SnapshotEvent struct {
Timestamp time.Time
RDBPath   string
NodesPath string
}
This is the unit of retention.

🔒 6. System Constraints (The Rules of the Game)
1. Watcher must never copy files
   Only detect and enqueue.

2. Worker must produce complete snapshot events
   Even if nodes.conf is postponed.

3. Retention must operate per folder
   Each rule = one folder.

4. Timestamps must be parsed in one place
   Avoid drift and bugs.

5. All file writes must be atomic
   tmp → rename.

6. Archive must remain monotonic
   Snapshots only move forward in time.

7. System must be restart‑safe
   Watcher will re‑detect changes.

🧭 7. Sequence Diagram (ASCII)
Code
Redis → Watcher: dump.rdb updated
Watcher → Queue: enqueue job
Queue → Worker: deliver job
Worker → FS: copy dump.rdb (tmp → rename)
Worker → FS: (later) copy nodes.conf
Worker → Retention: apply rules
Retention → FS: delete old snapshots
Worker → Main: SnapshotResult
🧱 8. Mermaid Diagram

```mermaid
flowchart TD
    A[Redis writes dump.rdb] --> B[Watcher detects change]
    B --> C[Debounce]
    C --> D[Queue.push(job)]
    D --> E[Worker.pop(job)]
    E --> F[Worker copies dump.rdb]
    F --> G[Worker copies nodes.conf (later)]
    G --> H[Worker returns SnapshotResult]
    H --> I[Retention.apply()]
    I --> J[Retention prunes old snapshots]
```
🎯 9. The Architecture in One Sentence
rdb‑archiver is a watcher→queue→worker→retention pipeline that turns raw Redis state files into timestamped snapshot events and keeps only the most recent ones per retention rule