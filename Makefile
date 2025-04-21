KRANGD_HOST=192.168.122.138
KRANGD_PROXY=ssh -W %h:%p root@192.168.50.200
HOME_REMOTE_PATH=/home/fedora/
KRANG_REMOTE_PATH=/var/lib/krangd/
SSH_PROXY := ssh -W %h:%p root@192.168.50.200
SSH_USER := fedora@192.168.122.138

define ssh_krangd
	ssh $(SSH_PROXY) $(SSH_USER) '$(1)'
endef

generate:
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./api/..."
	controller-gen crd:crdVersions=v1 paths="./api/..." output:crd:dir=manifests/crd

.PHONY: test
test:
	ginkgo -r ./controllers

krangd-dev:
	# Run the krangd target
	$(MAKE) krangd
	$(MAKE) krangd-remote-copy
	$(MAKE) krangd-kind-copy

krangd:
	GOARCH=amd64 CGO_ENABLED=0 go build -o bin/krangd ./cmd/krangd

krangctl:
	GOARCH=amd64 CGO_ENABLED=0 go build -o bin/krangctl ./cmd/krangctl
	scp -o ProxyCommand="$(KRANGD_PROXY)" bin/krangctl fedora@$(KRANGD_HOST):$(HOME_REMOTE_PATH)krangctl

krangd-kind-copy:
	@echo "📦 Copying krangd into Kind nodes..."
	@ssh -o ProxyCommand="$(SSH_PROXY)" $(SSH_USER) '\
		for node in $$(kind get nodes); do \
			echo "💾 Copying into $$node..."; \
			docker exec $$node mkdir -p $(KRANG_REMOTE_PATH); \
			docker cp $$HOME/krangd $$node:$(KRANG_REMOTE_PATH)/krangd; \
			docker exec $$node chmod +x $(KRANG_REMOTE_PATH)/krangd; \
		done'
	@echo "✅ Done with Kind node deploy."

krangd-remote-copy:
	@echo "🌐 SCP-ing krangd to remote dev box..."
	scp -o ProxyCommand="$(KRANGD_PROXY)" bin/krangd fedora@$(KRANGD_HOST):$(HOME_REMOTE_PATH)krangd
	scp -o ProxyCommand="$(KRANGD_PROXY)" manifests/krangd-dev-daemonset.yml fedora@$(KRANGD_HOST):$(HOME_REMOTE_PATH)
	@echo "✅ Done with remote deploy."
