#!/bin/bash

KUBE_GCE_ZONE=us-east1-b
MASTER_SIZE=n1-standard-4
NUM_NODES=50

KUBE_GCE_INSTANCE_PREFIX=stclair-heapster-scale
KUBE_ENABLE_CLUSTER_MONITORING=none

source "${KUBE_ROOT}/cluster/gce/config-default.sh"
