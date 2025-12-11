# API Contracts

## Overview

Flight Control provides multiple API interfaces for device and fleet management:

1. **REST API** - HTTP/JSON API for user-facing operations
2. **gRPC API** - High-performance protocol for agent communication
3. **Kubernetes-style API** - Declarative resource management (v1beta1)

## API Architecture

### REST API

**Location:** `internal/api_server/`, `internal/api/`

**Framework:** Chi Router (go-chi/chi/v5)

**Key Components:**
- API Server: `internal/api_server/server.go`
- Agent Server: `internal/api_server/agentserver/server.go`
- Middleware: `internal/api_server/middleware/middleware.go`
- Transport Layer: `internal/transport/` (handles auth, enrollment, console)

**Endpoints Pattern:**
- Authentication: `internal/transport/auth_*.go`
- Enrollment: `internal/transport/enrollmentrequest.go`
- Console: `internal/transport/console.go`
- Permission checks: `internal/transport/checkpermission.go`

### gRPC API

**Location:** `api/grpc/v1/`

**Services:**
- Enrollment Service: `enrollment_grpc.pb.go`, `enrollment.pb.go`
- Router Service: `router_grpc.pb.go`, `router.pb.go`

**Protocol:** gRPC with Protocol Buffers

### Kubernetes-style API (v1beta1)

**Location:** `api/v1beta1/`

**Resource Types:**
- Device
- Fleet
- EnrollmentRequest
- Repository
- ResourceSync
- CertificateSigningRequest
- AuthProvider
- PAM Issuer

**API Server:** `internal/api/server/server.go`

**Agent API Server:** `internal/api/server/agent/server.go`

## Authentication & Authorization

**Auth Providers:**
- OIDC/OAuth2: `internal/auth/authn/oauth2_auth.go`
- OpenShift: `internal/auth/authn/openshift_auth.go`
- PAM: `internal/auth/oidc/pam/`

**Transport Layer:**
- Auth Config: `internal/transport/auth_config.go`
- Auth Token: `internal/transport/auth_token.go`
- Auth UserInfo: `internal/transport/auth_userinfo.go`

## API Documentation

**Generated Docs:** `internal/api/server/docs.go`, `api/v1beta1/docs.go`

**OpenAPI/Swagger:** Uses `getkin/kin-openapi` for API documentation

## Notes

This is a quick scan summary. For detailed API documentation, refer to:
- `docs/user/references/api-resources.md`
- `api/v1beta1/README.md`
- Generated API documentation in code

