# Source Tree Analysis

## Project Structure

```
flightctl/
├── api/                          # API definitions
│   ├── grpc/v1/                  # gRPC API definitions
│   │   ├── enrollment.proto      # Enrollment service
│   │   └── router.proto          # Router service
│   └── v1beta1/                  # Kubernetes-style API
│       ├── types.go              # Resource types
│       ├── spec.gen.go           # Generated specs
│       └── agent/                # Agent API types
│
├── cmd/                          # Application entry points
│   ├── flightctl/                # CLI tool
│   ├── flightctl-api/           # API server
│   ├── flightctl-agent/         # Device agent
│   ├── flightctl-worker/        # Background worker
│   ├── flightctl-telemetry-gateway/  # Telemetry gateway
│   ├── flightctl-alert-exporter/     # Alert exporter
│   ├── flightctl-alertmanager-proxy/ # Alert Manager proxy
│   ├── flightctl-pam-issuer/     # PAM issuer
│   ├── flightctl-periodic/      # Periodic tasks
│   ├── flightctl-userinfo-proxy/    # UserInfo proxy
│   ├── flightctl-db-migrate/    # Database migrations
│   ├── flightctl-restore/       # Restore utility
│   └── flightctl-standalone/    # Standalone mode
│
├── internal/                     # Internal application code
│   ├── agent/                   # Device agent implementation
│   │   └── device/              # Device management
│   │       ├── console/         # Remote console
│   │       ├── lifecycle/       # Device lifecycle
│   │       ├── policy/          # Policy management
│   │       ├── spec/            # Device spec handling
│   │       ├── status/          # Status reporting
│   │       └── systemd/          # Systemd integration
│   │
│   ├── api/                      # API layer
│   │   ├── server/               # API server implementation
│   │   │   └── agent/           # Agent API server
│   │   ├── client/               # API client
│   │   └── common/               # Common API utilities
│   │
│   ├── api_server/              # REST API server
│   │   ├── agentserver/         # Agent server
│   │   └── middleware/          # HTTP middleware
│   │
│   ├── auth/                     # Authentication & authorization
│   │   ├── authn/                # Authentication
│   │   │   ├── oauth2_auth.go   # OAuth2
│   │   │   ├── oidc_auth.go     # OIDC
│   │   │   └── openshift_auth.go # OpenShift
│   │   ├── authz/                # Authorization
│   │   └── oidc/pam/             # PAM issuer
│   │
│   ├── cli/                      # CLI implementation
│   ├── config/                   # Configuration management
│   ├── console/                  # Console functionality
│   ├── crypto/                   # Cryptography utilities
│   ├── identity/                 # Identity management
│   ├── instrumentation/          # Observability instrumentation
│   ├── kvstore/                  # Key-value store
│   ├── org/                      # Organization management
│   ├── periodic_checker/         # Periodic checks
│   ├── quadlet/                  # Quadlet integration
│   ├── rendered/                  # Template rendering
│   ├── rollout/                  # Rollout management
│   ├── service/                  # Business logic layer
│   ├── store/                     # Data persistence layer
│   │   ├── model/                # Data models
│   │   └── selector/             # Resource selectors
│   ├── tasks/                    # Background tasks
│   ├── telemetry_gateway/        # Telemetry gateway
│   ├── tpm/                      # TPM (Trusted Platform Module)
│   └── transport/                # Transport layer
│
├── pkg/                          # Shared packages
│   ├── aap/                      # Ansible Automation Platform
│   ├── crypto/                   # Cryptographic utilities
│   ├── executer/                 # Execution utilities
│   ├── ignition/                 # Ignition integration
│   ├── k8s/                      # Kubernetes utilities
│   ├── k8sclient/                # Kubernetes client
│   ├── log/                      # Logging utilities
│   ├── poll/                     # Polling utilities
│   ├── queryparser/              # Query parsing
│   ├── queues/                   # Queue management
│   ├── reqid/                    # Request ID utilities
│   ├── ring_buffer/              # Ring buffer
│   ├── template/                 # Template processing
│   ├── thread/                   # Thread utilities
│   └── version/                  # Version management
│
├── test/                         # Test suites
│   ├── e2e/                      # End-to-end tests
│   ├── integration/              # Integration tests
│   ├── harness/                  # Test harness
│   └── util/                     # Test utilities
│
├── deploy/                       # Deployment configurations
│   ├── helm/                     # Helm charts
│   ├── podman/                   # Podman deployment
│   └── scripts/                  # Deployment scripts
│
├── docs/                         # Documentation
│   ├── developer/                # Developer docs
│   │   └── architecture/        # Architecture docs
│   └── user/                     # User docs
│
├── hack/                         # Build and development scripts
├── packaging/                    # Packaging files (RPM, DEB, etc.)
└── tools/                        # Development tools
```

## Critical Directories

### Entry Points
- **CLI**: `cmd/flightctl/main.go`
- **API Server**: `cmd/flightctl-api/main.go`
- **Agent**: `cmd/flightctl-agent/main.go`
- **Worker**: `cmd/flightctl-worker/main.go`

### Core Business Logic
- **Services**: `internal/service/` - Business logic layer
- **Store**: `internal/store/` - Data persistence
- **API**: `internal/api_server/` - REST API handlers

### Device Management
- **Agent**: `internal/agent/` - Device agent implementation
- **Transport**: `internal/transport/` - Communication layer

### Configuration & Deployment
- **Helm**: `deploy/helm/flightctl/` - Kubernetes deployment
- **Podman**: `deploy/podman/` - Container deployment
- **Config**: `internal/config/` - Configuration management

## Integration Points

- **API Layer**: REST (`internal/api_server/`) + gRPC (`api/grpc/`)
- **Data Layer**: GORM (`internal/store/gorm.go`) + PostgreSQL
- **Auth Layer**: Multiple providers (`internal/auth/`)
- **Observability**: OpenTelemetry (`internal/instrumentation/`)
- **Kubernetes**: Client integration (`pkg/k8sclient/`)

## Notes

This structure supports a microservices architecture with:
- Multiple service entry points
- Shared packages for common functionality
- Clear separation of concerns (API, service, store layers)
- Comprehensive testing infrastructure
- Multiple deployment options

