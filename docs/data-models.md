# Data Models

## Overview

Flight Control uses PostgreSQL as the primary database with GORM as the ORM layer.

## Database Architecture

**ORM:** GORM (gorm.io/gorm)  
**Driver:** PostgreSQL (gorm.io/driver/postgres)  
**Location:** `internal/store/`

## Core Data Models

### Device Management

**Device** (`internal/store/model/device.go`)
- Core device entity
- Tracks device state, configuration, and status

**Fleet** (`internal/store/model/fleet.go`)
- Fleet/group of devices
- Manages fleet-level policies and templates

**EnrollmentRequest** (`internal/store/model/enrollmentrequest.go`)
- Device enrollment requests
- Handles device registration workflow

### Resource Management

**Repository** (`internal/store/model/repository.go`)
- Container/image repository definitions
- Manages repository configurations

**ResourceSync** (`internal/store/model/resourcesync.go`)
- Resource synchronization state
- Tracks sync operations

**TemplateVersion** (`internal/store/model/templateversion.go`)
- Template versioning
- Manages template revisions

### Authentication & Authorization

**AuthProvider** (`internal/store/model/authprovider.go`)
- Authentication provider configurations
- OIDC, OAuth2, OpenShift, PAM providers

**Organization** (`internal/store/model/organization.go`)
- Multi-tenancy support
- Organization-level isolation

### Security

**CertificateSigningRequest** (`internal/store/model/certificatesigningrequest.go`)
- Certificate management
- CSR workflow

**Checkpoint** (`internal/store/model/checkpoint.go`)
- State checkpoints
- Recovery and consistency

### Events & Observability

**Event** (`internal/store/model/event.go`)
- Event logging
- System events and audit trail

**Resource** (`internal/store/model/resource.go`)
- Base resource model
- Common resource fields

## Store Layer

**Store Interface:** `internal/store/store.go`

**GORM Implementation:** `internal/store/gorm.go`

**Generic Operations:** `internal/store/generic.go`

## Selectors

**Location:** `internal/store/selector/`

**Types:**
- Label Selectors: `selector/label.go`
- Field Selectors: `selector/field.go`
- Annotation Selectors: `selector/annotation.go`
- Resolvers: `selector/resolvers.go`

## JSON Fields

**JSONField Model:** `internal/store/model/jsonfield.go`
- Flexible JSON field storage
- Schema-less data support

## Notes

This is a quick scan summary. For detailed schema information:
- Review `internal/store/model/*.go` files
- Check database migration files (if any)
- Refer to GORM models for field definitions

