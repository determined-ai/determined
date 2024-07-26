# Remote K8s Dev

## Reverse Proxy

- The firewall needs to have the configured port range open.
- There is a shared server prepared for engineers to use.

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
