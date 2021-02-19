# Event Trail Design

The event trail is a background thread that takes in events from the harness and asynchronously sends them to the master via API calls.

## Thread


- The `EventTrail` has one list per event type. This is a standard list with append and pop access wrapped with a lock*
- It stores a dictionary of EventType -> (list, lock, processing_fn, batch_size) 

- When an event is submitted to the event thread, the thread identifies the type of the event and identifies the list+lock
- The thread acquires the lock and appends the event to the list

*see below for why we use a standard list instead of an out-of-the box threadsafe queue (TLDR: bad tradeoffs - queues have performance we don't need, but lack certainty that we do need, such as 'have we processed every event?')

## Thread Main Loop

- For each EventType:
    - acquire the lock
    - if there are not any events in the list, continue
    - pull out a batch and update the list 
    - release the lock (we don't want to be holding the lock while making API calls because that could block the main python code)
    - Run the processing_fn on the batch (this is the code that takes in the batch of events and makes the api calls)

- During cleanup, 
    - send any remaining events (but only for MustDeliver EventType)



## EventType

- Name
- Processing_fn (should be written so that it works well with multi-threading)
- Batch size
- Priority

### Adding New Events

- Create an event that inherits from `EventTrailEvent`, e.g. `SomeEventV1`
    - Every event must be versioned
    - Any changes to the data being sent requires an increment in the version
- Create a `processing_fn` for that event, e.g. 

```python
def some_event_v1_processing_fn(batch: List[SomeEventV1]):
```

- Add a protobuf (unauthenticated) API under the `/api/v1/harness-events`. 
- Documentation is here (TODO)
- Have the API send a message to the telemetry actor

- Consider how BigQuery queries will incorporate both the old event and the new events. 

## Other Considerations

- Should we parallelize sending of events to API? 
    - Eventually we might want to use a threadpool or something, but right now that's overkill
    - The sending thread is designed to parallelize well if we want to go down this route eventually
- Should there be a way to support real-time events that are latency sensitive? 
    - e.g. having them at the top of the for loop?
    - Right now not enough events or sensitivity to latency for this to matter.
    - Parallelizing API calls would be the first step here
- Should we have some best-effort mechanism for sending droppable events during cleanup? 
    - It's complicated because it can add a non-negligible amount of time for each EventType.

## Lists vs multiprocessing Queues


