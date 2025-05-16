# run and cleanup the POC

SHELL := /bin/bash
KIND_CLUSTER := kubelogin-poc

.PHONY: all check-tools run run-pkce clean

all: check-tools
	@echo "targets: run run-pkce clean"

check-tools:
	@type -a kubectl-oidc_login &>/dev/null || echo "error: Install kubelogin: https://github.com/int128/kubelogin/releases"
	@type -a kind &>/dev/null || echo "error: Install kind: https://kind.sigs.k8s.io/docs/user/quick-start/"

run: check-tools
	./run.sh password

run-pkce: check-tools
	./run.sh device-code

clean:
	kind delete cluster --name $(KIND_CLUSTER)
	rm -rf ~/.kube/cache/oidc-login
