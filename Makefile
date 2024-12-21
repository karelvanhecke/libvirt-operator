OPERATOR_VERSION?=dev
# renovate: datasource=github-releases depName=kubernetes-sigs/kind versioning=semver
KIND_VERSION=v0.25.0
# renovate: datasource=github-releases depName=kubernetes-sigs/controller-tools versioning=semver
CONTROLLER_GEN_VERSION=v0.16.5
# renovate: datasource=github-releases depName=kubernetes/kubernetes versioning=semver
KUBECTL_VERSION=v1.31.4
# renovate: datasource=docker depName=kindest/node versioning=semver
KIND_IMAGE_VERSION=v1.31.2@sha256:18fbefc20a7113353c7b75b5c869d7145a6abd6269154825872dc59c1329912e
HEADERFILE=./hack/boilerplate.go.txt
ROLENAME=libvirt-operator

.PHONY: install-tools install-kind install-controller-gen install-kubectl generators gen-object gen-crd gen-rbac

install-tools: install-kind install-controller-gen install-kubectl

install-kind:
	@go install sigs.k8s.io/kind@$(KIND_VERSION)

install-controller-gen:
	@go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)

install-kubectl:
	@mkdir -p ~/.local/bin && curl -o /tmp/kubectl -L https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/linux/amd64/kubectl && \
		sha256sum /tmp/kubectl | grep "$(curl -L https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/linux/amd64/kubectl.sha256)" && \
		chmod +x /tmp/kubectl && mv /tmp/kubectl ~/.local/bin

generators: gen-object gen-crd gen-rbac

gen-object:
	@controller-gen object:headerFile=$(HEADERFILE) paths=./api/...

gen-crd:
	@controller-gen crd paths=./... output:crd:artifacts:config=./install/base/crd

gen-rbac:
	@controller-gen rbac:roleName=$(ROLENAME) paths=./... output:rbac:artifacts:config=./install/base

build:
	@CGO_ENABLED=0 go build ./...

test:
	@CGO_ENABLED=0 go test ./...

build-binary:
	@CGO_ENABLED=0 go build -o bin/operator \
		-ldflags "-s -w -X 'github.com/karelvanhecke/libvirt-operator/internal/version.version=$(OPERATOR_VERSION)' \
		-X 'github.com/karelvanhecke/libvirt-operator/internal/version.buildDate=$(shell date --iso-8601=s)' \
		-X 'github.com/karelvanhecke/libvirt-operator/internal/version.buildCommit=$(shell git rev-parse HEAD)'" \
		github.com/karelvanhecke/libvirt-operator/cmd/operator

build-container:
	@docker build --build-arg OPERATOR_VERSION=$(OPERATOR_VERSION) -t ghcr.io/karelvanhecke/libvirt-operator:$(OPERATOR_VERSION) .

create-kind-cluster:
	@kind create cluster --name operator-dev --image docker.io/kindest/node:${KIND_IMAGE_VERSION}

deploy-to-kind-cluster:
	@kind load docker-image --name operator-dev ghcr.io/karelvanhecke/libvirt-operator:$(OPERATOR_VERSION) && \
		mkdir -p ./install/development && \
		OPERATOR_VERSION=$(OPERATOR_VERSION) envsubst < ./hack/kustomization.yaml.txt > ./install/development/kustomization.yaml && \
		kubectl apply --context kind-operator-dev -k ./install/development && \
		kubectl -n libvirt-operator wait --for=condition=Available=true deployment/libvirt-operator

delete-kind-cluster:
	@kind delete cluster --name operator-dev
