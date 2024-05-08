:orphan:

**Improvements**

Kubernetes: Add Determined resource information such as `workspace` and `task ID` as pod labels. This improvement facilitates better resource tracking and management within Kubernetes environments.

Configuration: Introduce a DCGM Helm chart and Prometheus configuration to the `tools/observability` directory. Additionally, two new dashboards, "API Monitoring" and "Resource Utilization", have been added to improve observability and operational insight.