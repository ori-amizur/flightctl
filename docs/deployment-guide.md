# Deployment Guide

## Overview

Flight Control supports multiple deployment methods:

1. **Kubernetes/OpenShift** - Helm charts
2. **Podman** - Container-based deployment
3. **Linux** - Systemd service deployment

## Kubernetes Deployment

### Helm Charts

**Location:** `deploy/helm/flightctl/`

**Key Components:**
- API Server deployment
- Worker service
- Database (PostgreSQL)
- Key-Value store (Redis)
- Telemetry Gateway
- Alert Manager Proxy
- UI deployment (if applicable)

**Configuration Files:**
- `values.yaml` - Default values
- `values.dev.yaml` - Development values
- `values.nodeport.yaml` - NodePort configuration

**Templates:**
- API: `templates/api/`
- Worker: `templates/worker/`
- Database: `templates/db/`
- Telemetry: `templates/telemetry-gateway/`
- Alert Manager: `templates/alertmanager/`
- UI: `templates/ui/`
- CLI Artifacts: `templates/cli-artifacts/`

### Deployment Methods

**Standard Deployment:**
```bash
helm install flightctl ./deploy/helm/flightctl
```

**Development:**
```bash
helm install flightctl ./deploy/helm/flightctl -f deploy/helm/flightctl/values.dev.yaml
```

## Podman Deployment

**Location:** `deploy/podman/`

**Components:**
- Service configuration files
- Container definitions
- Systemd integration

## Container Images

**Containerfiles:**
- `Containerfile.api` - API server
- `Containerfile.worker` - Worker service
- `Containerfile.telemetry-gateway` - Telemetry gateway
- `Containerfile.alert-exporter` - Alert exporter
- `Containerfile.alertmanager-proxy` - Alert Manager proxy
- `Containerfile.pam-issuer` - PAM issuer
- `Containerfile.periodic` - Periodic tasks
- `Containerfile.userinfo-proxy` - UserInfo proxy
- `Containerfile.db-setup` - Database setup
- `Containerfile.cli-artifacts` - CLI artifacts

## Database

**Database:** PostgreSQL

**Migration:** `cmd/flightctl-db-migrate/`

**Configuration:**
- External database support
- Internal database deployment
- Migration jobs

## Observability Stack

**Components:**
- Prometheus for metrics
- OpenTelemetry for tracing
- Telemetry Gateway for data forwarding
- Alert Manager integration

**Configuration:** `observability/service-config.yaml`

## Security

**Authentication:**
- OIDC/OAuth2 providers
- OpenShift integration
- PAM issuer support
- Kubernetes RBAC

**Network Policies:**
- Network isolation
- Service mesh support (if applicable)

## CI/CD

**GitHub Actions:** `.github/workflows/` (if present)

**Build System:** Makefile-based builds

## Documentation

For detailed deployment instructions, refer to:
- `docs/user/installing/` - Installation guides
- `docs/user/installing/installing-service-on-kubernetes.md`
- `docs/user/installing/installing-service-on-linux.md`
- `docs/user/installing/installing-service-on-openshift-disconnected.md`

## Notes

This is a quick scan summary. For detailed deployment steps:
- Review Helm chart values
- Check Podman deployment scripts
- Refer to user documentation in `docs/user/installing/`

