# Streaming Updates Design Doc

## Glossary

### Record

Some state.  Basically a row in a database table.  Every record has a sequence
number.

### Sequence Number

A postgres sequence, applied to each record in a single table.  The sequence
number for a record is set when it is first created, and set again on each
update.  That means the sequence number for a record corresponds to how new
that record is within its table.

The sequence number of records in one table are not related to the sequence
numbers of other tables.

### Event

Something happens server-side.  Various kinds of events exist:

- Insertion
- Update
- Deletion
- Appearance
- Disappearance
- Fallin
- Fallout

### Message

Something sent to a streaming client.  There are only two kinds of messages:

- Record: works like an upsert
- Deletion: just the primary key of the record deleted

The server/client protocol is intentionally simple and declarative.
Additionally, a client shouldn't be able to tell the difference between a
deletion and a disappearance, in the same way that it can't tell the difference
between 404 due to nonexistence vs 404 due to RBAC.

### Streaming Client

An entity which connects over websocket to the server, and subscribes to some
set of states it wants to receive events for.

Streaming clients are assumed to be stateful and robust to disconnections.  A
streaming client will assume that it can reconnect with the same set of
subscriptions and the last known sequence number(s) seen before
disconnecting, and expect to be able to pick up streaming where it left off.

### Online vs Offline

"Online" and "offline" are descriptions of events.  "Online" means the event
occured while the streaming client was connected.  "Offline" means the client
wasn't connected at the time.

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

### Fallin/Fallout

A Fallin is when a record changes in a way that it begins to meet the
requirements of the subscription; it "falls in" to the subscribed set.

A Fallout is when it changes to no longer be in the subscription set.

An example of Fallin this would be a streaming client subscribed to all
experiments matching workspace\_id==1, and one experiment is moved into that
workspace from another.  Fallout would be if that same experiment were then
moved back to its original workspace.

Fallin/Fallout are only possible when server-side filtering is allowed on
mutable fields.  Streaming all trials with experiment\_id==1 can't have fallout
for instance, because trials cannot be moved from one experiment to another.

Also, since fallin/fallout involve modifying the record, the record's sequence
number should be modified.  Therefore, fallin cases are most likely handled by
the same logic that handles updates: a record gets updated and broadcast to all
relevant subscribers, which might include some new ones.

So it turns out that only fallout presents a problem; fallin is most likley
solved by normal Update handling.

## Problems

### RBAC problem

Obviously streaming must take RBAC into account.

Note that the presence of RBAC and the behaviors of RBAC are generally so
custom to our system that if we incorporate any 3rd party library to help us
with streaming updates, we must implement the RBAC portion in a customized way.

Generally speaking, this level of required customization renders most 3rd party
solutions unhelpful.

### Deletion problem

Online deletions are straightforward in concept.  A row that is deleted needs
to be passed to streaming clients who have subscribed to and permission to view
that deletion.  There are two basic strategies for accomplishing this:

- the **NOTIFY-queue** strategy: use the TRIGGER to pass filterable information
  about the OLD row via NOTIFY.  Broadcast events to subscriptions matching
  either the NEW or the OLD information.  Note that this makes NOTIFY
  queue-like rather than a flag-like.  In postgres, NOTIFY channels can fill up
  to 8GB of memory before they start to block, so it's likely that the
  queue-like behavior doesn't slow down the system until we hit a much larger
  scale.

  Note that when the TRIGGER runs, we must be able to gather the rbac ownership
  information for the OLD row (which is being deleted).  So long as then
  ownership info is either a field in the deleted record, or there is a foreign
  key relationship between this table and the table with the ownership info,
  that should not present any issue.

- the **cache** strategy: if you keep a cache containing the primary key, the
  filterable attributes, and the rbac ownership information, then you can emit
  just the primary key of the OLD record via NOTIFY, and no additional lookups
  need to run in the TRIGGER.  Note that this still makes NOTIFY queue-like,
  but the size of the messages in the queue would be tiny.  There is obviously
  a memory cost to this cache.

Without considering any other problems, the NOTIFY-queue strategy is a clear
winner here.

Offline deletions are more challenging.  There are generally three strategies:

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

#### Just Boot'em Variation: invalidate the client cache

Should just boot'em also invalidate the client cache?

Pros:
  - if just boot'em invalidates client cache, then even the offline appearence
    and disappearance problems go away.  Removes one requirement for the
    declarative strategy.

Cons:
  - It requires restreaming a lot more state at rbac transitions.  If the
    offline appearance and disappearence problem is solved passively (due to
    declarative strategy) it would seem like a pointless performance hit.

Conclusion: there's really no upside to invalidating the client cache, so we
won't.

#### Just Boot'em Variation: Boot Everybody

Should just boot'em boot everybody after every rbac change?

Pros:
  - Figuring out exactly who to boot is hard, especially when the query to get
    workspace access information involves about five different joins, but
    booting everybody is really easy and really correct.

Cons:
  - There is high computation cost to having every client connect at the same
    time, but it's rare, and likely on-par with the steady-state constant
    polling that we do right now.

Conclusion: we will boot everybody on every rbac change now, because it's
simple, easy, and correct.  We'll optimize later, if necessary.

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

### Fallout Problem

Unlike Appearance/Disappearance problems, it should never be possible for
fallin/fallout happen without a sequence number bump on the record in question.

As mentioned before, that means that fallin is probably not a problem at all;
most likely the subscription matching logic for handling updates will passively
solve both the offline and online fallin cases.

Fallout is still interesting.  The first and best strategy to fallout is the
**avoidance** strategy: just don't allow filtering on mutable columns.
Unfortunately, it's unlikely we can avoid fallout entirely.

Generally speaking, online fallout can be solved by pushing updates to
subscriptions which match either the old or new record when a record is
updated.  The strategies for accomplishing this are nearly identical to the
online deletion strategies:

- the **NOTIFY-queue** strategy: as with online deletions, use the TRIGGER to
  pass filterable information about the OLD row via NOTIFY.

- the **cache** strategy: as with online deletions, rely on an in-memory cache
  of primary keys, filterable info, and rbac ownership info.  Stream fallout
  events wherever a subscription matches the old record in the cache.

  There is a slight difference here from the online deletions case, which is
  that you can fully restore the flag-like NOTIFY behavior, since you don't
  need NOTIFY to pass information that is being deleted from the database.

For offline fallout, you'd either need to have an event log of transitions
(:puke:) or you'd have to use the declarative strategy.  The declarative
strategy effectively lets you leverage the client's cache to calculate offline
fallout without having to store historical states in the streaming server.

## Extensibility Ideas

In no particular order.

### In-Memory Caching

A cache containing primary keys, filterable columns, and rbac data from each
row in a postgres table could speed up the initial calculations of the
declarative strategy.

Such a cache could also make the NOTIFY-queue strategy unnecessary for online
problems, and less important for the online deletion cases (though we'd still
probably want to send the primary key being deleted for online deletions).

However, the memory cost could be quite large, and the performance benefits
might not justify the memory cost, so I think it's best to not worry about it
until we know we need it.

### Avoid goroutine-per-websocket

It is idiomatic to assume one goroutine per connection (that's how Echo works)
but it's not necessarily the most efficient strategy.  Benchmarking indicated
that context switching was a major bottleneck in delivering wakeups.

The bottleneck observed in benchmarking is unlikely to be observed in current
or near-future customer scales, and other effects may dominate (like network
throughput), so further investigation should be done before engaging in this
work.

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
