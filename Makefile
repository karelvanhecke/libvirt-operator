VERSION?=sha-$(shell git rev-parse HEAD)

# renovate: datasource=github-releases depName=kubernetes-sigs/kind versioning=semver
KIND_VERSION=v0.27.0
# renovate: datasource=github-releases depName=kubernetes-sigs/controller-tools versioning=semver
CONTROLLER_GEN_VERSION=v0.18.0
# renovate: datasource=github-releases depName=kubernetes/kubernetes versioning=semver
KUBECTL_VERSION=v1.32.4
# renovate: datasource=docker depName=kindest/node versioning=semver
KIND_IMAGE_VERSION=v1.32.3@sha256:b36e76b4ad37b88539ce5e07425f77b29f73a8eaaebf3f1a8bc9c764401d118c
# renovate: datasource=github-releases depName=cert-manager/cert-manager versioning=semver
CERT_MANAGER_VERSION=v1.17.2
HEADERFILE=./hack/boilerplate.go.txt
ROLENAME=libvirt-operator

.PHONY: install-tools
install-tools: install-kind install-controller-gen install-kubectl

.PHONY: install-kind
install-kind:
	@go install sigs.k8s.io/kind@$(KIND_VERSION)

.PHONY: install-controller-gen
install-controller-gen:
	@go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)

.PHONY: install-kubectl
install-kubectl:
	@mkdir -p ~/.local/bin && curl -o /tmp/kubectl -L https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/linux/amd64/kubectl && \
		sha256sum /tmp/kubectl | grep "$(curl -L https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/linux/amd64/kubectl.sha256)" && \
		chmod +x /tmp/kubectl && mv /tmp/kubectl ~/.local/bin

.PHONY: generators
generators: gen-object gen-crd gen-rbac

.PHONY: gen-object
gen-object:
	@controller-gen object:headerFile=$(HEADERFILE) paths=./api/...

.PHONY: gen-crd
gen-crd:
	@controller-gen crd paths=./... output:crd:artifacts:config=./install/base/crd

.PHONY: gen-rbac
gen-rbac:
	@controller-gen rbac:roleName=$(ROLENAME) paths=./... output:rbac:artifacts:config=./install/base

.PHONY: build
build:
	@CGO_ENABLED=0 go build ./...

.PHONY: test
test:
	@CGO_ENABLED=0 go test ./...

.PHONY: build-binary
build-binary:
	@CGO_ENABLED=0 go build -o bin/operator \
		-ldflags "-s -w -X 'github.com/karelvanhecke/libvirt-operator/internal/version.version=$(VERSION)'" \
		-trimpath \
		github.com/karelvanhecke/libvirt-operator/cmd/operator

.PHONY: build-container
build-container:
	@docker buildx build \
		--build-arg VERSION=$(VERSION) \
		--load \
		-t ghcr.io/karelvanhecke/libvirt-operator:$(VERSION) .

.PHONY: create-kind-cluster
create-kind-cluster:
	@kind create cluster --name operator-dev --image docker.io/kindest/node:${KIND_IMAGE_VERSION} && \
		kubectl apply --context kind-operator-dev -f https://github.com/cert-manager/cert-manager/releases/download/$(CERT_MANAGER_VERSION)/cert-manager.yaml && \
		kubectl -n cert-manager wait --timeout=60s --for=condition=Available=true deployment --all

.PHONY: deploy-to-kind-cluster
deploy-to-kind-cluster:
	@kind load docker-image --name operator-dev ghcr.io/karelvanhecke/libvirt-operator:$(VERSION) && \
		mkdir -p ./install/development && \
		TAG=$(VERSION) envsubst < ./hack/kustomization.yaml.txt > ./install/development/kustomization.yaml && \
		kubectl apply --context kind-operator-dev -k ./install/development && \
		kubectl -n libvirt-operator wait --for=condition=Available=true deployment/libvirt-operator

.PHONY: delete-kind-cluster
delete-kind-cluster:
	@kind delete cluster --name operator-dev
