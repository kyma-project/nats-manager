PROJECT_ROOT ?= ../..
KYMA_CLI ?= "${PROJECT_ROOT}/${KYMA}"
CLUSTER_NAME ?= kyma
REGISTRY_PORT ?= 5001
K3D_VERSION ?= 1.30.3

ifndef MODULE_VERSION
	include ${PROJECT_ROOT}/.version
endif

ifndef MODULE_SHA
    MODULE_SHA = ${shell git rev-parse --short HEAD}
endif

.PHONY: create-kyma-system-ns
create-kyma-system-ns: ## Create kyma-system namespace.
	kubectl create ns kyma-system

.PHONY: install-latest-nats-manager-release
install-latest-nats-manager-release: ## Install nats-manager only.
	kubectl apply -f https://github.com/kyma-project/nats-manager/releases/latest/download/nats-manager.yaml

.PHONY: create-k3d
create-k3d: ## Create k3d with kyma CRDs.
	"${KYMA_CLI}" provision k3d -p 8081:80@loadbalancer -p 8443:443@loadbalancer --registry-port ${REGISTRY_PORT} -k ${K3D_VERSION} --name ${CLUSTER_NAME} --ci

.PHONY: install-k3d-tools
install-k3d-tools: ## Create k3d with kyma CRDs.
	curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
