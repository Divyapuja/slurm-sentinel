# Slurm-Sentinel

A high-performance **Host Lifecycle Management (HLM)** and automated remediation engine designed for large-scale HPC Slurm compute farms. This repository functions as an automated fleet-reliability daemon that ingests hardware-level failure telemetry (such as PCIe link drops or unrecoverable GPU faults) and safely executes out-of-band node isolation and healing sequences.

The core objective of this project is to protect distributed MPI training workloads from cascading failures while strictly enforcing **blast-radius mitigation limits** to guarantee infrastructure Quality of Service (QoS).

---

## The Engineering Problem

In a massive GPU infrastructure fabric, multi-node distributed LLM training jobs are tightly coupled. If a single GPU on a single node suffers an unrecoverable hardware fault—such as an **NVIDIA Xid 79 error (GPU fallen off the PCIe bus)**—the entire multi-node training checkpoint crashes.

Manual intervention by an on-call SRE introduces severe latency, degrading cluster uptime and QoS. However, naive automated scripts often make the problem worse by blindly draining nodes, which can accidentally isolate healthy hardware and starve the scheduler.

`slurm-sentinel` solves this by building an end-to-end automated host-remediation loop with strict change controls, ensuring that degraded bare-metal nodes are safely isolated, drained without corrupting active job state, and targeted for out-of-band remediation.

---

## System Architecture & Deployment Model

To simulate a hybrid-cloud or on-prem control plane topology, the local testing footprint uses a decoupled design where a containerized control plane orchestrates telemetry for external mock hardware hosts.

```
                    +-----------------------------------------+
                    |           LOCAL BARE-METAL HOST         |
                    |      (Simulating Compute Farm Node)     |
                    +-----------------------------------------+
                                         |
                         (Scrapes via host.docker.internal)
                                         v
  +---------------------------------------------------------------------------------+
  | KIND KUBERNETES CONTROL PLANE                                                   |
  |                                                                                 |
  |  [ monitoring Namespace ]                                                       |
  |   +--------------------------+        Scrapes        +----------------------+   |
  |   | Prometheus Operator      |<----------------------| ServiceMonitor Hook  |   |
  |   | (Exposed on :9090)       |                       +----------------------+   |
  |   +--------------------------+                                                  |
  |                |                                                                |
  |                | Continuous Signal Evaluation                                   |
  |                v                                                                |
  |   +--------------------------+                                                  |
  |   | Sentinel Fleet Agent     |                                                  |
  |   | (Python State Machine)   |                                                  |
  |   +--------------------------+                                                  |
  +---------------------------------------------------------------------------------+
                                 |
                                 | Enforces Blast-Radius & Dispatches scontrol
                                 v
                 +-------------------------------+
                 | Custom Go Hardware Exporter   |
                 | (Serves Telemetry on :8080)   |
                 +-------------------------------+

```

* **Infrastructure as Code (IaC):** The local infrastructure layer is fully standardized and provisioned via Terraform utilizing the `tehcyx/kind` provider to ensure deterministic cluster states.
* **Orchestration Control Plane:** A 2-node local Kubernetes cluster via `kind` (1 control plane, 1 dedicated worker) representing the high-availability management infrastructure.
* **E2E Observability Fabric:** Powered by the CoreOS `kube-prometheus-stack` operator to simulate production-grade log and metric collection.
* **High-Throughput Telemetry Engine:** A custom, multi-threaded Go metrics daemon designed for low memory overhead, leveraging slice preallocation (`prealloc`) to maintain a zero-alloc footprint during high-frequency telemetry scrapings.

---

## Current Local Verification Status

The local development stack is initialized, validated, and currently running across the following ports:

* **Infrastructure Layer (`/infra/terraform`):** Successfully converged. `kind get clusters` and `kubectl get nodes -n monitoring` confirm a healthy, multi-node control plane.
* **Go Telemetry Daemon (`/cmd/exporter`):** Live and serving raw telemetry at **`localhost:8080/metrics`**. The background process successfully tracks and registers custom hardware-level metrics including `slurm_node_status` and `slurm_gpu_xid_errors_total`.
* **Prometheus Observability Engine:** The time-series database and UI are live at **`localhost:9090`**, actively verifying target configurations.

---

## Automated Fleet Guardrails & Linting

Operating with "operational excellence" means catching failure modes before they reach production. This repository enforces a strict `.pre-commit-config.yaml` pipeline designed to catch infrastructure anti-patterns:

* **Telemetry Leak Prevention:** Utilizes `gitleaks` to guarantee that no cluster access keys, internal host strings, or cloud tokens are ever accidentally committed.
* **Systems-Level Go Optimization:** Enforces checks via `golangci-lint` (including `bodyclose` to prevent HTTP connection leaks and `prealloc` to minimize garbage collection latency in the metrics pipeline).
* **Python Code Quality:** Leverages `ruff` to ensure the automated healing agent uses optimized iteration loops and standard system execution calls.

---

## Engineering Roadmap (Immediate Next Steps)

The baseline architecture is running locally on `:8080` and `:9090`. The immediate execution steps focus on establishing tight integration with the scheduler logic and designing for hardware failure domains:

1. **Telemetry Topology Bridging:** Finalize the deployment manifests in `infra/k8s/exporter.yaml`. Configure the cluster's Prometheus instance to discover the host-level Go exporter using `host.docker.internal:8080` via a dedicated `ServiceMonitor`.
2. **Idempotent Host State Machine (`sentinel/agent.py`):** Code the remediation engine loop to poll Prometheus. The agent must evaluate telemetry over a moving time window and interface with mock Slurm states (`IDLE`, `ALLOCATED`, `DRAIN`) to ensure actions are idempotent.
3. **Strict Blast-Radius Controls:** Program an algorithmic circuit-breaker inside the Python agent. If more than 5% of the total cluster fleet reports an error simultaneously, the agent must freeze automated remediation and escalate to prevent a cascading network-wide drain event.
4. **Fault Injection & Automated Runbooks:** Write a robust project `Makefile` implementing a `make inject-fault` target. This will simulate an unrecoverable GPU Bus drop (Xid 79) via an HTTP POST endpoint on the Go exporter, demonstrating the full telemetry-to-remediation pipeline in real time.