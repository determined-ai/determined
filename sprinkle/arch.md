# Pre-Sprinkle Architecture Diagram

All communication to the master is based on a single websocket per-container.
Rendezvous info and workloads go through the same channel, metrics and
checkpoints are returned on the same channel, and preemption is merged into the
workload stream via the TrialWorkloadSequencer in the master.

Notice that there are two different layers of workload duplication

```
                        __________
                       |          |
                       |  Master  |
                       |__________|
                            |
     stream of workloads   / \  duplicate workloads
     +--------------------+   +----------------------+
 ____|_____________________       ___________________|______
|    |                     |     |                   |      |
|    | Container 1         |     |      Container 2  |      |
|  __|_________________    |     |  _________________|__    |
| |  |                 |   |     | |  |                 |   |
| | SocketManager      |   |     | | SocketManager      |   |
| |  |                 |   |     | |  |                 |   |
| | WorkloadManager    |   |     | | WorkloadManager    |   |
| |  |                 |   |     | |  |                 |   |
| | SubprocessLauncher |   |     | | SubprocessLauncher |   |
| |__|_________________|   |     | |__|_________________|   |
|    |                     |     |    |                     |
|    | duplicate workloads |     |    | duplicate workloads |
|    +--------------+      |     |    +--------------+      |
|    |              |      |     |    |              |      |
|  __|_____    _____|__    |     |  __|_____    _____|__    |
| |  |     |  |     |  |   |     | |  |     |  |     |  |   |
| | Chief  |  | Worker |   |     | | Worker |  | Worker |   |
| |________|  |________|   |     | |________|  |________|   |
|__________________________|     |__________________________|

```

# Post-sprinkle Architecture Diagram

All communication to the master is based on the new push architecture.

```
 Push APIs: metrics, checkpoints,
    searcher ops, preemption
+------------------------------+
|                              |
|                          ____|_____
|          rendezvous api |          |  rendezvous api
|        +----------------|  Master  |-----------------+
|        |                |__________|                 |
|        |                                             |
|        |  Agent 1                        Agent 2     |
|    ____|___________________       ___________________|____
|   |    |                   |     |                   |    |
|   |    | Container 1       |     |      Container 2  |    |
|   |  __|_________________  |     |  _________________|__  |
|   | |                    | |     | |                    | |
|   | |  Rendezvous Layer  | |     | |  Rendezvous Layer  | | <- fixed behavior
|   | |____________________| |     | |____________________| |
|   |  ____________________  |     |  ____________________  |
|   | |                    | |     | |                    | |
|   | |    Launch Layer    | |     | |    Launch Layer    | | <- user-customizable
|   | |____________________| |     | |____________________| |
|   |  ________    ________  |     |  ________    ________  |
|   | |        |  |        | |     | |        |  |        | |
+------ Chief  |  | Worker | |     | | Worker |  | Worker | | <- user-customizable
    | |__|_____|  |__|_____| |     | |__|_____|  |__|_____| |
    |____|___________|_______|     |____|___________|_______|
         |           |                  |           |
         +-----------+                  |           |
         +------------------------------+           |
         +------------------------------------------+
         zmq comms: custom reducers + preemption sync
```
