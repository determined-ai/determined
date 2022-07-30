.. _elasticsearch-logging-backend:

##############################
 Elasticsearch-backed logging
##############################

This guide covers the limitations of the default logging backend, as a guideline on when to migrate
to Elasticsearch, and some tips for how to tune Elasticsearch to work best with Determined.

`Elasticsearch <https://www.elastic.co/what-is/elasticsearch>`__ is a search engine commonly used
for storing application logs for search and analytics. Determined supports using Elasticsearch as
the storage backend for task logs. Configuring Determined to use Elasticsearch is simple; however,
managing an Elasticsearch cluster at scale is an involved task, so this guide is recommended for
users who have hit the limitations of the default logging backend.

Using the default logging backend, with a standard deployment using ``det deploy``, the cluster can
ingest logs about as fast as Postgres can persist them. For example, with ``det deploy aws`` using
Aurora Serverless with 2 capacity units, ingestion speed maxes out around 10-15 MB/s (where the
database's CPU hits ~90%). To get a little more mileage from the default, we recommend increasing
the capacity of the database. At a certain point, the master instance itself will become the
bottleneck, since it has limited incoming network bandwidth for HTTP requests delivering logs and
limited resources to process them. The master instance size can be increased, but vertical scaling
is likely to be limited to a log throughput of around hundreds of megabytes per second; we recommend
moving to Elasticsearch to get past that limit.

Determined offers some additional recommendations for the Elasticsearch cluster configuration based
on how the cluster will be used:

-  Tune the default shards per index to your expected throughput (or use `index templates
   <https://www.elastic.co/guide/en/elasticsearch/reference/7.10/index-templates.html>`__).
   Determined ships logs in Logstash format rolling over to a new index each day. Depending on your
   log volume, the default number of shards could be too high or too low. The general rule of thumb
   is not to exceed 50 GB per shard while minimizing the number of shards per index. For
   high-utilization clusters, this may entail increasing the shards per index and rotating indices
   older than a few months out of the cluster periodically, to avoid the overhead accumulated from
   having too many shards. A more in-depth guide can be found `here
   <https://www.elastic.co/guide/en/elasticsearch/reference/current/size-your-shards.html>`__.

-  Though it may increase latency for end users, increasing the `refresh interval
   <https://www.elastic.co/guide/en/elasticsearch/reference/master/tune-for-indexing-speed.html#_unset_or_increase_the_refresh_interval>`__
   may help increase total throughput.

-  Apply the following `index template
   <https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-templates-v1.html>`__ to
   optimize the mappings in Determined log indices for ingest speed. This turns off analysis and in
   some cases indexing on properties for which Determined does not use these features.

.. code:: json

   {
     "index_patterns": ["determined-tasklogs-*"],
     "mappings": {
       "properties": {
           "task_id": {"type": "keyword", "index": true},
           "allocation_id": {"type": "keyword": "index": true},
           "agent_id": {"type": "keyword", "index": true},
           "container_id": {"type": "keyword", "index": true},
           "level": {"type": "keyword", "index": true},
           "log": {"type": "text", "index": false},
           "message": {"type": "text", "index": false},
           "source": {"type": "keyword", "index": true},
           "stdtype": {"type": "keyword", "index": true}
       }
     }
   }

The configuration settings to enable Elasticsearch as the task log backend are described in the
:ref:`cluster configuration <cluster-configuration>` reference.
