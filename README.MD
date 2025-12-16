# Automatic Connect Session Manager for Apache Spark

## Overview

Spark Connect was introduced in Apache Spark 3.4 with the goal of decoupling the client from the Spark execution engine.
The project aim to provide a way of dynamically managing Spark Connect session via proxy, so users only need to connect to a single endpoint during their workflow.
This proxy will handle the creation, management, and termination of Spark Connect sessions as needed.

## How it works
ACSM will manage creation of new Spark Connect sessions via Session Manager.
1. Client request a new session via ACSM GRPC endpoint.
2. ACSM Session Manager will try to reuse existing idle sessions or ask spawner to create a new Spark Connect server process.
3. ACSM Spawner will launch a new Spark Connect server process and return the connection info to Session Manager.
4. Client can then connect to the Spark Connect server via ACSM Proxy endpoint.
5. ACSM Proxy will forward the GRPC requests to the appropriate Spark Connect server based on `session--id` metadata.

### Spawner Supported:
- Local Process Spawner: Launch Spark Connect server as a local process.
- Kubernetes Pod Spawner: Launch Spark Connect server as a pod in Kubernetes cluster. (TODO)
- Custom Spawner: Implement your own spawner by implementing the Spawner interface.


## Getting Started

### Prerequisites
- Go 1.20+
- Apache Spark 4.0.1+
- Try to activate python env before you run ACSM.

1. Build ACSM Spark Proxy binary
```bash
go build -o acsm-spark ./cmd/acsm-spark
```
2. Start ACSM Spark Proxy
```bash
./acsm-spark
```
3. Use ACSM GRPC endpoint to create new Spark Connect session
4. Connect to Spark Connect server via ACSM Proxy endpoint with `session-id` metadata