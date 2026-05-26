package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	slurmNodeStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "slurm_node_status",
			Help: "Slurm node status (1 = Idle, 2 = Alloc, 3 = Down/Drain).",
		},
		[]string{"node_name"},
	)
	gpuXidErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "slurm_gpu_xid_errors_total",
			Help: "Total count of NVIDIA GPU Xid critical errors.",
		},
		[]string{"node_name", "xid"},
	)
)

func init() {
	prometheus.MustRegister(slurmNodeStatus)
	prometheus.MustRegister(gpuXidErrors)
}

func main() {
	// Seed initial mock data
	slurmNodeStatus.WithLabelValues("compute-node-01").Set(2) // Allocated
	slurmNodeStatus.WithLabelValues("compute-node-02").Set(1) // Idle

	// Simulate real-time metric updates/faults in the background
	go func() {
		for {
			time.Sleep(10 * time.Second)
			// Randomly simulate a GPU fault on node-01 every now and then
			if rand.Float64() > 0.7 {
				gpuXidErrors.WithLabelValues("compute-node-01", "79").Inc() // Xid 79: GPU fallen off bus
				slurmNodeStatus.WithLabelValues("compute-node-01").Set(3)  // Degraded/Drain
				log.Println("Alert: Simulated Xid 79 error on compute-node-01")
			}
		}
	}()

	log.Println("Slurm Sentinel Exporter listening on :8080...")
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8080", nil))
}