# Flight Control - Project Documentation Index

## Project Overview

- **Type:** Monolith (Backend Service)
- **Primary Language:** Go 1.24.0
- **Architecture:** Service-oriented backend with microservices architecture
- **Project:** flightctl

## Quick Reference

- **Tech Stack:** Go backend with gRPC/REST APIs, PostgreSQL, Redis, Kubernetes integration
- **Entry Points:** Multiple services in `cmd/` (API, Agent, Worker, etc.)
- **Architecture Pattern:** Service-oriented with clear API → Service → Store layers

## Generated Documentation

### Core Documentation

- [Project Overview](./project-overview.md) - High-level project summary
- [Source Tree Analysis](./source-tree-analysis.md) - Complete directory structure with annotations
- [Development Guide](./development-guide.md) - Setup, build, test, and development workflow
- [API Contracts](./api-contracts.md) - API architecture and endpoints _(Quick scan - pattern-based)_
- [Data Models](./data-models.md) - Database schema and models _(Quick scan - pattern-based)_
- [Deployment Guide](./deployment-guide.md) - Deployment methods and configurations _(Quick scan - pattern-based)_

## Existing Documentation

### User Documentation

- [User Guide Index](./user/README.md) - User documentation overview
- [Introduction](./user/introduction.md) - Project vision and concepts
- [Installation Guides](./user/installing/) - Installation instructions
  - [Kubernetes Installation](./user/installing/installing-service-on-kubernetes.md)
  - [Linux Installation](./user/installing/installing-service-on-linux.md)
  - [OpenShift Installation](./user/installing/installing-service-on-openshift-disconnected.md)
  - [Agent Installation](./user/installing/installing-agent.md)
  - [CLI Installation](./user/installing/installing-cli.md)
- [Configuration Guides](./user/installing/configuring-auth/) - Authentication and configuration
- [Usage Guides](./user/using/) - Device and fleet management
- [References](./user/references/) - API resources, CLI commands, metrics

### Developer Documentation

- [Developer Guide Index](./developer/README.md) - Developer documentation overview
- [Architecture](./developer/architecture/architecture.md) - System architecture
- [Architecture Details](./developer/architecture/) - Detailed architecture components
  - [Alerts](./developer/architecture/alerts.md)
  - [Field Selectors](./developer/architecture/field-selectors.md)
  - [Key-Value Store](./developer/architecture/key-value-store.md)
  - [Rollout Device Selection](./developer/architecture/rollout-device-selection.md)
  - [Rollout Disruption Budget](./developer/architecture/rollout-disruption-budget.md)
  - [Service Observability](./developer/architecture/service-observability.md)
- [Enhancement Proposals](./developer/enhancements/) - Feature enhancement proposals
- [PAM Issuer](./developer/pam-issuer-architecture.md) - PAM issuer architecture

## Getting Started

### For Users

1. Read the [Introduction](./user/introduction.md) to understand Flight Control
2. Choose your [Installation Method](./user/installing/)
3. Configure [Authentication](./user/installing/configuring-auth/)
4. Start [Managing Devices](./user/using/managing-devices.md)

### For Developers

1. Review [Project Overview](./project-overview.md)
2. Read [Development Guide](./development-guide.md) for setup
3. Explore [Source Tree](./source-tree-analysis.md) to understand structure
4. Check [Architecture Documentation](./developer/architecture/architecture.md)
5. Review [API Contracts](./api-contracts.md) and [Data Models](./data-models.md)

## Documentation Status

**Generated:** 2025-12-08 (Quick Scan - Pattern-based analysis)

**Note:** This documentation was generated using a quick scan (pattern-based, no source file reading). For detailed API documentation, data model schemas, and complete deployment instructions, refer to the existing documentation in `docs/user/` and `docs/developer/`, or run a deep/exhaustive scan for comprehensive analysis.

## Next Steps

When planning new features for this brownfield project:

1. Reference this index.md as the primary entry point
2. Use the generated documentation for quick reference
3. Consult existing architecture docs for detailed system understanding
4. Run PRD workflow with this index as input for comprehensive planning

---

**Workflow Status:** Document Project workflow completed (Quick Scan mode)

