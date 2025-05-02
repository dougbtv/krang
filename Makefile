REMOTE_HOST=192.168.122.138
VIRTHOST_PROXY=ssh -W %h:%p root@192.168.50.200
PROXY_COMMAND=-o ProxyCommand="$(VIRTHOST_PROXY)"
HOME_REMOTE_PATH=/home/fedora/
KRANG_REMOTE_PATH=/var/lib/krangd/
SSH_USER := fedora@192.168.122.138
KRANG_IMAGE=ghcr.io/dougbtv/krang:latest

generate:
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./api/..."
	controller-gen crd:crdVersions=v1 paths="./api/..." output:crd:dir=manifests/crd

.PHONY: test
test:
	ginkgo -r ./controllers

.PHONY: image
image:
	docker build -t $(KRANG_IMAGE) -f Dockerfile .

.PHONY: push
push:
	docker push $(KRANG_IMAGE)

krangd-dev:
	# Run the krangd target
	$(MAKE) krangd
	$(MAKE) krangd-remote-copy
	$(MAKE) krangd-kind-copy

build:
	$(MAKE) krangd
	$(MAKE) krangctl

krangd:
	GOARCH=amd64 CGO_ENABLED=0 go build -o bin/krangd ./cmd/krangd

krangctl:
	GOARCH=amd64 CGO_ENABLED=0 go build -o bin/krangctl ./cmd/krangctl

krangctl-dev:
	$(MAKE) krangctl
	scp $(PROXY_COMMAND) bin/krangctl fedora@$(REMOTE_HOST):$(HOME_REMOTE_PATH)krangctl

krangd-kind-copy:
	@echo "Copying krangd into Kind nodes..."
	@ssh $(PROXY_COMMAND) $(SSH_USER) '\
		for node in $$(kind get nodes); do \
			echo "💾 Copying into $$node..."; \
			docker exec $$node mkdir -p $(KRANG_REMOTE_PATH); \
			docker cp $$HOME/krangd $$node:$(KRANG_REMOTE_PATH)/krangd; \
			docker exec $$node chmod +x $(KRANG_REMOTE_PATH)/krangd; \
		done'
	@echo "✅ Done with Kind node deploy."

krangd-remote-copy:
	@echo "SCP-ing krangd to remote dev box..."
	scp $(PROXY_COMMAND) bin/krangd fedora@$(REMOTE_HOST):$(HOME_REMOTE_PATH)krangd
	scp $(PROXY_COMMAND) manifests/krangd-dev-daemonset.yml fedora@$(REMOTE_HOST):$(HOME_REMOTE_PATH)
	scp $(PROXY_COMMAND) manifests/crd/* fedora@$(REMOTE_HOST):$(HOME_REMOTE_PATH)
	@echo "Done with remote deploy."
