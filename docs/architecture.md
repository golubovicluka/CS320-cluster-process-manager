# Architecture

## Components

```mermaid
flowchart TB
    subgraph Interfaces
        CLI[clusterctl HTTP client]
        HTTP[REST transport]
        Simulator[Scenario runner]
    end
    subgraph Application
        Engine[Simulation engine]
        Controller[Cluster controller]
        Scheduler[Scheduler interface]
    end
    subgraph State and output
        State[In-memory cluster state]
        EventLog[Structured event log]
        Reports[Metrics and JSON/CSV reports]
        Snapshots[JSON snapshots]
    end
    CLI --> HTTP
    HTTP --> Controller
    HTTP --> Engine
    Simulator --> Engine
    Engine --> Controller
    Controller --> Scheduler
    Controller --> State
    Controller --> EventLog
    State --> Reports
    State --> Snapshots
```

The controller is the sole owner of live state. Public mutations take an exclusive lock; snapshots and list operations take a read lock and return deep copies. Schedulers receive immutable `NodeSnapshot` values and cannot mutate cluster state.

## Tick sequence

```mermaid
sequenceDiagram
    participant E as Engine
    participant C as Controller
    participant S as Scheduler
    participant N as Worker nodes
    E->>C: Step()
    C->>C: Increment virtual clock
    C->>C: Apply scheduled submissions/failures
    C->>C: Detect heartbeat timeouts
    loop each READY process
        C->>S: SelectNode(process, snapshots)
        S-->>C: node ID or no capacity
        C->>N: Allocate CPU and memory
    end
    C->>C: Sample utilization
    C->>C: Decrement RUNNING processes
    C->>N: Release completed resources
    C->>C: Validate invariants
```

The order is deterministic: process IDs and node snapshots use stable sorting, and scenario events occur on explicit virtual ticks. The same scenario, scheduler, and seed therefore produce the same result.

## Process lifecycle

```mermaid
stateDiagram-v2
    [*] --> NEW
    NEW --> READY: submit
    READY --> RUNNING: schedule
    RUNNING --> READY: failure restart / reschedule
    RUNNING --> WAITING: simulated wait
    WAITING --> READY: wait complete
    RUNNING --> PAUSED: pause
    PAUSED --> READY: resume
    RUNNING --> TERMINATED: remaining ticks = 0
    READY --> FAILED: crash
    RUNNING --> FAILED: crash
    PAUSED --> FAILED: crash
    FAILED --> READY: restart allowed
    READY --> KILLED: kill
    RUNNING --> KILLED: kill
    PAUSED --> KILLED: kill
    TERMINATED --> [*]
    FAILED --> [*]
    KILLED --> [*]
```

Only `RUNNING` processes consume resources. Leaving `RUNNING` releases the exact CPU and memory request and removes the process from the node list.

## Node failure recovery

```mermaid
flowchart TD
    Failure[Manual, scheduled, or heartbeat failure] --> Mark[Mark node FAILED]
    Mark --> Reset[Reset node allocations]
    Reset --> Affected[Visit affected processes in stable ID order]
    Affected --> Policy{Restart policy permits retry?}
    Policy -->|yes| Retry[Increment bounded restart count and return to READY]
    Policy -->|no| Failed[Keep process FAILED]
    Retry --> Reschedule[Scheduler selects another ONLINE node]
    Recover[Recover node] --> Recovering[RECOVERING]
    Recovering --> Online[ONLINE with no former assignments]
```

## Invariants

After every tick:

1. node CPU and memory allocations are between zero and capacity;
2. allocation counters equal the sum of requests for listed running processes;
3. every running process appears on exactly one node and its `NodeID` matches;
4. queued, paused, failed, killed, and terminated processes have no node assignment;
5. every node process ID exists in the process registry.

Failures are returned before a partially invalid command can corrupt an existing state. Snapshot consumers cannot mutate live objects because maps, slices, pointers, nodes, and processes are copied.

## Concurrency and shutdown

The real-time engine owns one cancellable ticker goroutine. `Pause` and `Close` cancel it and wait for completion. The HTTP server uses bounded read/write/header timeouts and a ten-second graceful shutdown. CI runs the complete suite with Go's race detector.
