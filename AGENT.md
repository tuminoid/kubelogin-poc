# Copilot Instructions for kubelogin-poc

## Project Overview

This is a POC (Proof of Concept) for implementing Kubernetes OIDC authentication using the `int128/kubelogin` plugin with Dex as the OIDC provider and OpenLDAP as the backend authentication source. The project demonstrates two authentication flows: password grant and device-code/PKCE flow.

## Architecture & Key Components

### Authentication Chain

- **kubectl** → **kubelogin** → **Dex (OIDC Provider)** → **OpenLDAP**
- Kubernetes apiserver is configured to trust Dex as OIDC provider
- Dex acts as identity broker between k8s and LDAP

### Key Files Structure

- `run.sh` - Main orchestration script that sets up entire environment
- `Makefile` - Convenience targets for common operations
- `gencert.sh` - Generates test certificates for TLS setup
- `kind.conf` - Kind cluster configuration with OIDC settings
- `k8s/dex-password.yaml` - Dex configuration for password grant flow
- `k8s/dex-device-code.yaml` - Dex configuration for device-code/PKCE flow
- `k8s/openldap.yaml` - OpenLDAP test deployment
- `ssl/` - Generated certificates directory

### Two Authentication Flows

#### Password Grant Flow

- Direct username/password exchange for tokens
- Requires client secret in kubeconfig
- Configured via `dex-password.yaml`
- Client: `kubelogin-test` with secret `kubelogin-test-secret`

#### Device-Code Flow (Recommended)

- PKCE-based authentication with device codes
- No client secret required (public client)
- Browser-based authentication on separate device
- Configured via `dex-device-code.yaml`

## Development Workflows

### Quick Start Commands

```bash
# Password flow setup
make run
# OR
./run.sh password

# Device-code flow setup
make run-pkce
# OR
./run.sh device-code

# Cleanup
make clean
```

### Manual Testing Flow

1. **Environment Setup**: `./run.sh <flow-type>` creates Kind cluster, deploys Dex/LDAP, configures kubectl
2. **Certificate Management**: `gencert.sh` generates CA and TLS certs that must be trusted by both apiserver and client
3. **Authentication**: `kubectl --user oidc get pods -A` triggers OIDC flow
4. **Test Users**:
   - `customuser/custompassword` → cluster-admin role
   - `foo/bar` → view role

### Key Configuration Points

#### Kind Cluster Setup

- Mounts `/etc/ssl/certs/dex-test` for Dex CA trust
- OIDC flags: `--oidc-issuer-url`, `--oidc-client-id`, `--oidc-username-claim=email`, `--oidc-groups-claim=groups`

#### Dex Configuration Differences

- **Password flow**: Requires `passwordConnector: ldap` and static client with secret
- **Device-code flow**: Requires `responseTypes: ["code", "token", "id_token"]` and public client with redirect URIs

#### Certificate Requirements

- Dex must have valid TLS cert for domain `dex.example.com`
- CA cert must be trusted by Kubernetes apiserver
- Client can use `--insecure-skip-tls-verify` for testing

## Project-Specific Patterns

### Configuration Management

- Environment-specific YAML files for different auth flows
- Shared base configuration with flow-specific overrides
- Certificate generation tied to specific domain (`dex.example.com`)

### Service Exposure Strategy

- Dex exposed via NodePort 32000 (simulates external OIDC provider)
- OpenLDAP exposed via NodePort 31389 (debugging access)
- Domain mapping via `/etc/hosts` or replace with `127.0.0.1`

### RBAC Integration

- ClusterRoleBindings created based on OIDC username/groups claims
- Username comes from `email` claim in OIDC token
- Groups come from `groups` claim for group-based authorization

## Common Issues & Debugging

### Certificate Trust Issues

- Symptom: OIDC authentication fails despite valid tokens
- Check: Ensure Dex CA is properly mounted and trusted by apiserver
- Debug: Verify certificate CN and SAN match `dex.example.com`

### LDAP Connection Problems

- Check: OpenLDAP pod status and service connectivity
- Debug: Use NodePort 31389 to test LDAP connectivity directly
- Verify: LDAP connector configuration in Dex matches OpenLDAP schema

### Token Cache Issues

- Clear: `rm -rf ~/.kube/cache/oidc-login` to force re-authentication
- Location: kubelogin stores tokens in user's kube cache directory

### Flow-Specific Troubleshooting

- **Password flow**: Ensure `passwordConnector` is set correctly
- **Device-code flow**: Verify redirect URIs and response types configuration

## Dependencies & Prerequisites

- `kubectl` with `oidc-login` plugin installed as `kubectl-oidc_login`
- `kind` for local Kubernetes cluster
- `openssl` for certificate generation
- Domain resolution for `dex.example.com` (via /etc/hosts or DNS)

## Testing Users

The OpenLDAP deployment includes pre-configured test users:

- **customuser** (admin access) / **custompassword**
- **foo** (view access) / **bar**

RBAC is configured to map these users to appropriate cluster roles based on their email claim values.
