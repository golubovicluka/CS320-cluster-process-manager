# REST API

Base URL: `http://localhost:8080/api/v1`

All JSON request bodies reject unknown fields and are limited to 1 MiB. Domain errors are translated to `400`, `404`, `409`, or `422`; unexpected errors return `500`.

## Nodes

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/nodes` | Register a node |
| `GET` | `/nodes` | List nodes |
| `GET` | `/nodes/{id}` | Get one node |
| `PATCH` | `/nodes/{id}/status` | Set `ONLINE`, `DRAINING`, `OFFLINE`, or `FAILED` |
| `POST` | `/nodes/{id}/fail` | Fail a node and process its workloads |
| `POST` | `/nodes/{id}/recover` | Recover a failed/offline node |
| `POST` | `/nodes/{id}/heartbeat` | Record a heartbeat at the current tick |
| `DELETE` | `/nodes/{id}` | Remove an empty node |

```bash
curl -X POST http://localhost:8080/api/v1/nodes \
  -H 'Content-Type: application/json' \
  -d '{"id":"node-1","name":"Worker 1","cpuCapacity":8,"memoryCapacityMB":16384}'
```

## Processes

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/processes` | Validate and submit a process to `READY` |
| `GET` | `/processes` | List processes |
| `GET` | `/processes/{id}` | Get one process |
| `POST` | `/processes/{id}/pause` | Pause a running process and release resources |
| `POST` | `/processes/{id}/resume` | Return a paused process to `READY` |
| `POST` | `/processes/{id}/wait` | Move a running process to simulated I/O waiting |
| `POST` | `/processes/{id}/wake` | Complete simulated I/O and return to `READY` |
| `POST` | `/processes/{id}/kill` | Kill a non-terminal process |
| `POST` | `/processes/{id}/fail` | Inject a process failure |

```bash
curl -X POST http://localhost:8080/api/v1/processes \
  -H 'Content-Type: application/json' \
  -d '{"id":"p1","name":"image-worker","priority":10,"cpuRequest":2,"memoryRequestMB":512,"totalTicks":20,"timeQuantum":3,"restartPolicy":"ON_FAILURE","maxRestarts":2}'
```

Larger numeric priority values run first. Priority aging adds one effective priority point per five ticks spent in `READY`.

## Simulation and scheduler

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/simulation/start` | Start real-time ticking |
| `POST` | `/simulation/pause` | Stop real-time ticking |
| `POST` | `/simulation/step` | Run `{"ticks": N}` deterministic ticks |
| `POST` | `/simulation/reset` | Clear state and events |
| `GET` | `/simulation/status` | Return a deep cluster snapshot |
| `POST` | `/simulation/scenarios` | Atomically load a scenario object |
| `GET` | `/scheduler` | Get current/available schedulers |
| `PUT` | `/scheduler` | Select `{"name":"least-loaded"}` |

Loading a scenario does not automatically run it. Use `step` or `start` afterward.

## Events and reports

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/events` | Structured event history |
| `GET` | `/metrics` | Current report |
| `GET` | `/reports/summary` | Current report alias |
| `GET` | `/reports/export?format=json` | Download JSON report |
| `GET` | `/reports/export?format=csv` | Download CSV report |
| `GET` | `/healthz` | Service health (outside `/api/v1`) |

Metrics include process state counts, restarts, waiting/turnaround time, throughput, success rate, CPU/memory utilization, deferrals, failures, reschedulings, tick count, and finish reason.
