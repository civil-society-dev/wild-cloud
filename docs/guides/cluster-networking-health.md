# Cluster Networking Health Checklist

Verifying every item on this list confirms the full networking stack is functioning correctly, from node-level overlay through DNS to external ingress.

## Node Layer

1. **All nodes Ready** — no cordons, no taints (e.g. `maintenance:NoExecute`)
2. **Flannel pods running on every node** — stale VXLAN tunnels break cross-node pod traffic
3. **Cross-node pod connectivity** — pods on each worker can reach pods on every other node

## Service Routing

4. **kube-proxy pods running on every node** — nftables rules route ClusterIP traffic to pod endpoints
5. **CoreDNS pods running and resolving** — both cluster-internal names (`*.svc.cluster.local`) and external names
6. **CoreDNS upstream reachability** — Talos DNS proxy at `169.254.116.108` responding from all nodes

## Load Balancing

7. **MetalLB speakers running on all nodes** — L2 ARP announcements for LoadBalancer IPs
8. **MetalLB ServiceL2Status resources valid** — `status.node` matches actual pod placement (stale entries block announcements)
9. **LoadBalancer IPs reachable** — Traefik LB IP responds from LAN

## Ingress & Security

10. **Traefik ingress routing** — forwards to backend services, TLS termination working
11. **CrowdSec LAPI running** — can reach `api.crowdsec.net` (depends on CoreDNS external resolution)
12. **CrowdSec bouncer registered with LAPI** — unregistered bouncer blocks all forwardAuth requests

## Storage

13. **Longhorn managers running on all workers** — enables volume replica scheduling and rebuilds
14. **Longhorn volume replicas healthy** — all volumes at target replica count across nodes

## LAN DNS

15. **dnsmasq on Wild Central** — resolves LAN-local domains to correct LoadBalancer IPs (hairpin NAT)
