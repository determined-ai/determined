# Streaming Updates ERD

## Glossary

### Record

Some state.  Basically a row in a database table.  Every record has a sequence
number.

### Sequence Number

A monotonically increasing sequence of records in a single table.  Each table
will have its own sequence, and the sequence number of records in one table
are not related to the sequence numbers of other tables.

### Event

A message sent to the streaming client.  Various event types exist:

- Insertion
- Update
- Deletion
- Appearance
- Disappearance
- Fallout

### Streaming Client

An entity which connects over websocket to the server, and subscribes to some
set of states it wants to receive events for.

Streaming clients are assumed to be stateful and robust to disconnections.  A
streaming client will assume that it can reconnect with the same set of
subscriptions and the last known sequence number(s) seen before
disconnecting, and expect to be able to pick up streaming where it left off.

### Online vs Offline

"Online" and "offline" are adjectives which apply to an event, if the event
occurs while a streaming client is connected ("online") or while the streaming
client is disconnected ("offline").

The words "online" and "offline" only make sense when considering a single
streaming client at a time; an event may be "online" from one client's
perspective, but "offline" from another client's perspective.

### Insertion

A record is added.  Immediately after insertion, its sequence number should be
higher than the sequence number all previously-inserted records.

### Update

A record is changed.  Immediately after the change, its sequence number should
be higher than the sequence number all previously-inserted records.

### Deletion

A record is deleted.  The record no longer exists in the table[1].

[1] in theory, it could be left in place and flagged as deleted, or it could be
added to a deletion log with some TTL.

### Appearance/Disappearance

A record is unchanged, but due to RBAC changes, the client is now able to view
record that it was not able to view earlier (or not view, in the case of
disappearances).  Immediately after an appearance or disappearance, there is
_no guarantee_ that the new record has a sequence number which is higher than
any other records in its table, as the record itself wasn't modified.

## Fallin/Fallout

A Fallin is when a record changes in a way that it no longer meets the
requirements of the subscription; it "falls in" to the subscribed set.

A Fallout is when it changes to no longer be in the subscription set.

An example of Fallout this would be a streaming client subscribed to all
experiments matching workspace\_id==1, and one such experiment is moved to a
different workspace.

Fallin/Fallout are only possible when server-side filtering is allowed on
mutable fields.  Streaming all trials with experiment\_id==1 can't have fallout
for instance, because trials cannot be moved from one experiment to another.

## Problems

### RBAC problem

Obviously streaming must take RBAC into account.

Note that the presence of RBAC and the behaviors of RBAC are generally so
custom to our system that if we incorporate any 3rd party library to help us
with streaming updates, we must implement the RBAC portion in a customized way.

Generally speaking, this level of required customization renders most 3rd party
solutions unhelpful.

### Deletion problem

Broadcasting online deletions is fairly easy.  As long as the TRIGGER emits the
primary key and filterable attributes of the deleted entity, all connected
clients whose subscriptions match the deleted entity can simply be sent a
deletion message.  An alternate solution would be to keep an in-memory cache of
filterable attributes, and use the in-memory cache to do the subscription
matching.

Broadcasting offline deletions is more challenging.  There are generally three
strategies:

- The **soft deletion** strategy: don't delete rows from the database, just set
  a deleted=1 column.  This has space, performance, and privacy penalties.

- The **event log** strategy: keep a log of deletions, so clients which
  reconnect can find out what has been deleted since they last logged on.  By
  storing less than the whole record, you get immediate space benefits vs the
  soft deletion strategy.  Further space benefits can be obtained by adding a
  TTL to entries in the log, and just invalidating client caches when
  long-out-of-date clients connect.

- The **declarative** strategy: the client reconnects with a list of entities
  of which it knows about.  Among the first wave of messages the server sends
  is a list of client-known entities which no longer exist.

We stick to the declarative strategy because it is best suited to solving the
remaining problems.

### Disappearance Problem

Offline disappearances are effectively solved by the **declarative** strategy
for deletions, and don't require much additional thought.

Online disappearances should be fairly rare (presumably rbac changes are much
less common than say, experiment or trial state changes).  Possible strategies
are:

- The **just boot'em** strategy: any streaming client whose rbac situation has
  changed gets forcibly disconnected.  Then the declarative strategy for
  offline deletions can serve as our solution.

- The **sequence bump** strategy: artificially update every sequence number for
  every item in the database affected by an rbac change, forcing the streaming
  architecture to revisit/restream all affected items.  This isn't a complete
  solution, as it leaves the server in a situation similar to Fallout.  Note
  that the upper limit on number of records modified is effectively the whole
  database.  This isn't a great idea.

- The **streaming** strategy: for each rbac change, figure out which connected
  streamers are affected and just send them deletion-like messages.
  Technically possible, but not easy.  Might require keeping an in-memory clone
  of what entities a client knows about on a per-connection basis, which would
  have big memory costs.

We prefer to just boot'em, as it's the simplest strategy, and the problem
shouldn't arise very often.

### Appearance Problem

Online appearances can be solved by combining just boot'em with the declarative
strategy for offline deletions, however it adds constraints to how the
declarative strategy is implemented.  The declarative strategy could normally
be implemented in two different ways:

- compare the client's known entities against all client-visible entities and
  calculate deletions

- compare the client's known entities against all client-visible entities
  _matching the stream's subscription_ and calculate deletions.

The first way is less coupled; the client's known entities can be
processed independently of the client's subscriptions.  But the first way
does not help solve the appearance problem.

The second way involves collecting all IDs of all entities matching the
client's subscription.  That same information can be used for calculating both
disappearances _and_ appearances.

So the appearance problem effectively adds requirements to how we implement the
declarative strategy for deletions, but is otherwise easy to solve.

Note that the second way can be accelerated with a cache of known entities and
their filterable data.

### Fallin/Fallout Problem

The first and best strategy to fallin/fallout is the **avoidance** strategy:
just don't allow filtering on mutable columns.

For times when avoidance is not possible, I have some ideas for online
fallin/fallout:

- keep a cache: the same cache that can accelerate the appearance/disappearance
  problem can be used to solve the fallout problem.  When you see an update,
  to a record, you can look at your cache to see what the old record was, and
  you can use the old info as a way to decide which clients will want to see a
  deletion message (or one last update message showing the entity doesn't fit
  the subscription anymore, if you care to distinguish those situations).

- I suppose since it's online, you could just use the TRIGGER to pass the extra
  information instead of storing the cache, same as online deletions.
  Otherwise, you do as with the cache, where you use the old and new data to
  calculate subscriptions.

Offline fallin/fallout is effectively solved by the declarative strategy with
the implementation dicated by the Appearance problem.

## Extensibility

In no particular order.

### In-Memory Caching

A cache containing primary keys, filterable columns, and rbac data from each
row in a postgres table could speed up the initial calculations for:

- Deletions
- Disappearances
- Appearances
- Fallin
- Fallout

Admittedly, passing additional information to NOTIFY about the OLD row during a
TRIGGER could meet most of the same needs, but maybe not for the
Appearance/Disappearance problem, since that involves rbac information not
necessarily present in the row being udpated.

It's not clear to me if this extension is necessary, or how beneficial it would
be, so I think it's best to not worry about it until we know we need it.

### Avoid goroutine-per-websocket

It is idiomatic to assume one goroutine per connection (that's how Echo works)
but it's not necessarily the most efficient strategy.  Benchmarking indicated
that context switching was a major bottleneck in delivering wakeups.

The bottleneck observed in benchmarking is unlikely to be observed in current
or near-future customer scales, and other effects may dominate (like network
throughput), so further investigation should be

But [here is a good blog post](
https://www.freecodecamp.org/news/million-websockets-and-go-cc58418460bb/
) describing a setup where a single go websocket server could handle three
million websockets.  There are some differences; they have a mostly-idle server
while we have a write-heavy server.   But it's worth keeping in mind in case it
is useful in the future.

### Standalone API Server

We want to separate the REST API server into its own process some day.

That means that our streaming server must not hook into the actor system, and
should only depend on the database connection.

### Multiple Streaming Servers

The streaming server must not keep state about any streaming client beyond the
lifetime of a single webscoket connection.  That way if we ever need to set up
multiple streaming servers with a load balancer in front of them, it would
work transparently (one streaming client doesn't need to reconnect to the same
streaming server to continue streaming).
