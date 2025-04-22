#!/bin/sh
set -o errexit

export PATH=${PATH}:./bin

# define the OCI binary to be used. Acceptable values are `docker`, `podman`.
# Defaults to `docker`.
OCI_BIN="${OCI_BIN:-docker}"

# deploy cluster with kind
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
  - role: worker
    kubeadmConfigPatches:
    - |
      kind: InitConfiguration
      nodeRegistration:
        kubeletExtraArgs:
          pod-manifest-path: "/etc/kubernetes/manifests/"
          feature-gates: "DynamicResourceAllocation=true,DRAResourceClaimDeviceStatus=true,KubeletPodResourcesDynamicResources=true"
  - role: worker
# Required by DRA Integration
##
featureGates:
  DynamicResourceAllocation: true
  DRAResourceClaimDeviceStatus: true
  KubeletPodResourcesDynamicResources: true
runtimeConfig:
  "api/beta": "true"
containerdConfigPatches:
# Enable CDI as described in
# https://github.com/container-orchestrated-devices/container-device-interface#containerd-configuration
- |-
  [plugins."io.containerd.grpc.v1.cri"]
      enable_cdi = true
##
EOF
