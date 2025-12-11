---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
lastStep: 8
status: 'complete'
completedAt: '2025-12-09T12:21:09+02:00'
inputDocuments:
  - docs/prd.md
  - docs/index.md
  - docs/project-overview.md
  - docs/developer/architecture/architecture.md
workflowType: 'architecture'
lastStep: 0
project_name: 'flightctl'
user_name: 'Ori'
date: '2025-12-09T12:21:09+02:00'
---

# Architecture Decision Document

_This document builds collaboratively through step-by-step discovery. Sections are appended as we work through each architectural decision together._

## Project Context Analysis

### Requirements Overview

**Functional Requirements:**

The project includes 28 functional requirements organized into 5 capability areas:

1. **Image Pruning Management (9 FRs)** - Core pruning operations including:
   - Automatic identification of unused container images and OCI artifacts
   - Removal of unused images from device storage
   - Querying Podman for all images/artifacts
   - Determining image references from current and previous device specs

2. **Rollback Safety (6 FRs)** - Critical safety requirements:
   - Preserving current and previous application images per application
   - Preserving current and rollback OS images (managed by bootc)
   - Verifying required images exist before pruning
   - Validating rollback capability after pruning operations

3. **Alert Response (4 FRs)** - Emergency handling:
   - Detecting critical disk space alerts from resource manager
   - Triggering emergency pruning operations
   - Responding without network connectivity
   - Clearing alerts after successful pruning

4. **Configuration Management (4 FRs)** - Control and flexibility:
   - Enabling/disabling pruning via agent configuration
   - Device-level configuration application
   - Default settings (enabled by default)

5. **Observability (5 FRs)** - Visibility and troubleshooting:
   - Logging pruning operations (images removed, space reclaimed)
   - Logging operation timing and errors
   - Support staff verification through logs

**Non-Functional Requirements:**

**Performance (4 NFRs):**
- Pruning operations must not block spec reconciliation
- Emergency pruning must respond within minutes of alert detection
- Operations must not significantly impact device performance
- Asynchronous execution to avoid blocking main reconciliation loop

**Reliability (6 NFRs):**
- Pruning failures must not block reconciliation
- Never remove images required for rollback
- Idempotent operations (safe to retry)
- Rollback capability validation after pruning
- Graceful handling of edge cases (concurrent updates, partial failures)
- System integrity preservation even if Podman operations fail

**Integration (4 NFRs):**
- Seamless integration with existing agent lifecycle hooks (AfterUpdate)
- Correct operation with Podman's image and artifact management APIs
- Respect bootc's OS image management (no manual OS pruning)
- Proper coordination with resource manager's critical disk alert system

**Scale & Complexity:**

- **Primary domain:** Backend service enhancement (agent functionality)
- **Complexity level:** Medium
- **Project type:** Brownfield - extending existing FlightCtl agent system
- **Estimated architectural components:** 6-8 components
  - Pruning Manager/Service (core pruning logic)
  - Image Tracker (determining eligible images)
  - Podman Client Extensions (list/remove operations)
  - Integration with Applications Manager (image usage tracking)
  - Integration with Resource Manager (alert handling)
  - Configuration Handler (pruning settings)

### Technical Constraints & Dependencies

**Existing System Integration:**
- Must integrate with existing agent lifecycle (AfterUpdate hook after specManager.Upgrade())
- Must work with existing Podman client infrastructure
- Must respect existing spec management (desired.json, current.json, rollback.json)
- Must coordinate with existing resource manager for disk alerts
- Must not disrupt existing reconciliation flow

**External Dependencies:**
- Podman for container/OCI image and artifact management
- bootc for OS image management (automatic rollback handling)
- Resource manager for critical disk alert detection
- Agent configuration system for pruning settings

**Technical Constraints:**
- No persistent state needed - determine eligible images on each pruning run
- Must work in offline scenarios (no network connectivity)
- Must handle edge cases: concurrent updates, partial failures, network interruptions
- Pruning must be non-blocking and failure-tolerant

### Cross-Cutting Concerns Identified

1. **State Management:**
   - Tracking images referenced in current.json and rollback.json specs
   - Determining current vs previous image per application
   - No persistent state - stateless pruning logic

2. **Error Handling:**
   - Pruning failures must not block reconciliation
   - Log warnings but continue operation
   - Retry logic for transient failures
   - Validation of rollback capability after pruning

3. **Observability:**
   - Logging pruning operations (what was removed, space reclaimed)
   - Logging operation timing and errors
   - Support staff verification through logs

4. **Configuration Management:**
   - Agent-level enable/disable flag
   - Default settings (enabled by default)
   - Future: per-fleet policies (post-MVP)

5. **Safety & Validation:**
   - Pre-pruning verification of required images
   - Post-pruning validation of rollback capability
   - Never prune images referenced in current or rollback specs
   - Respect bootc's OS image management

## Starter Template Evaluation

### Primary Technology Domain

**Backend Service Enhancement (Agent Functionality)** - This is a brownfield project extending an existing FlightCtl agent system written in Go.

### Starter Template Applicability

**Not Applicable** - Starter templates are designed for greenfield projects starting from scratch. This project extends an existing Go codebase with established architecture patterns.

**Rationale:**
- Existing codebase: Go 1.24.0 with established project structure
- Extending agent functionality, not creating new project
- Must integrate with existing architecture patterns
- No new project initialization required

### Existing Technical Foundation

**Language & Runtime:**
- Go 1.24.0
- Standard Go project layout
- Module-based dependency management

**Architecture Patterns (Already Established):**
- Manager-based architecture with lifecycle hooks
- Interface-driven design for testability
- Separation of concerns (device, applications, spec, lifecycle managers)
- Client abstraction pattern (Podman client, OS client)

**Project Structure:**
- `internal/agent/device/` - Core device management
- `internal/agent/client/` - External system clients (Podman, bootc, etc.)
- `internal/agent/device/applications/` - Application lifecycle management
- `internal/agent/device/spec/` - Spec management (desired, current, rollback)
- Standard Go testing patterns (`*_test.go` files)

**Development Experience:**
- Go standard tooling (go build, go test, go mod)
- Existing test infrastructure
- Established error handling patterns
- Logging via `pkg/log` package

**Note:** Architecture decisions will focus on integrating pruning functionality with existing patterns rather than establishing new foundations.

## Core Architectural Decisions

### Decision Priority Analysis

**Critical Decisions (Block Implementation):**
1. Pruning Component Architecture - New Manager Component
2. Integration Point - AfterUpdate Hook
3. Image Eligibility Determination - Spec-Based Analysis
4. Error Handling Strategy - Fail-Safe with Logging
5. Configuration Management - Structured Config Object
6. Observability Approach - Structured Logging Only
7. Alert Integration - Event-Based via Agent

**Important Decisions (Shape Architecture):**
- Podman Client Extensions - Add list/remove methods to existing client
- Safety Validation - Pre-pruning verification and post-pruning validation
- Rollback Coordination - Respect bootc's OS management, preserve application images

**Deferred Decisions (Post-MVP):**
- Advanced metrics and observability dashboard
- Per-fleet pruning policies
- Customizable retention policies beyond current + previous
- Pruning analytics and reporting
- Performance optimization strategies

### Pruning Component Architecture

**Decision:** Create new Manager Component following existing manager pattern

**Rationale:**
- Maintains consistency with existing agent architecture (applications manager, OS manager, lifecycle manager)
- Clear separation of concerns - pruning is distinct from application lifecycle
- Follows established interface patterns for testability
- Aligns with existing code organization in `internal/agent/device/`

**Implementation:**
- Create `internal/agent/device/pruning/manager.go`
- Implement manager interface pattern consistent with other managers
- Provide methods for: `Prune(ctx context.Context) error`, `PruneOnAlert(ctx context.Context) error`

**Affects:**
- Component structure and organization
- Testing approach (mockable interface)
- Integration with agent lifecycle

### Integration Point

**Decision:** Integrate via AfterUpdate Hook after successful spec reconciliation

**Rationale:**
- Matches PRD requirement: "Pruning runs after successful spec reconciliation"
- Natural integration point - runs after `specManager.Upgrade()` completes
- Maintains existing agent lifecycle flow
- Ensures pruning only runs after successful updates

**Implementation:**
- Extend agent's `afterUpdate()` method to call pruning manager
- Pruning executes after all other AfterUpdate hooks complete
- Non-blocking execution (async goroutine or quick synchronous call)

**Affects:**
- Agent lifecycle coordination
- Timing of pruning operations
- Error handling in reconciliation flow

### Image Eligibility Determination

**Decision:** Spec-Based Analysis - Parse current.json and rollback.json to extract image references

**Rationale:**
- Stateless approach - no persistent state needed (matches PRD requirement)
- Directly matches requirements: track images in current and rollback specs
- Simpler to reason about and test
- No cache invalidation concerns
- Determines eligibility on each pruning run

**Implementation:**
- Parse `current.json` and `rollback.json` specs
- Extract image references from application specs
- For each application: identify current and previous image
- Query Podman for all images/artifacts
- Compare lists to identify unused images

**Affects:**
- Pruning logic complexity
- Performance (parsing on each run)
- State management approach

### Error Handling Strategy

**Decision:** Fail-Safe with Logging - Log errors but never block reconciliation

**Rationale:**
- Meets NFR5: "Pruning failures must not block spec reconciliation"
- Simple implementation for MVP
- Ensures system reliability - pruning failures don't impact core functionality
- Logging provides visibility for troubleshooting

**Implementation:**
- All pruning operations wrapped in error handling
- Errors logged with context (which images failed, why)
- Reconciliation continues regardless of pruning outcome
- Warnings logged for partial failures

**Affects:**
- Error handling code structure
- Logging requirements
- System reliability guarantees

### Configuration Management

**Decision:** Structured Config Object - Config struct with enable flag and future extensibility

**Rationale:**
- Simple for MVP (just enable/disable flag)
- Extensible for post-MVP features (retention policies, schedules)
- Follows existing agent config patterns
- Allows future expansion without breaking changes

**Implementation:**
- Add `PruningConfig` struct to agent config
- Fields: `Enabled bool` (default: true)
- Future fields: `RetentionPolicy`, `Schedule`, etc.
- Configurable via agent configuration file

**Affects:**
- Configuration structure
- Default behavior
- Future extensibility

### Observability Approach

**Decision:** Structured Logging Only - Use existing log package with structured fields

**Rationale:**
- Meets MVP requirements (FR24-28: logging capabilities)
- Uses existing infrastructure (`pkg/log` package)
- Simple implementation
- Can add metrics in post-MVP phase
- Logs provide sufficient visibility for troubleshooting

**Implementation:**
- Use existing `*log.PrefixLogger` from agent
- Log pruning operations with structured fields:
  - Images removed (list)
  - Space reclaimed (bytes)
  - Operation duration
  - Errors/warnings
- Support staff can verify through logs (FR28)

**Affects:**
- Logging implementation
- Observability capabilities
- Post-MVP enhancement path

### Alert Integration

**Decision:** Event-Based via Agent - Agent receives alert and triggers pruning

**Rationale:**
- Maintains separation of concerns - resource manager handles alerts, agent coordinates response
- Loose coupling between components
- Agent can coordinate multiple responses to alerts
- Matches existing agent coordination patterns

**Implementation:**
- Resource manager detects critical disk alert
- Agent receives alert notification
- Agent triggers pruning manager's `PruneOnAlert()` method
- Pruning executes emergency cleanup
- Agent clears alert after successful pruning

**Affects:**
- Component coordination
- Alert handling flow
- Emergency response mechanism

### Decision Impact Analysis

**Implementation Sequence:**
1. Extend Podman client with list/remove methods (foundation)
2. Create pruning manager component (core logic)
3. Implement spec-based image eligibility determination (core logic)
4. Integrate with AfterUpdate hook (lifecycle integration)
5. Add configuration support (control)
6. Implement alert integration (emergency response)
7. Add structured logging (observability)

**Cross-Component Dependencies:**
- **Podman Client** → Pruning Manager (depends on list/remove methods)
- **Applications Manager** → Pruning Manager (needs to understand image usage)
- **Spec Manager** → Pruning Manager (needs to read current/rollback specs)
- **Resource Manager** → Agent → Pruning Manager (alert coordination)
- **Agent** → Pruning Manager (lifecycle integration)
- **Config** → Pruning Manager (configuration)

**Technology Versions:**
- Go: 1.24.0 (existing)
- Podman: Existing client integration (no version change needed)
- bootc: Existing integration (no version change needed)

## Implementation Patterns & Consistency Rules

### Pattern Categories Defined

**Critical Conflict Points Identified:**
6 areas where AI agents could make different choices:
1. Manager interface and implementation structure
2. Error handling and logging patterns
3. Testing patterns and mock usage
4. Package organization and file naming
5. Method naming conventions
6. Context usage patterns

### Naming Patterns

**Manager Interface & Implementation:**
- **Interface:** `Manager` (public, capitalized)
- **Implementation struct:** `manager` (private, lowercase)
- **Constructor:** `NewManager(...) Manager`
- **Package:** lowercase, single word (e.g., `pruning`)

**Method Naming:**
- **Public methods:** PascalCase (e.g., `Prune`, `PruneOnAlert`)
- **Private helpers:** camelCase (e.g., `determineEligibleImages`, `extractImageReferences`)
- **Context parameter:** Always first: `func (m *manager) Method(ctx context.Context, ...) error`

**File Naming:**
- Manager implementation: `manager.go`
- Tests: `manager_test.go`
- Mocks: `mock_manager.go` (if needed, following existing mock patterns)

**Variable Naming:**
- Follow Go conventions: camelCase for variables, PascalCase for exported
- Use descriptive names: `eligibleImages` not `images`, `currentSpec` not `spec`
- Constants: PascalCase for exported, camelCase for internal

### Structure Patterns

**Package Organization:**
- **Location:** `internal/agent/device/pruning/`
- **Files:** `manager.go` (interface + implementation), `manager_test.go`
- **Interface and implementation:** Same file (`manager.go`)

**Dependencies:**
- Inject via constructor (Podman client, spec manager, logger, readWriter)
- No global state or singletons
- Follow existing dependency injection patterns from other managers

**Manager Structure:**
```go
type Manager interface {
    Prune(ctx context.Context) error
    PruneOnAlert(ctx context.Context) error
}

type manager struct {
    podmanClient *client.Podman
    specManager  spec.Manager
    readWriter   fileio.ReadWriter
    log          *log.PrefixLogger
    config       PruningConfig
}
```

### Format Patterns

**Error Handling:**
- Use `fmt.Errorf("context: %w", err)` for error wrapping
- Use predefined errors from `errors` package when applicable
- Include context in error messages: what operation failed, which resource
- Never return errors that block reconciliation - log and continue

**Error Logging Pattern:**
```go
if err := m.podmanClient.RemoveImage(ctx, image); err != nil {
    m.log.Warnf("Failed to remove image %s: %v", image, err)
    // Continue with next image, don't return error
    continue
}
```

**Logging Levels:**
- **Debug:** Detailed operation info (image lists, eligibility checks)
- **Info:** Operations completed (images pruned, space reclaimed)
- **Warn:** Errors that don't block operation (individual image removal failures)
- **Error:** Critical failures (should not occur in normal operation)

**Structured Logging:**
```go
m.log.Infof("Pruned %d images, reclaimed %d bytes in %v", count, bytes, duration)
m.log.Debugf("Eligible images for pruning: %v", eligibleImages)
```

### Communication Patterns

**Context Usage:**
- Always accept `context.Context` as first parameter
- Use context for cancellation and timeouts
- Pass context to all external calls (Podman, file I/O, spec reading)
- Respect context cancellation in long-running operations

**Method Signatures:**
```go
func (m *manager) Prune(ctx context.Context) error
func (m *manager) determineEligibleImages(ctx context.Context, current, rollback *v1beta1.Device) ([]string, error)
```

**Event/Alert Integration:**
- Pruning triggered via method call from agent (not direct event subscription)
- Agent coordinates between resource manager alerts and pruning manager
- Pruning manager doesn't know about alert system directly

### Process Patterns

**Pruning Execution:**
- **Non-blocking:** Errors logged but don't return (fail-safe pattern)
- **Idempotent:** Safe to retry if interrupted
- **Stateless:** Determine eligibility on each run (no persistent state)
- **Validate before and after:** Verify required images exist before pruning, validate rollback capability after

**Pruning Flow Pattern:**
1. Check if pruning is enabled (config)
2. Read current and rollback specs
3. Extract image references from specs
4. Query Podman for all images
5. Determine eligible images (not in current or rollback)
6. Verify required images exist before pruning
7. Remove eligible images (log errors, continue on failure)
8. Validate rollback capability after pruning
9. Log results

**Configuration:**
- Read from agent config struct at startup
- Default values: pruning enabled by default
- No runtime configuration changes (read-only after initialization)
- Config struct: `PruningConfig` with `Enabled bool` field

**Integration Points:**
- Called from agent's `afterUpdate()` method after `specManager.Upgrade()`
- Can be triggered by alert events via agent coordination
- Never blocks reconciliation flow
- Executes asynchronously or quickly synchronously

### Testing Patterns

**Test Structure:**
- Table-driven tests for multiple scenarios
- Use `testify/require` for assertions
- Use `gomock` for mocking dependencies (Podman client, spec manager)
- Test file: `manager_test.go` in same package

**Mock Patterns:**
- Mock external dependencies: Podman client, spec manager, file I/O
- Use gomock expectations for behavior verification
- Test both success and failure paths
- Test edge cases: empty lists, concurrent operations, partial failures

**Test Example Pattern:**
```go
func TestManager_Prune(t *testing.T) {
    require := require.New(t)
    testCases := []struct {
        name        string
        setupMocks  func(*client.MockPodman, *spec.MockManager)
        current     *v1beta1.Device
        rollback    *v1beta1.Device
        wantRemoved []string
        wantError   bool
    }{
        // Test cases...
    }
    // Test execution...
}
```

### Enforcement Guidelines

**All AI Agents MUST:**

1. **Follow Manager Pattern:**
   - Public `Manager` interface
   - Private `manager` struct implementation
   - `NewManager()` constructor with dependency injection
   - Interface verification: `var _ Manager = (*manager)(nil)`

2. **Use Consistent Error Handling:**
   - Wrap errors with `fmt.Errorf("context: %w", err)`
   - Log errors but don't block reconciliation
   - Include context in error messages

3. **Follow Context Pattern:**
   - Context as first parameter in all methods
   - Pass context to all external calls
   - Respect context cancellation

4. **Use Existing Dependencies:**
   - Use `*log.PrefixLogger` from existing logging package
   - Use `fileio.ReadWriter` for file operations
   - Use existing Podman client (extend, don't replace)
   - Use existing spec manager for reading specs

5. **Follow Testing Patterns:**
   - Table-driven tests for multiple scenarios
   - Use gomock for mocking
   - Test both success and failure paths

**Pattern Enforcement:**
- Code reviews should verify pattern compliance
- Linter rules can enforce naming conventions
- Tests should verify interface compliance
- Documentation should reference these patterns

### Pattern Examples

**Good Examples:**

```go
// Manager interface and implementation
type Manager interface {
    Prune(ctx context.Context) error
}

type manager struct {
    podmanClient *client.Podman
    specManager  spec.Manager
    log          *log.PrefixLogger
}

func NewManager(podmanClient *client.Podman, specManager spec.Manager, log *log.PrefixLogger) Manager {
    return &manager{
        podmanClient: podmanClient,
        specManager:  specManager,
        log:          log,
    }
}

// Error handling with logging
func (m *manager) removeImage(ctx context.Context, image string) error {
    if err := m.podmanClient.RemoveImage(ctx, image); err != nil {
        m.log.Warnf("Failed to remove image %s: %v", image, err)
        return fmt.Errorf("removing image %s: %w", image, err)
    }
    return nil
}
```

**Anti-Patterns:**

```go
// ❌ DON'T: Public struct, no interface
type PruningManager struct { ... }

// ❌ DON'T: Block reconciliation on errors
func (m *manager) Prune(ctx context.Context) error {
    if err := m.removeImages(ctx); err != nil {
        return err  // This blocks reconciliation!
    }
}

// ❌ DON'T: Skip context
func (m *manager) Prune() error { ... }

// ❌ DON'T: Global state
var pruningManager *manager

// ❌ DON'T: Ignore errors silently
if err := m.removeImage(ctx, image); err != nil {
    // Silent failure - no logging!
}
```

## Project Structure & Boundaries

### Complete Project Directory Structure

**New Files and Directories for Pruning Feature:**

```
flightctl/
├── internal/
│   └── agent/
│       ├── agent.go                    # MODIFY: Add pruning manager initialization and AfterUpdate integration
│       ├── config/
│       │   └── config.go               # MODIFY: Add PruningConfig struct to agent config
│       ├── client/
│       │   └── podman.go               # MODIFY: Extend with ListImages, ListArtifacts, RemoveImage, RemoveArtifact methods
│       │   └── podman_test.go          # MODIFY: Add tests for new methods
│       └── device/
│           ├── device.go               # MODIFY: Call pruning manager in afterUpdate() method
│           ├── device_test.go          # MODIFY: Add integration tests for pruning
│           └── pruning/                # NEW: Pruning manager component
│               ├── manager.go          # NEW: Manager interface + implementation
│               ├── manager_test.go     # NEW: Unit tests for pruning logic
│               └── docs.go             # NEW: Package documentation (optional)
```

**Existing Structure (Unchanged):**
- All other directories and files remain unchanged
- Existing managers continue to function as before
- No changes to existing Podman client methods

### Architectural Boundaries

**Component Boundaries:**

**Pruning Manager Package:**
- **Location:** `internal/agent/device/pruning/`
- **Responsibilities:**
  - Image eligibility determination
  - Pruning execution logic
  - Rollback safety validation
  - Error handling and logging
- **Dependencies (Injected):**
  - Podman client (for image/artifact operations)
  - Spec manager (for reading current/rollback specs)
  - File I/O (via spec manager)
  - Logger (for observability)
  - Config (for pruning settings)
- **No Direct Dependencies On:**
  - Resource manager (coordinated via agent)
  - Applications manager (uses spec manager instead)
  - OS manager (bootc handles OS images)

**Podman Client Extensions:**
- **Location:** `internal/agent/client/podman.go`
- **New Methods (Additions Only):**
  - `ListImages(ctx context.Context) ([]string, error)`
  - `ListArtifacts(ctx context.Context) ([]string, error)`
  - `RemoveImage(ctx context.Context, image string) error`
  - `RemoveArtifact(ctx context.Context, artifact string) error`
- **Boundary:** Extends existing client, doesn't modify existing methods

**Agent Integration:**
- **Location:** `internal/agent/device/device.go`
- **Integration Point:** `afterUpdate()` method
- **Boundary:** Agent coordinates pruning, pruning manager doesn't know about agent lifecycle directly

**Configuration:**
- **Location:** `internal/agent/config/config.go`
- **Boundary:** Pruning config embedded in agent config struct
- **Access:** Read-only after initialization

### Requirements to Structure Mapping

**FR Category: Image Pruning Management (FR1-9)**
- **Core Implementation:** `internal/agent/device/pruning/manager.go`
  - `Prune(ctx context.Context) error` - Normal pruning after reconciliation (FR5)
  - `determineEligibleImages(ctx context.Context) ([]string, error)` - Eligibility logic (FR1, FR2)
  - `extractImageReferences(ctx context.Context, device *v1beta1.Device) ([]string, error)` - Spec parsing (FR6, FR7)
  - `removeEligibleImages(ctx context.Context, images []string) error` - Removal logic (FR3, FR4)
- **Podman Extensions:** `internal/agent/client/podman.go`
  - `ListImages()` - Query all images (FR8)
  - `ListArtifacts()` - Query all artifacts (FR9)
  - `RemoveImage()` - Remove image (FR3)
  - `RemoveArtifact()` - Remove artifact (FR4)

**FR Category: Rollback Safety (FR10-15)**
- **Implementation:** `internal/agent/device/pruning/manager.go`
  - `validateRequiredImages(ctx context.Context) error` - Pre-pruning verification (FR14)
  - `validateRollbackCapability(ctx context.Context) error` - Post-pruning validation (FR15)
  - Image preservation logic in `determineEligibleImages()` (FR10, FR11, FR12, FR13)
- **Integration:** Uses spec manager to read current/rollback specs

**FR Category: Alert Response (FR16-19)**
- **Implementation:** `internal/agent/device/pruning/manager.go`
  - `PruneOnAlert(ctx context.Context) error` - Emergency pruning (FR17, FR18)
- **Integration:** `internal/agent/device/device.go`
  - Agent receives alert, calls `PruneOnAlert()` (FR16)
  - Agent clears alert after successful pruning (FR19)

**FR Category: Configuration Management (FR20-23)**
- **Configuration:** `internal/agent/config/config.go`
  - `PruningConfig` struct with `Enabled bool` field (FR20, FR21, FR22, FR23)
- **Usage:** Pruning manager reads config at initialization

**FR Category: Observability (FR24-28)**
- **Implementation:** `internal/agent/device/pruning/manager.go`
  - Structured logging throughout pruning operations (FR24, FR25, FR26, FR27, FR28)
- **Uses:** Existing `pkg/log` package

### Integration Points

**Internal Communication:**

**Agent → Pruning Manager:**
- Method call: `pruningManager.Prune(ctx)` in `afterUpdate()`
- Method call: `pruningManager.PruneOnAlert(ctx)` on critical disk alert
- Communication: Synchronous method calls, non-blocking errors

**Pruning Manager → Podman Client:**
- Method calls: `ListImages()`, `ListArtifacts()`, `RemoveImage()`, `RemoveArtifact()`
- Communication: Direct method calls with context
- Error handling: Logged but don't propagate to caller

**Pruning Manager → Spec Manager:**
- Method calls: `Read(Current)`, `Read(Rollback)`
- Communication: Read-only access to spec files
- Error handling: Return errors if spec reading fails

**Resource Manager → Agent → Pruning Manager:**
- Event flow: Resource manager detects alert → Agent receives notification → Agent calls `PruneOnAlert()`
- Communication: Indirect via agent coordination
- Error handling: Pruning errors don't affect alert handling

**External Integrations:**

**Podman:**
- Integration: Via Podman client abstraction
- Operations: List, remove images and artifacts
- Error handling: Podman errors logged, don't block reconciliation

**bootc (OS Image Management):**
- Integration: Indirect - pruning manager doesn't touch OS images
- Boundary: bootc manages OS images, pruning only handles application images
- Communication: No direct communication, respect bootc's management

**Data Flow:**

1. **Normal Pruning Flow:**
   ```
   Agent.afterUpdate() 
   → PruningManager.Prune(ctx)
   → SpecManager.Read(Current, Rollback)
   → Extract image references from specs
   → PodmanClient.ListImages(), ListArtifacts()
   → Determine eligible images
   → PodmanClient.RemoveImage(), RemoveArtifact()
   → Log results
   ```

2. **Alert-Triggered Pruning Flow:**
   ```
   ResourceManager (detects critical disk alert)
   → Agent (receives alert notification)
   → PruningManager.PruneOnAlert(ctx)
   → [Same pruning logic as normal flow]
   → Agent (clears alert after success)
   ```

### File Organization Patterns

**Configuration Files:**
- Pruning config embedded in agent config struct
- No separate config file
- Default: pruning enabled
- Location: `internal/agent/config/config.go`

**Source Organization:**
- One manager per package pattern
- Pruning manager in dedicated `pruning/` package
- Interface and implementation in `manager.go`
- Tests co-located in `manager_test.go`
- Documentation in `docs.go` (optional)

**Test Organization:**
- Unit tests: `pruning/manager_test.go` (pruning logic, eligibility, error handling)
- Integration tests: `device/device_test.go` (agent lifecycle integration)
- Client tests: `client/podman_test.go` (new Podman methods)

**Asset Organization:**
- No static assets needed
- No additional data files
- Configuration via code (struct)

### Development Workflow Integration

**Development Server Structure:**
- No changes to development workflow
- Standard Go build and test commands
- Existing test infrastructure used

**Build Process Structure:**
- No changes to build process
- Pruning manager compiled into agent binary
- No additional build steps

**Deployment Structure:**
- No changes to deployment
- Pruning feature included in agent binary
- Configuration via existing agent config mechanism

## Architecture Validation Results

### Coherence Validation ✅

**Decision Compatibility:**
All architectural decisions work together cohesively:
- Manager pattern decision aligns with existing agent architecture
- AfterUpdate integration decision fits existing lifecycle hooks
- Spec-based analysis decision supports stateless requirement
- Fail-safe error handling decision meets NFR5 (non-blocking)
- Structured config decision allows future extensibility
- All technology choices use existing Go/Podman infrastructure

**Pattern Consistency:**
Implementation patterns fully support architectural decisions:
- Naming patterns align with existing codebase conventions (manager, NewManager, etc.)
- Structure patterns follow existing manager package organization
- Error handling patterns match existing agent fail-safe approaches
- Testing patterns use existing testify/gomock infrastructure
- Communication patterns respect existing dependency injection

**Structure Alignment:**
Project structure fully supports architectural decisions:
- New pruning package integrates seamlessly with existing device managers
- Podman client extensions maintain existing interface patterns
- Agent integration uses existing afterUpdate hook mechanism
- Configuration follows existing agent config patterns
- All boundaries respect existing component separation

### Requirements Coverage Validation ✅

**Functional Requirements Coverage:**
All 28 functional requirements are architecturally supported:

- **FR1-9 (Image Pruning Management):** ✅
  - Pruning manager implements eligibility determination and removal
  - Podman client extensions provide list/remove operations
  - Spec manager integration enables image reference extraction

- **FR10-15 (Rollback Safety):** ✅
  - Validation methods ensure required images exist
  - Spec-based analysis preserves current + previous images
  - bootc integration respected (no OS image pruning)

- **FR16-19 (Alert Response):** ✅
  - PruneOnAlert method handles emergency pruning
  - Agent coordination enables alert-triggered execution
  - Non-blocking design supports offline operation

- **FR20-23 (Configuration Management):** ✅
  - PruningConfig struct in agent config
  - Enable/disable flag with defaults
  - Device-level configuration support

- **FR24-28 (Observability):** ✅
  - Structured logging throughout pruning operations
  - Existing log package infrastructure
  - Support staff verification via logs

**Non-Functional Requirements Coverage:**
All 14 non-functional requirements are architecturally addressed:

- **Performance (NFR1-4):** ✅
  - Non-blocking design prevents reconciliation delays
  - Async execution capability
  - Acceptable timeframes maintained
  - Minimal device performance impact

- **Reliability (NFR5-10):** ✅
  - Fail-safe error handling (NFR5)
  - Rollback validation ensures safety (NFR6, NFR8)
  - Idempotent operations (NFR7)
  - Edge case handling (NFR9, NFR10)

- **Integration (NFR11-14):** ✅
  - AfterUpdate hook integration (NFR11)
  - Podman API compatibility (NFR12)
  - bootc respect (NFR13)
  - Resource manager coordination (NFR14)

### Implementation Readiness Validation ✅

**Decision Completeness:**
- ✅ All critical decisions documented with rationale
- ✅ Technology versions specified (Go 1.24.0, existing integrations)
- ✅ Integration patterns clearly defined
- ✅ Configuration approach fully specified
- ✅ Error handling strategy documented

**Structure Completeness:**
- ✅ Complete directory structure defined
- ✅ All new files and modifications specified
- ✅ Integration points clearly mapped
- ✅ Component boundaries well-defined
- ✅ Test organization specified

**Pattern Completeness:**
- ✅ Naming conventions comprehensive with examples
- ✅ Error handling patterns complete with code examples
- ✅ Testing patterns specified with mock patterns
- ✅ Communication patterns defined (context usage, method signatures)
- ✅ Process patterns documented (pruning flow, configuration)

### Gap Analysis Results

**Critical Gaps:** None identified
- All requirements have architectural support
- All integration points are defined
- All patterns are comprehensive

**Important Gaps (Non-Blocking):**
- Could add more detailed examples of pruning logic flow (helpful but not required)
- Could specify exact Podman command patterns for new methods (can be discovered during implementation)
- Could add more edge case handling examples (covered by testing patterns)

**Nice-to-Have Gaps (Post-MVP):**
- Metrics integration patterns (deferred to post-MVP per scope)
- Advanced configuration patterns (deferred to post-MVP per scope)
- Performance optimization strategies (can be added based on real-world usage)

### Validation Issues Addressed

No critical or important issues found. Architecture is coherent, complete, and ready for implementation.

### Architecture Completeness Checklist

**✅ Requirements Analysis**
- [x] Project context thoroughly analyzed
- [x] Scale and complexity assessed (Medium complexity, backend enhancement)
- [x] Technical constraints identified (brownfield extension, existing patterns)
- [x] Cross-cutting concerns mapped (state management, error handling, observability, configuration, safety)

**✅ Architectural Decisions**
- [x] Critical decisions documented with versions (7 critical decisions)
- [x] Technology stack fully specified (Go 1.24.0, existing integrations)
- [x] Integration patterns defined (AfterUpdate hook, event-based alerts)
- [x] Performance considerations addressed (non-blocking, async execution)

**✅ Implementation Patterns**
- [x] Naming conventions established (Manager interface, manager struct, NewManager)
- [x] Structure patterns defined (package organization, file naming)
- [x] Communication patterns specified (context usage, method signatures)
- [x] Process patterns documented (pruning flow, error handling, testing)

**✅ Project Structure**
- [x] Complete directory structure defined (pruning package, Podman extensions, agent integration)
- [x] Component boundaries established (pruning manager, Podman client, spec manager)
- [x] Integration points mapped (agent → pruning, resource manager → agent → pruning)
- [x] Requirements to structure mapping complete (all FRs mapped to specific files)

### Architecture Readiness Assessment

**Overall Status:** ✅ READY FOR IMPLEMENTATION

**Confidence Level:** High - Architecture is complete, coherent, and all requirements are supported

**Key Strengths:**
1. **Consistency:** All decisions align with existing codebase patterns
2. **Completeness:** All 28 FRs and 14 NFRs have architectural support
3. **Clarity:** Patterns and structure are well-defined with examples
4. **Integration:** Seamless integration with existing agent architecture
5. **Safety:** Rollback preservation and fail-safe error handling ensure reliability

**Areas for Future Enhancement:**
1. Metrics and observability dashboard (post-MVP)
2. Advanced configuration options (per-fleet policies, custom retention)
3. Performance optimization based on real-world usage data
4. Intelligent pruning strategies (prioritize larger images, predictive pruning)

### Implementation Handoff

**AI Agent Guidelines:**

1. **Follow Architectural Decisions Exactly:**
   - Use Manager interface pattern (public interface, private struct)
   - Integrate via AfterUpdate hook after specManager.Upgrade()
   - Use spec-based analysis for image eligibility
   - Implement fail-safe error handling (log but don't block)

2. **Use Implementation Patterns Consistently:**
   - Follow naming conventions (Manager, manager, NewManager)
   - Use context.Context as first parameter
   - Wrap errors with fmt.Errorf("context: %w", err)
   - Use existing log.PrefixLogger for all logging

3. **Respect Project Structure and Boundaries:**
   - Create pruning manager in `internal/agent/device/pruning/`
   - Extend Podman client in `internal/agent/client/podman.go`
   - Add config to `internal/agent/config/config.go`
   - Integrate in `internal/agent/device/device.go`

4. **Refer to This Document:**
   - All architectural questions should reference this document
   - Patterns section provides code examples
   - Structure section shows exact file locations
   - Validation confirms all requirements are supported

**First Implementation Priority:**

1. **Extend Podman Client** (`internal/agent/client/podman.go`):
   - Add `ListImages(ctx context.Context) ([]string, error)`
   - Add `ListArtifacts(ctx context.Context) ([]string, error)`
   - Add `RemoveImage(ctx context.Context, image string) error`
   - Add `RemoveArtifact(ctx context.Context, artifact string) error`
   - Add tests for new methods

2. **Create Pruning Manager** (`internal/agent/device/pruning/manager.go`):
   - Define Manager interface
   - Implement manager struct
   - Implement NewManager constructor
   - Implement Prune() method
   - Implement PruneOnAlert() method
   - Add helper methods (determineEligibleImages, extractImageReferences, etc.)

3. **Add Configuration** (`internal/agent/config/config.go`):
   - Add PruningConfig struct with Enabled bool field
   - Set default: Enabled = true

4. **Integrate with Agent** (`internal/agent/device/device.go`):
   - Initialize pruning manager in agent setup
   - Call pruningManager.Prune(ctx) in afterUpdate() method
   - Handle alert-triggered pruning coordination

## Architecture Completion Summary

### Workflow Completion

**Architecture Decision Workflow:** COMPLETED ✅
**Total Steps Completed:** 7
**Date Completed:** 2025-12-09T12:21:09+02:00
**Document Location:** docs/architecture.md

### Final Architecture Deliverables

**📋 Complete Architecture Document**

- All architectural decisions documented with specific versions
- Implementation patterns ensuring AI agent consistency
- Complete project structure with all files and directories
- Requirements to architecture mapping
- Validation confirming coherence and completeness

**🏗️ Implementation Ready Foundation**

- 7 architectural decisions made (component architecture, integration, eligibility, error handling, configuration, observability, alert integration)
- Comprehensive implementation patterns defined (naming, structure, format, communication, process, testing)
- 6 architectural components specified (pruning manager, Podman extensions, agent integration, spec manager integration, config, resource manager coordination)
- 42 requirements fully supported (28 FRs + 14 NFRs)

**📚 AI Agent Implementation Guide**

- Technology stack with verified versions (Go 1.24.0, existing integrations)
- Consistency rules that prevent implementation conflicts
- Project structure with clear boundaries
- Integration patterns and communication standards

### Implementation Handoff

**For AI Agents:**
This architecture document is your complete guide for implementing the automatic image pruning feature for FlightCtl. Follow all decisions, patterns, and structures exactly as documented.

**First Implementation Priority:**

1. **Extend Podman Client** (`internal/agent/client/podman.go`):
   - Add `ListImages(ctx context.Context) ([]string, error)`
   - Add `ListArtifacts(ctx context.Context) ([]string, error)`
   - Add `RemoveImage(ctx context.Context, image string) error`
   - Add `RemoveArtifact(ctx context.Context, artifact string) error`
   - Add tests for new methods

2. **Create Pruning Manager** (`internal/agent/device/pruning/manager.go`):
   - Define Manager interface
   - Implement manager struct
   - Implement NewManager constructor
   - Implement Prune() method
   - Implement PruneOnAlert() method
   - Add helper methods (determineEligibleImages, extractImageReferences, etc.)

3. **Add Configuration** (`internal/agent/config/config.go`):
   - Add PruningConfig struct with Enabled bool field
   - Set default: Enabled = true

4. **Integrate with Agent** (`internal/agent/device/device.go`):
   - Initialize pruning manager in agent setup
   - Call pruningManager.Prune(ctx) in afterUpdate() method
   - Handle alert-triggered pruning coordination

**Development Sequence:**

1. Extend Podman client with list/remove methods (foundation)
2. Create pruning manager component (core logic)
3. Implement spec-based image eligibility determination (core logic)
4. Integrate with AfterUpdate hook (lifecycle integration)
5. Add configuration support (control)
6. Implement alert integration (emergency response)
7. Add structured logging (observability)
8. Write comprehensive tests (unit + integration)

### Quality Assurance Checklist

**✅ Architecture Coherence**

- [x] All decisions work together without conflicts
- [x] Technology choices are compatible (Go, existing Podman, existing spec manager)
- [x] Patterns support the architectural decisions (manager pattern, error handling, testing)
- [x] Structure aligns with all choices (pruning package, Podman extensions, agent integration)

**✅ Requirements Coverage**

- [x] All functional requirements are supported (28 FRs mapped to specific implementations)
- [x] All non-functional requirements are addressed (14 NFRs covered by architectural decisions)
- [x] Cross-cutting concerns are handled (state management, error handling, observability, configuration, safety)
- [x] Integration points are defined (agent lifecycle, Podman client, spec manager, resource manager)

**✅ Implementation Readiness**

- [x] Decisions are specific and actionable (7 critical decisions with implementation details)
- [x] Patterns prevent agent conflicts (comprehensive naming, structure, format, communication patterns)
- [x] Structure is complete and unambiguous (all files and directories specified)
- [x] Examples are provided for clarity (good examples and anti-patterns documented)

### Project Success Factors

**🎯 Clear Decision Framework**
Every architectural decision was made collaboratively with clear rationale, ensuring the pruning feature integrates seamlessly with existing FlightCtl agent architecture.

**🔧 Consistency Guarantee**
Implementation patterns and rules ensure that AI agents will produce compatible, consistent code that follows existing codebase conventions and works together seamlessly.

**📋 Complete Coverage**
All 42 project requirements (28 FRs + 14 NFRs) are architecturally supported, with clear mapping from business needs to technical implementation.

**🏗️ Solid Foundation**
The architecture builds on existing FlightCtl patterns (manager-based architecture, dependency injection, lifecycle hooks) providing a consistent extension to the codebase.

---

**Architecture Status:** READY FOR IMPLEMENTATION ✅

**Next Phase:** Begin implementation using the architectural decisions and patterns documented herein. The architecture document serves as the single source of truth for all technical decisions.

**Document Maintenance:** Update this architecture when major technical decisions are made during implementation or when extending the feature beyond MVP scope.
```

