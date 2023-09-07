.. _elasticsearch-logging-backend:

##############################
 Elasticsearch-backed logging
##############################

It is possible to use Elasticsearch as an alternative to the default logging backend. In the past,
such configuration was recommended for larger installations, but it's no longer the case.

Example configuration:

.. code:: yaml

   logging:
   type: elastic
   host: "elastic.example.com"
   port: 443
   security:
      username: "elastic-user"
      password: "mypassword"
      tls:
      enabled: true

The configuration settings to enable Elasticsearch as the task log backend are described in the
:ref:`cluster configuration <cluster-configuration>` reference.

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

