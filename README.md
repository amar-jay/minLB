# minLB
minLB is a simple Load Balancer.
Just playing around it.

It uses RoundRobin algorithm to send requests into set of backends and support retries too.
It also performs active cleaning and passive recovery for unhealthy backends.

### How to?
[server.go](./server.go) to run demo servers annd [./cmd/...](./cmd/) is the actual load balancer
