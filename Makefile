# run and cleanup the POC

SHELL := /bin/bash
KIND_CLUSTER := kubelogin-poc

# Tool versions
KUBELOGIN_VERSION := v1.34.1
KIND_VERSION := v0.30.0
INSTALL_DIR := /usr/local/bin

# OS and architecture detection
OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH := $(shell uname -m | sed 's/x86_64/amd64/g' | sed 's/aarch64/arm64/g')

.PHONY: all check-tools install-tools run run-pkce clean client

all: check-tools
	@echo "targets: run run-pkce clean install-tools client"

check-tools:
	@type -a kubectl-oidc_login &>/dev/null || echo "error: forgot to run 'make install-tools'?"
	@type -a kind &>/dev/null || echo "error: forgot to run 'make install-tools'?"

install-tools:
	@echo "Installing tools to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)

	echo "Installing kubelogin $(KUBELOGIN_VERSION)..."; \
	curl -sSL -o /tmp/kubelogin.zip "https://github.com/int128/kubelogin/releases/download/$(KUBELOGIN_VERSION)/kubelogin_$(OS)_$(ARCH).zip"; \
	unzip -q -o /tmp/kubelogin.zip -d /tmp; \
	sudo install -m 755 /tmp/kubelogin $(INSTALL_DIR)/kubectl-oidc_login; \
	rm -f /tmp/kubelogin.zip /tmp/kubelogin

	echo "Installing kind $(KIND_VERSION)..."; \
	curl -sSL -o /tmp/kind "https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-$(OS)-$(ARCH)"; \
	sudo install -m 755 /tmp/kind $(INSTALL_DIR)/kind; \
	rm -f /tmp/kind

run: check-tools
	./run.sh password

run-pkce: check-tools
	./run.sh device-code

clean:
	kind delete cluster --name $(KIND_CLUSTER)
	rm -rf ~/.kube/cache/oidc-login

logs:
	echo "LDAP logs:"
	kubectl -n openldap logs deployments/openldap | tail -20
	echo
	echo "Dex logs:"
	kubectl -n dex logs deployments/dex | tail -20
	echo
	echo "K8s apiserver logs:"
	kubectl logs -n kube-system kube-apiserver-kubelogin-poc-control-plane | tail -20

client:
	@echo "Building and running OAuth2 test client..."
	@echo "Clearing old token cache and getting fresh tokens..."
	rm -rf ~/.kube/cache/oidc-login
	kubectl --user oidc get pods -A
	@echo "Running test client with fresh tokens..."
	cd client && go build -o test-client test-client.go
	cd client && time ./test-client
