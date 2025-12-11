# Project Overview

## Flight Control

**Type:** Backend Service (Monolith)  
**Language:** Go 1.24.0  
**Architecture:** Service-oriented backend with microservices architecture

## Project Purpose

Flight Control is a service for declarative management of fleets of edge devices and their workloads. It provides:

- **Device Management**: Enroll, configure, and monitor edge devices
- **Fleet Management**: Manage groups of devices with common policies
- **Workload Management**: Deploy and manage container/VM workloads
- **Declarative APIs**: Kubernetes-like APIs for GitOps workflows
- **Agent-based Architecture**: Scalable, robust management under adverse networking conditions

## Technology Stack

| Category | Technology |
|----------|-----------|
| Language | Go 1.24.0 |
| API Framework | Chi Router (REST), gRPC |
| Database | PostgreSQL with GORM |
| Cache/Queue | Redis |
| Observability | OpenTelemetry, Prometheus |
| Container | Podman/Docker |
| Deployment | Helm, Kubernetes |
| Testing | Ginkgo/Gomega |

## Architecture

### Service Components

- **API Server**: REST and gRPC endpoints
- **Agent**: Device management agent
- **Worker**: Background job processing
- **Telemetry Gateway**: Observability data forwarding
- **Alert Manager Proxy**: Alert routing
- **PAM Issuer**: Authentication provider

### Key Features

- Multi-tenant support (organizations)
- Multiple authentication providers (OIDC, OAuth2, OpenShift, PAM)
- Kubernetes integration
- Remote console access
- Device attestation and security
- Rollout management with disruption budgets
- Event-driven architecture

## Repository Structure

- **Monolith**: Single cohesive codebase
- **Multiple Services**: Multiple entry points in `cmd/`
- **Clear Layers**: API → Service → Store architecture
- **Comprehensive Testing**: Unit, integration, and E2E tests

## Documentation

- **User Docs**: `docs/user/` - Installation, usage, references
- **Developer Docs**: `docs/developer/` - Architecture, enhancements
- **API Docs**: Generated from code and OpenAPI specs

## Deployment

Supports multiple deployment methods:
- Kubernetes/OpenShift (Helm charts)
- Podman containers
- Linux systemd services

## Getting Started

1. Review [User Documentation](user/README.md)
2. Check [Developer Documentation](developer/README.md)
3. See [Installation Guides](user/installing/)
4. Explore [API Resources](user/references/api-resources.md)

## Project Status

Currently in beta. See README.md for latest status.

