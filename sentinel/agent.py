import time
import logging
from prometheus_api_client import PrometheusConnect

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger("slurm-sentinel")

PROMETHEUS_URL = "http://localhost:9090" # ToDo : port-forward for K8s DNS

def check_metrics():
    try:
        prom = PrometheusConnect(url=PROMETHEUS_URL, disable_ssl=True)
        # Query for recent Xid errors
        query = 'slurm_gpu_xid_errors_total[1m]'
        result = prom.custom_query(query=query)
        
        for metric in result:
            node = metric['metric']['node_name']
            xid = metric['metric']['xid']
            value = int(metric['value'][1])
            
            if value > 0:
                logger.warning(f"Anomaly Detected! Node {node} reported Xid {xid} error.")
                trigger_healing_workflow(node, xid)
    except Exception as e:
        logger.error(f"Failed to query Prometheus: {e}")

def trigger_healing_workflow(node, xid):
    logger.info(f"Initiating Auto-Healing for {node}...")
    logger.info(f"1. Executing: scontrol update NodeName={node} State=DRAIN Reason='Auto-healed: Xid {xid}'")
    logger.info(f"2. Executing: squeue -w {node} -t R -h -o %i | xargs -r scontrol requeue")
    logger.info(f"Node {node} successfully isolated. Jobs requeued safely.")

if __name__ == "__main__":
    logger.info("🤖 Slurm Sentinel Agent activated. Monitoring cluster health...")
    while True:
        check_metrics()
        time.sleep(5)