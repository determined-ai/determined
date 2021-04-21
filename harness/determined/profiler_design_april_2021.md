# Profiler Design Notes (April 2021)


This doc describes in detail the design of the ProfilingAgent as of when it was written. This is not documented inline because the details of the implementation will likely drift from the original design over time. However, this will be a useful starting point for someone unfamiliar with the Profiling code.

This differs from the ERD in that it describes the specifics of the Python implementation.


## Overview

ERD: https://docs.google.com/document/d/1iDO7_a5nXhbj68I6DmKNWZS8d3JmU-3809tieD4HuwQ/edit?usp=sharing

We want to be able to collect profiling information from the harness and send it to the master to be displayed in the WebUI. There are two main types of profiling information that we want right now: System Metrics (e.g. network throughput, memory usage, GPU utilization, etc) and Timings (e.g. duration of dataloader.next(), duration of forwards pass, etc).

System Metrics need to be collected at a high granularity to be useful. For example, if we average network throughput over a minute, we are averaging periods of high network usage (e.g. backwards passes) with periods of low network usage (e.g. forwards passes) and will get an average network throughput that says we are not bottlenecked by network, even though we may be bottlenecked by network during every backwards pass.

## Features

We need many of the standard features of a metric collection utility like Telegraf as well as some Determined-specific ones

- Send data to the master API in batches
- Associate each System Metric or Timing measurement with a specific batch idx
- Begin on a specific batch
- Optionally end after a specific batch
- End when the harness exits - do not allow threads to prevent the harness from shutting down
- Collect no more than 5 minutes worth of data (to protect the DB)
- Collect System Metrics once per machine (local rank=0)

## Acceptable Deficiencies 

- If a measurement fails, log and skip it
- If the harness cannot make API calls to the master, retry once, then log and skip it.
- Do not implement backpressure - try to send data to the master as quickly as possible to avoid a buildup of data to be sent, but do no more than that.
- When the Profiler shuts down, drop any partial batches rather than sending them.
- Collect timings only from a single GPU (global rank=0)


## Python Implementation

### ProfilingAgent
There is a top-level object called the ProfilerAgent. This has a minimal public API. It has startup/shutdown methods `start` and `end` (also usable via a context manager). Then it has two public facing methods: `update_batch_idx` that should be called by the training loop every time the batch index advances and `record_timing` which is called whenever there is a new Timing to record.

```python
prof = ProfilerAgent(self, trial_id, agent_id, master_url, start_on_batch, end_after_batch)
prof.start()

for batch_idx, batch in enumerate(batches):
    prof.update_batch_idx(batch_idx)

    # NOTE: Timing API has not been fully developed yet
    forward_pass_timing = Timing("forward_pass")
    forward_pass_timing.start()
    # Do forward pass
    forward_pass_timing.end()
    prof.record_timing(forward_pass_timing)

prof.end()
```

This top-level ProfilerAgent object exists primarily to keep the public API consistent and simple. It also holds the logic around what is enabled/active (see next section).

#### States: 'enabled' vs 'active'

The ProfilingAgent has a number of different states. The concepts of 'enabled' vs 'active' are distinct, where 'enabled' is fixed over the lifetime of the ProfilingAgent while 'active' changes over the lifetime. 'enabled' is whether profiling should happen at all, while 'active' is whether profiling data should be being collected and sent to the master right now.

In terms of being 'enabled': 
- the ProfilingAgent overall can be disabled or enabled based on the `profiling_enabled` field in the experiment config. 
- System Metric collection is enabled if `profiling_enabled=True` in the experiment config and `local_rank = 0`
- Timing collection is enabled if if `profiling_enabled=True` in the experiment config and `global_rank = 0`
- If `local_rank != 0` and `global_rank != 0`, the behavior is identical to `profiling_enabled=False`

This is reflected by the following fields/properties in the ProfilingAgent
- `profiling_is_enabled_in_experiment_config` reflects the experiment config setting
- `sysmetrics_is_enabled` reflects a combination of the experiment config setting and the local rank
- `timings_is_enabled` reflects a combination of the experiment config setting and the global rank
- `is_enabled` reflects all of the above.


'active' reflects whether SysMetrics/Timings are actively being collected and sent to the master. In terms of being 'active':
- The ProfilerAgent is not active when `current_batch_idx < start_on_batch`
- Once the `start_on_batch` is reached, the ProfilerAgent becomes active
- The ProfilerAgent goes back to being inactive either when `current_batch_idx > end_after_batch` or after the ProfilerAgent has been active for 5 minutes.
- If `is_enabled==False`, the ProfilerAgent can never be active.

While SysMetrics may be 'enabled' while Timings are 'disabled', 'active' is a property of the ProfilingAgent as a whole. If both SysMetrics and Timings are enabled, they will either both be actively collecting/sending data or neither will.

 
#### Children Threads

Inside of the ProfilingAgent, there are a number of children threads. 
1. There is the `SenderThread`, which is fed by the `send_queue`. It takes in batches of data and sends them to the Master API. It exists so that sending data to the master does not block. 
2. There is the `SysMetricCollectorThread` which collects System Metrics as `Measurements` every 100 ms. This thread is also responsible for batching together `Measurements` every 10 seconds (`SysMetricBatcher`). Those batches are then sent off to the `SenderThread` (fire-and-forget). The `SysMetricBatcher` is responsible for converting the data from collections of `Measurements` to the format expected by the Master API before sending to the `SenderThread`.
3. There is a the `TimingsBatcherThread`. This is TBD, but exists to batch together Timings to be sent off to the `SenderThread`
4. There is a `TimeoutThread` that exists to signal all the other threads to shut down after 5 minutes.

Each of the threads has a `control_queue` that receives `StartMessage`s and `ShutdownMessage`s. `StartMessage`s indicate that we should begin collecting metrics. `SenderThread` is simple enough that it does not need a `StartMessage`. `ShutdownMessage` indicates that the thread should shut down. The control queue is abstracted away behind the `activate` and `kill` methods that each thread has.

Which of these threads are created depends on what is enabled. If either System Metrics or Timings are enabled (`prof.is_enabled==True`), then we will need the TimeoutThread and the SenderThread. If System Metrics are enabled, then the `SysMetricCollectorThread` is created. If Timings are enabled, then the `TimingsBatcherThread` is enabled.

#### Logical Flow

The ProfilerAgent has a couple of steps.

1. Init - create the ProfilerAgent object and create the child thread objects. Do not start the threads.
2. Start - Start the child threads. The threads are not doing any work, they are just waiting for a `StartMessage`\*
3. Begin collection - Send the `StartMessage`s so the threads start doing useful work. This begins the 5 minute timeout timer
4. End collection - Send `ShutdownMessage`s so the threads stop doing work. After this step, no child threads are running.

\* Technically the `SenderThread` doesn't wait for a StartMessage, but it has nothing to send until one of the other threads starts doing useful work, so logically it is also waiting for the `StartMessage`s to be sent.





