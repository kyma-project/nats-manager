PROJECT_ROOT ?= ../..
KYMA_CLI ?= "${PROJECT_ROOT}/${KYMA}"
CLUSTER_NAME ?= kyma
REGISTRY_PORT ?= 5001

ifndef MODULE_VERSION
	include ${PROJECT_ROOT}/.version
endif

ifndef MODULE_SHA
    MODULE_SHA = ${shell git rev-parse --short HEAD}
endif

.PHONY: module-build
module-build: ## Build the Module, push it to a registry and print it based on the commit hash.
	@make -C ${PROJECT_ROOT} module-build \
		IMG=${IMG} \
		MODULE_REGISTRY=${MODULE_REGISTRY} \
		MODULE_VERSION=${MODULE_VERSION}-${MODULE_SHA}
	cp -f ${PROJECT_ROOT}/module-template.yaml ${ARTIFACTS}/module-template.yaml
	@echo "Added template to job artifacts: ${ARTIFACTS}/module-template.yaml"
	@echo "\n~~~~~~~~~~~~BEGINNING OF MODULE TEMPLATE~~~~~~~~~~~~~~"
	@cat ${ARTIFACTS}/module-template.yaml
	@echo "\n~~~~~~~~~~~~~~~END OF MODULE TEMPLATE~~~~~~~~~~~~~~~~"

.PHONY: create-kyma-system-ns
create-kyma-system-ns: ## Create kyma-system namespace.
	kubectl create ns kyma-system

.PHONY: install-nats-manager
install-nats-manager: ## Install nats-manager only.
	kubectl apply -f https://github.com/kyma-project/nats-manager/releases/latest/download/nats-manager.yaml

.PHONY: create-k3d
create-k3d: ## Create k3d with kyma CRDs.
	"${KYMA_CLI}" provision k3d -p 8081:80@loadbalancer -p 8443:443@loadbalancer --registry-port ${REGISTRY_PORT} --name ${CLUSTER_NAME} --ci

.PHONY: install-k3d-tools
install-k3d-tools: ## Create k3d with kyma CRDs.
	curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | TAG=${K3D_VERSION} bash
