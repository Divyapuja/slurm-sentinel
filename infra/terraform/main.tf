terraform {
  required_providers {
    kind = {
      source  = "tehcyx/kind"
      version = "~> 0.4"
    }
  }
}

resource "kind_cluster" "sentinel" {
  name = "slurm-sentinel"
  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"
    node {
      role = "control-plane"
    }
    node {
      role = "worker"
    }
  }
}

output "kubeconfig" {
  value     = kind_cluster.sentinel.kubeconfig
  sensitive = true
}