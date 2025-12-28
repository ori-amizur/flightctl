---
stepsCompleted: [1, 2, 3, 4]
lastStep: 4
status: 'complete'
completedAt: '2025-12-28T15:30:00+02:00'
inputDocuments:
  - docs/prd.md
  - docs/architecture.md
workflowType: 'epics-stories'
project_name: 'flightctl'
user_name: 'Ori'
date: '2025-12-28T15:30:00+02:00'
---

# flightctl - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for flightctl image and artifact pruning functionality. **Stories are organized so that each source file is modified by exactly one story**, ensuring clear ownership and avoiding merge conflicts.

## File-to-Story Mapping

| File | Story | Description |
|------|-------|-------------|
| `internal/agent/client/podman.go` + `podman_test.go` | 1.1 | Podman client methods |
| `internal/agent/device/pruning/manager.go` + `manager_test.go` | 1.2 | Complete pruning manager implementation |
| `internal/agent/config/config.go` + `config_pruning_dropin_test.go` | 2.1 | Configuration support |
| `internal/agent/device/device.go` + `device_test.go` | 1.3 | Agent lifecycle integration |

**✅ Each file is modified by exactly one story**

## Story Dependencies

### Dependency Graph

```
Story 1.1 (Podman Client)
    │
    ├─> Story 1.2 (Pruning Manager) ──┐
    │                                   │
Story 2.1 (Configuration) ─────────────┼─> Story 1.3 (Agent Integration)
    │                                   │
    └──────────────────────────────────┘
    
Story 3.1 (Verification) ──> Depends on Stories 1.1 and 1.2 (verification only, no code changes)
```

### Detailed Dependencies

**Story 1.1: Extend Podman Client**
- **Dependencies:** None (foundational)
- **Used by:** Story 1.2 (pruning manager uses Podman client methods)

**Story 2.1: Add Pruning Configuration Support**
- **Dependencies:** None (can be done in parallel with Story 1.1)
- **Used by:** Story 1.2 (pruning manager uses `config.Pruning` type and checks `config.Enabled`)

**Story 1.2: Implement Complete Pruning Manager**
- **Dependencies:**
  - **Story 1.1** (requires): Uses Podman client methods:
    - `ListImages()` - to query all images
    - `ListArtifacts()` - to query all artifacts
    - `RemoveImage()` - to remove eligible images
    - `RemoveArtifact()` - to remove eligible artifacts
    - `ImageExists()` - to categorize references and validate
    - `ArtifactExists()` - to categorize references and validate
  - **Story 2.1** (requires): Uses configuration:
    - `config.Pruning` type (aliased as `PruningConfig`)
    - `config.Enabled` field to check if pruning is enabled
- **Used by:** Story 1.3 (agent integration uses pruning manager interface)

**Story 1.3: Integrate Pruning with Agent Lifecycle**
- **Dependencies:**
  - **Story 1.2** (requires): Uses pruning manager:
    - `pruning.Manager` interface
    - `Prune(ctx)` method
    - `RecordReferences(ctx)` method
  - **Story 2.1** (indirect): Configuration must be loaded by agent, but Story 1.3 doesn't modify config files
- **Used by:** None (final integration story)

**Story 3.1: Verify Comprehensive Pruning Operation Logging**
- **Dependencies:**
  - **Story 1.1** (verification): Verifies logging in Podman client methods
  - **Story 1.2** (verification): Verifies logging throughout pruning manager
- **Used by:** None (verification story, no code changes)

### Implementation Order

**Phase 1: Foundation (can be done in parallel)**
1. Story 1.1: Extend Podman Client
2. Story 2.1: Add Pruning Configuration Support

**Phase 2: Core Implementation**
3. Story 1.2: Implement Complete Pruning Manager (requires 1.1 and 2.1)

**Phase 3: Integration**
4. Story 1.3: Integrate Pruning with Agent Lifecycle (requires 1.2)

**Phase 4: Verification**
5. Story 3.1: Verify Comprehensive Pruning Operation Logging (verifies 1.1 and 1.2)

### Dependency Notes

- **Story 1.1 and 2.1 are independent** and can be implemented in parallel
- **Story 1.2 requires both 1.1 and 2.1** to be completed first
- **Story 1.3 requires 1.2** to be completed first
- **Story 3.1 is verification-only** and can be done after 1.1 and 1.2 are complete

## Requirements Inventory

### Functional Requirements

FR1: The agent can automatically identify container images that were previously referenced but are no longer referenced in current specs
FR2: The agent can automatically identify OCI artifacts that were previously referenced but are no longer referenced in current specs
FR3: The agent can remove unused application images from device storage
FR4: The agent can remove unused OCI artifacts from device storage
FR5: The agent can execute pruning operations after successful spec reconciliation
FR6: The agent can determine which images and artifacts are referenced by the current device spec
FR7: The agent can determine which images and artifacts are referenced by the desired device spec
FR8: The agent can query Podman for all container images stored on the device
FR9: The agent can query Podman for all OCI artifacts stored on the device
FR10: The agent can preserve images and artifacts referenced in current or desired specs during pruning operations
FR11: The agent can preserve OS images that are still referenced in specs during pruning operations
FR12: The agent can prune OS images if they lose their references
FR13: The agent can record all image and artifact references to a persistent file before upgrades
FR14: The agent can accumulate references across multiple upgrades in the references file
FR15: The agent can remove successfully pruned items from the references file
FR16: The agent can maintain the references file even when pruning is disabled
FR17: The agent can extract nested OCI targets (images and artifacts) from image-based applications
FR18: Fleet operators can enable automatic image pruning via file dropin configuration
FR19: Fleet operators can disable automatic image pruning via file dropin configuration
FR20: The agent can apply pruning configuration settings at the device or fleet level
FR21: The agent can use default pruning settings when configuration is not explicitly set (default: enabled)
FR22: The agent can log pruning operations including which images and artifacts were removed
FR23: The agent can log pruning operation start and completion times
FR24: The agent can log warnings when pruning operations encounter errors
FR25: Support staff can verify pruning behavior through device logs

### NonFunctional Requirements

NFR1: Pruning operations must complete within acceptable timeframes that do not block or significantly delay spec reconciliation
NFR2: Pruning operations must not significantly impact device performance or active workloads during execution
NFR3: Pruning failures must not block spec reconciliation - failures should be logged and reconciliation should continue
NFR4: Pruning operations must be idempotent - safe to retry if interrupted or if partial failures occur
NFR5: Pruning must handle edge cases gracefully (concurrent updates, partial failures, network interruptions) without causing system instability
NFR6: Pruning must preserve system integrity even if Podman operations fail or return unexpected results
NFR7: Pruning must integrate seamlessly with existing agent lifecycle hooks without disrupting current workflows
NFR8: Pruning must work correctly with Podman's image and artifact management APIs
NFR9: The references file must be maintained accurately even when pruning is disabled to ensure correct behavior when pruning is later enabled
NFR10: Pruning must handle images and artifacts separately throughout the process for accurate categorization and removal

## Epic 1: Core Automatic Image and Artifact Pruning

Devices automatically remove unused images and artifacts after successful updates by tracking reference history and only removing items that have lost their references.

**FRs covered:** FR1-17

**User Outcome:** Fleet operators no longer need to manually clean up unused images and artifacts. Devices automatically prune items that were previously referenced but are no longer needed, while intelligently preserving all currently referenced items including OS images.

**Implementation Notes:**
- Maintain accumulated references file tracking all ever-referenced images/artifacts
- Record references before upgrades start
- Only prune items that were previously referenced but are no longer referenced
- Handle images and artifacts separately throughout
- Extract nested OCI targets from image-based applications
- Remove successfully pruned items from references file

### Story 1.1: Extend Podman Client with Image and Artifact Management Methods

**Files Modified:** `internal/agent/client/podman.go`, `internal/agent/client/podman_test.go`

As a developer,
I want the Podman client to support listing and removing images and artifacts,
So that the pruning manager can query and remove unused items.

**Acceptance Criteria:**

**Given** the Podman client exists with existing image management methods
**When** I call `ListImages(ctx context.Context) ([]string, error)`
**Then** it returns a list of all container images stored on the device
**And** it handles errors gracefully and returns them wrapped

**Given** the Podman client exists
**When** I call `ListArtifacts(ctx context.Context) ([]string, error)`
**Then** it returns a list of all OCI artifacts stored on the device
**And** it handles errors gracefully and returns them wrapped

**Given** the Podman client exists
**When** I call `RemoveImage(ctx context.Context, image string) error`
**Then** it removes the specified container image from Podman
**And** it returns an error if the removal fails
**And** it handles non-existent images gracefully

**Given** the Podman client exists
**When** I call `RemoveArtifact(ctx context.Context, artifact string) error`
**Then** it removes the specified OCI artifact from Podman
**And** it returns an error if the removal fails
**And** it handles non-existent artifacts gracefully

**Given** the new Podman client methods
**When** I run the test suite
**Then** all new methods have unit tests with gomock
**And** tests cover success and failure scenarios

**Requirements Fulfilled:** FR8, FR9 (query capabilities), foundation for FR3, FR4 (removal capabilities)

### Story 1.2: Implement Complete Pruning Manager

**Files Modified:** `internal/agent/device/pruning/manager.go`, `internal/agent/device/pruning/manager_test.go`

As a developer,
I want a complete pruning manager that handles all pruning operations,
So that unused images and artifacts can be automatically removed while preserving required items.

**Acceptance Criteria:**

**Component 1: Manager Structure**

**Given** the existing agent manager patterns
**When** I create the pruning manager package at `internal/agent/device/pruning/`
**Then** it follows the manager pattern (public Manager interface, private manager struct)
**And** it includes a `NewManager()` constructor with dependency injection
**And** it includes interface verification: `var _ Manager = (*manager)(nil)`
**And** the Manager interface includes `Prune(ctx context.Context) error` method
**And** the Manager interface includes `RecordReferences(ctx context.Context) error` method

**Component 2: Reference Extraction from Specs**

**Given** current and desired device specs exist
**When** I call `getImageReferencesFromSpecs(ctx context.Context) ([]string, error)`
**Then** it reads both current and desired specs using the spec manager
**And** it extracts all container image references from both specs
**And** it extracts all OCI artifact references from both specs
**And** it returns a combined unique list of image/artifact identifiers
**And** it handles missing desired spec gracefully (optional)

**Given** a device spec with applications
**When** I extract image references
**Then** it identifies images from all application types (Compose, Quadlet, Container)
**And** it extracts images from application volumes
**And** it extracts OS images if present
**And** it extracts nested OCI targets from image-based applications (using OCITargetCache)
**And** it handles nested extraction failures gracefully (best-effort)

**Component 3: Reference Recording and File Management**

**Given** an upgrade is about to start
**When** I call `RecordReferences(ctx context.Context) error`
**Then** it reads the existing references file if it exists
**And** it extracts current references from current and desired specs
**And** it merges new references with existing references (accumulation)
**And** it categorizes references as images or artifacts based on Podman queries
**And** it writes the accumulated references to the file

**Given** multiple upgrades occur
**When** I record references before each upgrade
**Then** the file accumulates references from all upgrades
**And** previously referenced items remain in the file
**And** new references are added to the file

**Given** pruning is disabled
**When** upgrades occur
**Then** references are still recorded before each upgrade
**And** the file is maintained even when pruning is disabled

**Given** items have been successfully pruned
**When** I call `removePrunedReferencesFromFile(removedImages []string, removedArtifacts []string) error`
**Then** it removes successfully pruned images and artifacts from the references file
**And** it writes the updated file back

**Component 4: Eligibility Determination**

**Given** a references file exists with previously referenced items
**When** I call `determineEligibleImages(ctx context.Context) (*EligibleItems, error)`
**Then** it reads the previous references from the file
**And** it queries Podman for all container images and artifacts
**And** it gets current required references from current and desired specs
**And** it categorizes current references as images or artifacts
**And** it identifies items that were previously referenced but are no longer referenced (FR1, FR2)
**And** images are compared only with previously referenced images
**And** artifacts are compared only with previously referenced artifacts
**And** items referenced in current or desired specs are NOT included in the eligible list (FR10)
**And** OS images that are still referenced are preserved (FR11)
**And** OS images that lost their references can be pruned (FR12)

**Component 5: Removal Logic**

**Given** eligible images and artifacts have been determined
**When** I call the removal logic
**Then** it removes each eligible container image via Podman client (FR3)
**And** it removes each eligible OCI artifact via Podman client (FR4)
**And** it tracks which items were successfully removed
**And** it logs each successful removal
**And** it logs warnings for failed removals but continues with next item
**And** it validates that required images still exist after pruning

**Component 6: Pruning Orchestration**

**Given** pruning is enabled in configuration
**When** I call `Prune(ctx context.Context) error`
**Then** it checks if pruning is enabled (returns early if disabled)
**And** it calls `determineEligibleImages()` to find items to prune
**And** if no items are eligible, it calls `recordImageArtifactReferences()` to update the file
**And** if items are eligible, it calls removal logic
**And** after successful removal, it calls `removePrunedReferencesFromFile()` to clean up the file
**And** it calls `validateCapability()` to verify required images still exist
**And** it calls `recordImageArtifactReferences()` to update the file with new references
**And** it logs the complete operation

**Given** pruning encounters errors at any stage
**When** an error occurs
**Then** it logs warnings but doesn't block reconciliation
**And** it continues with subsequent steps where possible
**And** it never returns an error that would block reconciliation

**Component 7: Comprehensive Testing**

**Given** the complete pruning manager implementation
**When** I run unit tests
**Then** tests cover manager structure and initialization
**And** tests verify reference extraction from specs (including nested targets)
**And** tests verify reference recording and accumulation
**And** tests verify eligibility determination (lost references approach)
**And** tests verify removal logic (images and artifacts separately)
**And** tests verify complete pruning workflow
**And** tests verify error handling at each stage
**And** tests verify logging occurs appropriately
**And** tests verify the workflow is idempotent

**Implementation Notes:**
- Create complete manager.go with all functions:
  - Structure: Manager interface, manager struct, NewManager()
  - Extraction: getImageReferencesFromSpecs, extractImageReferences, extractImagesFromApplication, extractComposeImages, extractQuadletImages, extractVolumeImages, extractNestedTargetsFromSpec
  - Recording: RecordReferences, recordImageArtifactReferences, readPreviousReferences, categorizeReference, removePrunedReferencesFromFile
  - Eligibility: determineEligibleImages
  - Removal: removeEligibleImages, removeEligibleArtifacts, validateCapability
  - Orchestration: Prune()
- Add types: ImageArtifactReferences, EligibleItems
- Add constant: ReferencesFileName
- All pruning logic in this single story

**Requirements Fulfilled:** FR1-17 (all core pruning functionality), NFR1, NFR3, NFR4, NFR6, NFR9, NFR10

### Story 1.3: Integrate Pruning with Agent Lifecycle

**Files Modified:** `internal/agent/device/device.go`, `internal/agent/device/device_test.go`

As an agent,
I want pruning to execute automatically after successful spec reconciliation,
So that unused images and artifacts are cleaned up without manual intervention.

**Acceptance Criteria:**

**Given** the agent has completed spec reconciliation successfully
**When** `specManager.Upgrade()` completes successfully
**Then** the agent calls `pruningManager.Prune(ctx)` after the upgrade (FR5)
**And** pruning executes after all other AfterUpdate hooks complete
**And** pruning errors are logged but don't block reconciliation (NFR3)

**Given** an upgrade is about to start
**When** `beforeUpdate()` is called and `IsUpgrading()` returns true
**Then** the agent calls `pruningManager.RecordReferences(ctx)` before any changes are applied
**And** references are recorded even if pruning is disabled
**And** recording errors are logged but don't block reconciliation

**Given** pruning is enabled in configuration
**When** Prune() is called
**Then** pruning manager executes pruning operations
**And** pruning operations execute quickly synchronously (NFR1)
**And** reconciliation continues regardless of pruning outcome

**Given** pruning is disabled in configuration
**When** Prune() is called
**Then** pruning manager returns early without executing
**And** no pruning operations occur
**And** reconciliation continues normally

**Given** the agent integration
**When** I run integration tests
**Then** tests verify RecordReferences is called before upgrades
**Then** tests verify Prune is called after successful reconciliation
**Then** tests verify pruning is not called if disabled
**And** tests verify reconciliation continues even if pruning fails
**And** tests verify pruning doesn't block other AfterUpdate operations

**Given** the pruning manager initialization
**When** the agent starts up
**Then** pruning manager is created with all required dependencies
**And** pruning manager is registered with the agent lifecycle
**And** configuration is read and applied correctly

**Implementation Notes:**
- Modify `device.go` to add pruningManager field
- Modify `beforeUpdate()` to call RecordReferences when IsUpgrading() is true
- Modify `syncDeviceSpec()` to call Prune() after Upgrade()
- Add integration tests in `device_test.go`

**Requirements Fulfilled:** FR5 (execute after reconciliation), NFR1 (non-blocking), NFR3 (fail-safe), NFR7 (seamless integration)

## Epic 2: Configuration and Control

Fleet operators can enable/disable pruning and configure behavior per device or fleet.

**FRs covered:** FR18-21

**User Outcome:** Fleet operators have control over pruning behavior, with the ability to enable/disable pruning and configure settings at the device or fleet level via file dropins, with sensible defaults (enabled by default).

**Implementation Notes:**
- Add PruningConfig struct to agent config
- Support file dropin configuration (YAML files in dropin directory)
- Implement enable/disable logic in pruning manager
- Default: pruning enabled

### Story 2.1: Add Pruning Configuration Support

**Files Modified:** `internal/agent/config/config.go`, `internal/agent/config/config_pruning_dropin_test.go`

As a fleet operator,
I want to configure pruning behavior via file dropin configuration,
So that I can enable or disable pruning per device or fleet.

**Acceptance Criteria:**

**Given** the agent configuration structure
**When** I add PruningConfig to the config
**Then** it includes an `Enabled bool` field (FR18, FR19)
**And** it follows existing config struct patterns
**And** it's readable from file dropins (YAML files in dropin directory)

**Given** the PruningConfig struct
**When** configuration is not explicitly set
**Then** the default value is `Enabled = true` (FR21)
**And** pruning is enabled by default

**Given** a file dropin with pruning configuration
**When** I set `pruning.enabled = false`
**Then** pruning operations are disabled (FR19)
**And** no pruning occurs during reconciliation
**And** references are still recorded before upgrades

**Given** a file dropin with pruning configuration
**When** I set `pruning.enabled = true`
**Then** pruning operations are enabled (FR18)
**And** pruning executes normally

**Given** the configuration implementation
**When** I run unit tests
**Then** tests verify default is enabled
**And** tests verify enable/disable flags work correctly
**And** tests verify config is read from file dropins
**And** tests verify config is read at agent startup

**Implementation Notes:**
- Add `Pruning` struct to Config
- Add `loadPruningFromDropins()` method
- Add tests in `config_pruning_dropin_test.go`
- Configuration is checked in manager.go's Prune() method (from Story 1.2)

**Requirements Fulfilled:** FR18 (enable), FR19 (disable), FR20 (apply config at device/fleet level), FR21 (default enabled)

## Epic 3: Observability and Verification

Support staff can verify pruning behavior and troubleshoot issues through comprehensive logging.

**FRs covered:** FR22-25

**User Outcome:** Support staff can verify that pruning is working correctly, see what images and artifacts were removed, and troubleshoot any issues through structured logs.

**Implementation Notes:**
- Logging is implemented throughout Stories 1.1-1.2
- No separate story needed - logging is part of each implementation story
- This epic documents the logging requirements that are fulfilled by previous stories

### Story 3.1: Verify Comprehensive Pruning Operation Logging

**Files Modified:** None (logging verified in Stories 1.1-1.2)

As support staff,
I want detailed logs of pruning operations,
So that I can verify pruning is working correctly and troubleshoot issues.

**Acceptance Criteria:**

**Given** pruning operations execute (Story 1.2)
**When** images and artifacts are removed
**Then** logs show which images were removed (FR22)
**And** logs show which artifacts were removed (FR22)
**And** logs show the operation start time (FR23)
**And** logs show the operation completion time (FR23)
**And** logs show the total number of images and artifacts removed separately

**Given** pruning operations encounter errors
**When** individual item removals fail
**Then** logs show warning messages with item name and error (FR24)
**And** logs show which items failed to remove
**And** logs continue for successful removals

**Given** reference recording operations (Story 1.2)
**When** references are recorded
**Then** logs show how many images and artifacts were recorded
**And** logs show when the references file is updated

**Given** the logging implementation
**When** I review the code from Stories 1.1-1.2
**Then** all required information is logged
**And** log format is consistent
**And** error cases are logged appropriately

**Requirements Fulfilled:** FR22 (log operations), FR23 (log timing), FR24 (log warnings), FR25 (support staff verification)

## Summary: File Ownership

| File | Story | Status |
|------|-------|--------|
| `internal/agent/client/podman.go` | 1.1 | ✅ Single owner |
| `internal/agent/client/podman_test.go` | 1.1 | ✅ Single owner |
| `internal/agent/device/pruning/manager.go` | 1.2 | ✅ Single owner |
| `internal/agent/device/pruning/manager_test.go` | 1.2 | ✅ Single owner |
| `internal/agent/config/config.go` | 2.1 | ✅ Single owner |
| `internal/agent/config/config_pruning_dropin_test.go` | 2.1 | ✅ Single owner |
| `internal/agent/device/device.go` | 1.3 | ✅ Single owner |
| `internal/agent/device/device_test.go` | 1.3 | ✅ Single owner |

**✅ All files are modified by exactly one story**

## FR Coverage Map

**Epic 1: Core Automatic Image and Artifact Pruning**
- FR1: Identify container images that lost their references (Story 1.2)
- FR2: Identify OCI artifacts that lost their references (Story 1.2)
- FR3: Remove unused application images (Story 1.2)
- FR4: Remove unused OCI artifacts (Story 1.2)
- FR5: Execute pruning after spec reconciliation (Story 1.2, 1.3)
- FR6: Determine images/artifacts in current spec (Story 1.2)
- FR7: Determine images/artifacts in desired spec (Story 1.2)
- FR8: Query Podman for all container images (Story 1.1, 1.2)
- FR9: Query Podman for all OCI artifacts (Story 1.1, 1.2)
- FR10: Preserve images/artifacts referenced in current or desired specs (Story 1.2)
- FR11: Preserve OS images that are still referenced (Story 1.2)
- FR12: Prune OS images if they lose their references (Story 1.2)
- FR13: Record all image/artifact references to file (Story 1.2)
- FR14: Accumulate references across multiple upgrades (Story 1.2)
- FR15: Remove successfully pruned items from references file (Story 1.2)
- FR16: Maintain references file when pruning is disabled (Story 1.2)
- FR17: Extract nested OCI targets from image-based applications (Story 1.2)

**Epic 2: Configuration and Control**
- FR18: Enable pruning via file dropin configuration (Story 2.1)
- FR19: Disable pruning via file dropin configuration (Story 2.1)
- FR20: Apply pruning configuration at device/fleet level (Story 2.1)
- FR21: Use default pruning settings (enabled by default) (Story 2.1)

**Epic 3: Observability and Verification**
- FR22: Log pruning operations (images/artifacts removed) (Story 1.2)
- FR23: Log operation start and completion times (Story 1.2)
- FR24: Log warnings when errors occur (Story 1.1, 1.2)
- FR25: Support staff can verify through logs (Story 3.1 - verification)

## Epic List

### Epic 1: Core Automatic Image and Artifact Pruning

Devices automatically remove unused images and artifacts after successful updates by tracking reference history and only removing items that have lost their references.

**FRs covered:** FR1-17

**User Outcome:** Fleet operators no longer need to manually clean up unused images and artifacts. Devices automatically prune items that were previously referenced but are no longer needed, while intelligently preserving all currently referenced items including OS images.

**Implementation Notes:**
- Maintain accumulated references file tracking all ever-referenced images/artifacts
- Record references before upgrades start
- Only prune items that were previously referenced but are no longer referenced
- Handle images and artifacts separately throughout
- Extract nested OCI targets from image-based applications
- Remove successfully pruned items from references file

### Epic 2: Configuration and Control

Fleet operators can enable/disable pruning and configure behavior per device or fleet.

**FRs covered:** FR18-21

**User Outcome:** Fleet operators have control over pruning behavior, with the ability to enable/disable pruning and configure settings at the device or fleet level via file dropins, with sensible defaults (enabled by default).

**Implementation Notes:**
- Add PruningConfig struct to agent config
- Support file dropin configuration (YAML files in dropin directory)
- Implement enable/disable logic in pruning manager
- Default: pruning enabled

### Epic 3: Observability and Verification

Support staff can verify pruning behavior and troubleshoot issues through comprehensive logging.

**FRs covered:** FR22-25

**User Outcome:** Support staff can verify that pruning is working correctly, see what images and artifacts were removed, and troubleshoot any issues through structured logs.

**Implementation Notes:**
- Logging is implemented throughout Stories 1.1-1.2
- No separate implementation story needed
- This epic documents the logging requirements that are fulfilled by previous stories
