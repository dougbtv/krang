#!/bin/bash
set -o errexit

kubectl apply \
  -f manifests/crd/k8s.cni.cncf.io_cnimutationrequests.yaml \
  -f manifests/crd/k8s.cni.cncf.io_cnipluginregistrations.yaml \
  -f manifests/daemonset.yaml


kubectl rollout status daemonset/krangd -n kube-system --timeout=60s

kubectl create -f manifests/testing/replicaset.yml

timeout=60
interval=2
elapsed=0

while true; do
  ready=$(kubectl get rs demotuning -o jsonpath='{.status.readyReplicas}')
  desired=$(kubectl get rs demotuning -o jsonpath='{.spec.replicas}')
  if [[ "$ready" == "$desired" && "$ready" != "" ]]; then
    echo "ReplicaSet demotuning is ready"
    break
  fi
  if (( elapsed >= timeout )); then
    echo "Timed out waiting for ReplicaSet demotuning"
    exit 1
  fi
  sleep "$interval"
  elapsed=$((elapsed + interval))
done

ARP_FILTER_BEFORE=$(kubectl exec $(kubectl get pods | grep "demotuning" | head -n1 | awk '{print $1}') -- sysctl -n net.ipv4.conf.eth0.arp_filter)
echo $ARP_FILTER_BEFORE is currently the arp filter.

echo "Installing plugins..."

krangctl register --binary-path /cni-plugins/bin/tuning --cni-type tuning --name tuning --image "quay.io/dosmith/cni-plugins:v1.6.2a"
krangctl register --binary-path /usr/src/multus-cni/bin/passthru --cni-type passthru --name passthru --image "ghcr.io/k8snetworkplumbingwg/multus-cni:snapshot-thick"

echo "Waiting for all krang-install jobs to complete..."
sleep 1

TIMEOUT=180
INTERVAL=1
ELAPSED=0

plugins=("/opt/cni/bin/tuning" "/opt/cni/bin/passthru")
container_id=$(docker ps --filter "name=kind-worker2" --format "{{.ID}}")
for plugin in "${plugins[@]}"; do
  while true; do
    
    if docker exec "$container_id" test -f $plugin; then
      echo "$plugin is present."
      break
    fi
  
    if (( ELAPSED >= TIMEOUT )); then
      echo "Timed out waiting for $plugin install."
      exit 1
    fi
  
    sleep "$INTERVAL"
    ELAPSED=$((ELAPSED + INTERVAL))
  done
done

# Now lets modify...

echo "Mutating, dude."
krangctl mutate --cni-type tuning --interface eth0 --matchlabels app=demotuning --config ./manifests/testing/tuning-passthru-conf.json

echo "Sleeping for a sec..."
sleep 4

ARP_FILTER_AFTER=$(kubectl exec $(kubectl get pods | grep "demotuning" | head -n1 | awk '{print $1}') -- sysctl -n net.ipv4.conf.eth0.arp_filter)

# Check that the arp filter changed.
if [ "$ARP_FILTER_BEFORE" != "$ARP_FILTER_AFTER" ]; then
  echo "ARP filter changed from $ARP_FILTER_BEFORE to $ARP_FILTER_AFTER"
else
  echo "ARP filter did not change, still $ARP_FILTER_AFTER"
  exit 1
fi

