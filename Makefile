generate:
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./api/..."
	controller-gen crd:crdVersions=v1 paths="./api/..." output:crd:dir=manifests/crd
