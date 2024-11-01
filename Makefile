OPERATOR_VERSION=v0.1.0-dev
KIND_VERSION=v0.24.0
CONTROLLER_GEN_VERSION=v0.16.5
KUBECTL_VERSION=v1.31.2
HEADERFILE=./hack/boilerplate.go.txt
ROLENAME=libvirt-operator

.PHONY: install-kind install-controller-gen install-kubectl gen-object gen-crd gen-rbac

install-tools: install-kind install-controller-gen install-kubectl

install-kind:
	@go install sigs.k8s.io/kind@$(KIND_VERSION)

install-controller-gen:
	@go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)

install-kubectl:
	@mkdir -p ~/.local/bin && curl -o /tmp/kubectl -L https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/linux/amd64/kubectl && \
		sha256sum /tmp/kubectl | grep "$(curl -L https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl.sha256)" && \
		chmod +x /tmp/kubectl && mv /tmp/kubectl ~/.local/bin

gen-object:
	@controller-gen object:headerFile=$(HEADERFILE) paths=./api/...

gen-crd:
	@controller-gen crd paths=./... output:crd:artifacts:config=./install/base/crd

gen-rbac:
	@controller-gen rbac:roleName=$(ROLENAME) paths=./... output:rbac:artifacts:config=./install/base

build:
	@CGO_ENABLED=0 go build ./...

build-binary:
	@CGO_ENABLED=0 go build -o bin/operator \
		-ldflags "-s -w -X 'github.com/karelvanhecke/libvirt-operator/internal/version.version=$(OPERATOR_VERSION)' \
		-X 'github.com/karelvanhecke/libvirt-operator/internal/version.buildDate=$(shell date --iso-8601=s)' \
		-X 'github.com/karelvanhecke/libvirt-operator/internal/version.buildCommit=$(shell git rev-parse HEAD)'" \
		github.com/karelvanhecke/libvirt-operator/cmd/operator

build-container:
	@docker build -t ghcr.io/karelvanhecke/libvirt-operator:${OPERATOR_VERSION} .

setup-kind:
	@kind create cluster --name operator-dev && \
		kind load docker-image --name operator-dev ghcr.io/karelvanhecke/libvirt-operator:${OPERATOR_VERSION} && \
		kubectl apply --context kind-operator-dev -k install/development

cleanup-kind:
	@kind delete cluster --name operator-dev
