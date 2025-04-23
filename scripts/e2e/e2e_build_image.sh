#!/bin/sh
set -o errexit

export PATH=${PATH}:./bin

# define the OCI binary to be used. Acceptable values are `docker`, `podman`.
# Defaults to `docker`.
OCI_BIN="${OCI_BIN:-docker}"

KRANG_DOCKERFILE="${KRANG_DOCKERFILE:-Dockerfile}"

kind_network='kind'
if [ "${KRANG_DOCKERFILE}" != "none" ]; then
	$OCI_BIN build -t localhost:5000/krang:e2e -f ${KRANG_DOCKERFILE} .
fi

# load krang image from container host to kind node
kind load docker-image localhost:5000/krang:e2e

# modify the install daemonset.
sed -i -e 's|ghcr.io/dougbtv/krang:latest|localhost:5000/krang:e2e|' manifests/daemonset.yaml