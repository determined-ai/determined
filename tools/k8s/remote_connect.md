# Remote K8s Dev

This document describes the remote dev tooling for connecting devcluster to a kubernetes
cluster running in the cloud.

Overview of how this works:
- set up reverse proxy
    - port collision
    - err handling
    - clean up
- ensure there is a Gateway running the target cluster
    - launch if necessary
    - grab its configuration
- updated the given devcluster config based on the collected and launched services
- run devcluster

## Configuring the tool

This tool can be configured via a configuration file which can be passed in using the `--config or -c` flag. The same arguments can also be passed in directly via CLI in the kabob-case format. CLI arguments take precedence over the configuration file.

## Reverse Proxy Server
A public facing server is used as a reverse proxy to forward traffic to the internal Determined master running on your local machine to make it accessible to Determined workloads running on the target cluster.

- Ensure that the firewall is configured to allow traffic on the given port range.
- A shared server might be available for engineers to use for reverse proxy purposes.

### SSH Server Configuration
```ini
# Enable TCP forwarding to allow reverse proxy setups.
AllowTcpForwarding yes

# Make the forwarded ports accessible from any IP address.
# Change to 'no' if you want to restrict to localhost or a specific IP.
GatewayPorts yes

# Disable root login over SSH for security reasons.
PermitRootLogin no

# Set a reasonable limit for maximum concurrent SSH sessions to prevent abuse.
MaxSessions 20

# Optional: Enable compression if bandwidth is a concern. This may increase CPU usage.
# Compression yes

# KeepAlive settings to maintain the SSH connection and detect issues.
ClientAliveInterval 60  # Time in seconds for sending keepalive messages to the client.
ClientAliveCountMax 10  # Number of keepalive messages sent without receiving any message back from the client.
```

## TODO
- Determine how to handle multiple gateways on the cluster.
- Develop a cleanup protocol for gateways if they are dynamically provisioned.
- Can we migrate some of these tasks into the `devcluster` setup stages? It would need dynamic configuration of next stages.
