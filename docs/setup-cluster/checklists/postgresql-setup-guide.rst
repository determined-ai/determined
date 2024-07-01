.. _postgresql-database-tuning:

############################
 PostgreSQL Database Tuning
############################

When setting up PostgreSQL for your deployment, it's crucial to configure several key parameters to
ensure optimal performance. Below are the most important settings to consider:

#. **max_connections**: Ensure ``max_connections`` is set to at least 96.
#. **shared_buffers**: Set ``shared_buffers`` to approximately 25% of total system memory, adjust
   based on system workloads.
#. **max_wal_size and min_wal_size**: Adjust ``max_wal_size`` and ``min_wal_size`` according to your
   usage patterns for improved performance.

For more details, visit the relevant section below.

*****************
 max_connections
*****************

``max_connections`` is the most important and universal parameter to adjust. We recommend confirming
that ``max_connections`` is set to at least 96. This setting ensures that your database can handle
the number of concurrent connections required for Determined.

****************
 shared_buffers
****************

The ``shared_buffers`` setting determines how much memory PostgreSQL uses for caching data. It is
generally recommended to set this to around 25% of your system's total memory. However, you should
adjust this based on other workloads running on the same system.

*******************************
 max_wal_size and min_wal_size
*******************************

The Write-Ahead Logging (WAL) settings, ``max_wal_size`` and ``min_wal_size``, are more dependent on
your specific usage patterns. Increasing these values can help improve performance for larger
deployments. For detailed information on configuring these settings, please refer to the `WAL
Configuration <https://www.postgresql.org/docs/current/wal-configuration.html>`__ page or the
`PostgreSQL Runtime Configuration
<https://www.postgresql.org/docs/current/runtime-config-resource.html>`__ page.

Properly configuring these settings will help you achieve optimal performance and reliability with
your PostgreSQL deployment.
