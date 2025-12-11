---
stepsCompleted: [1, 2, 3, 4]
lastStep: 4
status: 'complete'
completedAt: '2025-12-09T12:36:40+02:00'
inputDocuments:
  - docs/prd.md
  - docs/architecture.md
workflowType: 'epics-stories'
project_name: 'flightctl'
user_name: 'Ori'
date: '2025-12-09T12:36:40+02:00'
---

# flightctl - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for flightctl, decomposing the requirements from the PRD, UX Design if it exists, and Architecture requirements into implementable stories.

## Requirements Inventory

### Functional Requirements

FR1: The agent can automatically identify unused container images that are no longer referenced by current or previous device specs
FR2: The agent can automatically identify unused OCI artifacts that are no longer referenced by current or previous device specs
FR3: The agent can remove unused application images from device storage
FR4: The agent can remove unused OCI artifacts from device storage
FR5: The agent can execute pruning operations after successful spec reconciliation
FR6: The agent can determine which images are referenced by the current device spec
FR7: The agent can determine which images are referenced by the previous device spec (for rollback)
FR8: The agent can query Podman for all container images stored on the device
FR9: The agent can query Podman for all OCI artifacts stored on the device
FR10: The agent can preserve the current application image for each application during pruning operations
FR11: The agent can preserve the previous application image for each application during pruning operations
FR12: The agent can preserve the current OS image during pruning operations (managed by bootc)
FR13: The agent can preserve the rollback OS image during pruning operations (managed by bootc)
FR14: The agent can verify that required images for rollback exist before executing pruning
FR15: The agent can validate that rollback capability is maintained after pruning operations
FR16: The agent can detect critical disk space alerts from the resource manager
FR17: The agent can trigger emergency pruning operations when critical disk alerts are detected
FR18: The agent can respond to critical disk alerts without requiring network connectivity
FR19: The agent can clear critical disk alerts after successful emergency pruning
FR20: Fleet operators can enable automatic image pruning via agent configuration
FR21: Fleet operators can disable automatic image pruning via agent configuration
FR22: The agent can apply pruning configuration settings at the device level
FR23: The agent can use default pruning settings when configuration is not explicitly set (default: enabled)
FR24: The agent can log pruning operations including which images were removed
FR25: The agent can log the amount of disk space reclaimed by pruning operations
FR26: The agent can log pruning operation start and completion times
FR27: The agent can log warnings when pruning operations encounter errors
FR28: Support staff can verify pruning behavior through device logs

### NonFunctional Requirements

NFR1: Pruning operations must complete within acceptable timeframes that do not block or significantly delay spec reconciliation
NFR2: Emergency pruning triggered by critical disk alerts must respond and begin execution within minutes of alert detection
NFR3: Pruning operations must not significantly impact device performance or active workloads during execution
NFR4: Pruning operations should be designed to execute asynchronously to avoid blocking the main reconciliation loop
NFR5: Pruning failures must not block spec reconciliation - failures should be logged and reconciliation should continue
NFR6: Pruning must never remove images required for rollback - system must verify required images exist before pruning
NFR7: Pruning operations must be idempotent - safe to retry if interrupted or if partial failures occur
NFR8: Rollback capability must be validated after pruning operations to ensure required images are still available
NFR9: Pruning must handle edge cases gracefully (concurrent updates, partial failures, network interruptions) without causing system instability
NFR10: Pruning must preserve system integrity even if Podman operations fail or return unexpected results
NFR11: Pruning must integrate seamlessly with existing agent lifecycle hooks (AfterUpdate) without disrupting current workflows
NFR12: Pruning must work correctly with Podman's image and artifact management APIs
NFR13: Pruning must respect bootc's OS image management - no manual pruning of OS images (bootc handles rollback automatically)
NFR14: Pruning must coordinate properly with resource manager's critical disk alert system

### Additional Requirements

**From Architecture Document:**

**Starter Template:** Not applicable - This is a brownfield project extending existing Go codebase. No starter template needed.

**Technical Implementation Requirements:**
- Extend Podman client with new methods: ListImages, ListArtifacts, RemoveImage, RemoveArtifact
- Create pruning manager component following existing manager pattern (public interface, private struct, NewManager constructor)
- Integrate pruning manager with agent lifecycle via AfterUpdate hook
- Add PruningConfig struct to agent config with Enabled bool field (default: true)
- Implement spec-based image eligibility determination (parse current.json and rollback.json)
- Follow existing error handling patterns (fmt.Errorf with %w, log but don't block)
- Use existing log.PrefixLogger for all logging operations
- Use fileio.ReadWriter interface for any file operations (not os package directly)
- Follow existing testing patterns (testify/require, gomock, table-driven tests)

**Integration Requirements:**
- Pruning manager must integrate with existing spec manager for reading current/rollback specs
- Pruning manager must integrate with existing Podman client (extend, don't replace)
- Agent must coordinate pruning via afterUpdate() method after specManager.Upgrade()
- Agent must handle alert-triggered pruning coordination (resource manager → agent → pruning manager)

**Safety Requirements:**
- Never prune OS images (bootc manages these automatically)
- Always verify required images exist before pruning
- Always validate rollback capability after pruning
- Never prune images referenced in current or rollback specs

**File Structure Requirements:**
- Create `internal/agent/device/pruning/` package
- Create `manager.go` with interface and implementation
- Create `manager_test.go` for unit tests
- Extend `internal/agent/client/podman.go` with new methods
- Modify `internal/agent/config/config.go` to add PruningConfig
- Modify `internal/agent/device/device.go` to integrate pruning manager

## Epic 1: Core Automatic Image Pruning

Devices automatically remove unused images after successful updates while preserving rollback capability.

**FRs covered:** FR1-15

**User Outcome:** Fleet operators no longer need to manually clean up unused images. Devices automatically prune unused application images and OCI artifacts after successful spec reconciliation, while intelligently preserving current and previous images needed for rollback operations.

**Implementation Notes:**
- Extend Podman client with list/remove methods
- Create pruning manager component following manager pattern
- Implement spec-based image eligibility determination
- Integrate with AfterUpdate hook after specManager.Upgrade()
- Ensure rollback safety through validation

### Story 1.1: Extend Podman Client with Image Management Methods

As a developer,
I want the Podman client to support listing and removing images and artifacts,
So that the pruning manager can query and remove unused images.

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

### Story 1.2: Create Pruning Manager Component Structure

As a developer,
I want a pruning manager component following the existing manager pattern,
So that pruning functionality integrates seamlessly with the agent architecture.

**Acceptance Criteria:**

**Given** the existing agent manager patterns
**When** I create the pruning manager package at `internal/agent/device/pruning/`
**Then** it follows the manager pattern (public Manager interface, private manager struct)
**And** it includes a `NewManager()` constructor with dependency injection
**And** it includes interface verification: `var _ Manager = (*manager)(nil)`

**Given** the pruning manager structure
**When** I define the Manager interface
**Then** it includes `Prune(ctx context.Context) error` method
**And** it includes `PruneOnAlert(ctx context.Context) error` method

**Given** the pruning manager constructor
**When** I create a new manager instance
**Then** it accepts Podman client, spec manager, file I/O, logger, and config as dependencies
**And** it returns the Manager interface (not concrete struct)
**And** all dependencies are properly initialized

**Given** the pruning manager package
**When** I run go build
**Then** it compiles without errors
**And** it follows existing code organization patterns

**Requirements Fulfilled:** Foundation for all pruning functionality, architectural pattern compliance

### Story 1.3: Implement Image Reference Extraction from Specs

As a pruning manager,
I want to extract image references from current and rollback device specs,
So that I can determine which images are required and should not be pruned.

**Acceptance Criteria:**

**Given** current and rollback device specs exist
**When** I call `extractImageReferences(ctx context.Context, device *v1beta1.Device) ([]string, error)`
**Then** it parses the device spec and extracts all container image references
**And** it extracts all OCI artifact references
**And** it returns a list of unique image/artifact identifiers
**And** it handles malformed specs gracefully with error return

**Given** a device spec with applications
**When** I extract image references
**Then** it identifies images from all application types (Compose, Quadlet, Container)
**And** it handles nested OCI artifacts correctly
**And** it returns both current and previous image versions per application

**Given** the spec manager integration
**When** I read current and rollback specs
**Then** it uses the spec manager's Read() method (not direct file access)
**And** it handles spec reading errors appropriately
**And** it returns empty list if no images are referenced

**Given** the image reference extraction
**When** I run unit tests
**Then** tests cover various spec structures (single app, multiple apps, no apps)
**And** tests verify correct extraction of current and previous images
**And** tests handle error cases (malformed specs, missing files)

**Requirements Fulfilled:** FR6, FR7 (determine images in current and rollback specs)

### Story 1.4: Implement Image Eligibility Determination

As a pruning manager,
I want to determine which images are eligible for pruning,
So that I can safely identify unused images without affecting required ones.

**Acceptance Criteria:**

**Given** all images on the device and image references from specs
**When** I call `determineEligibleImages(ctx context.Context) ([]string, error)`
**Then** it queries Podman for all container images (FR8)
**And** it queries Podman for all OCI artifacts (FR9)
**And** it compares against images referenced in current spec (FR6)
**And** it compares against images referenced in rollback spec (FR7)
**And** it identifies images not in either spec as eligible for pruning (FR1, FR2)

**Given** images referenced in current or rollback specs
**When** I determine eligible images
**Then** those images are NOT included in the eligible list
**And** current application images are preserved (FR10)
**And** previous application images are preserved (FR11)
**And** OS images are excluded (bootc manages these) (FR12, FR13)

**Given** the eligibility determination logic
**When** I run unit tests
**Then** tests verify correct identification of unused images
**And** tests verify required images are not marked eligible
**And** tests handle edge cases (empty lists, all images in use, no images on device)
**And** tests verify OS images are never marked eligible

**Given** the pruning manager
**When** eligibility determination encounters errors
**Then** it logs warnings but continues operation
**And** it returns partial results if some operations fail
**And** it never blocks reconciliation

**Requirements Fulfilled:** FR1, FR2 (identify unused images), FR8, FR9 (query Podman), FR10-13 (preserve required images)

### Story 1.5: Implement Rollback Safety Validation

As a pruning manager,
I want to validate that required images exist before and after pruning,
So that rollback capability is always maintained.

**Acceptance Criteria:**

**Given** eligible images have been determined
**When** I call `validateRequiredImages(ctx context.Context) error`
**Then** it verifies all images referenced in current spec exist (FR14)
**And** it verifies all images referenced in rollback spec exist
**And** it returns an error if any required image is missing
**And** it logs which required images are missing

**Given** pruning operations have completed
**When** I call `validateRollbackCapability(ctx context.Context) error`
**Then** it verifies current application images still exist (FR10)
**And** it verifies previous application images still exist (FR11)
**And** it verifies OS images are still managed by bootc (FR12, FR13)
**And** it returns an error if rollback capability is compromised (FR15)
**And** it logs validation results

**Given** the validation logic
**When** I run unit tests
**Then** tests verify validation passes when all required images exist
**And** tests verify validation fails when required images are missing
**And** tests verify post-pruning validation catches issues
**And** tests handle edge cases (empty specs, missing rollback spec)

**Given** validation fails before pruning
**When** pruning is attempted
**Then** pruning is aborted
**And** an error is logged
**And** reconciliation continues (doesn't block)

**Requirements Fulfilled:** FR14 (verify before pruning), FR15 (validate after pruning), FR10-13 (preserve images)

### Story 1.6: Implement Image Removal Logic

As a pruning manager,
I want to remove eligible images from device storage,
So that unused images are deleted and disk space is reclaimed.

**Acceptance Criteria:**

**Given** eligible images have been determined and validated
**When** I call the image removal logic
**Then** it removes each eligible container image via Podman client (FR3)
**And** it removes each eligible OCI artifact via Podman client (FR4)
**And** it logs each successful removal
**And** it logs warnings for failed removals but continues with next image

**Given** image removal encounters errors
**When** removing an individual image fails
**Then** it logs a warning with the image name and error
**And** it continues removing other eligible images
**And** it does not return an error (fail-safe pattern)
**And** it tracks which images were successfully removed

**Given** the removal logic
**When** I run unit tests
**Then** tests verify successful removal of images
**And** tests verify partial failures don't block other removals
**And** tests verify error logging occurs for failures
**And** tests verify removal is idempotent (safe to retry)

**Given** pruning operations complete
**When** images are removed
**Then** it logs the total number of images removed
**And** it logs the total disk space reclaimed (if available from Podman)
**And** it logs operation duration
**And** it never blocks reconciliation even if all removals fail

**Requirements Fulfilled:** FR3, FR4 (remove unused images), NFR5 (fail-safe error handling), NFR7 (idempotent)

### Story 1.7: Integrate Pruning with Agent Lifecycle

As an agent,
I want pruning to execute automatically after successful spec reconciliation,
So that unused images are cleaned up without manual intervention.

**Acceptance Criteria:**

**Given** the agent has completed spec reconciliation successfully
**When** `specManager.Upgrade()` completes successfully
**Then** the agent calls `pruningManager.Prune(ctx)` in the `afterUpdate()` method (FR5)
**And** pruning executes after all other AfterUpdate hooks complete
**And** pruning errors are logged but don't block reconciliation (NFR5)

**Given** pruning is enabled in configuration
**When** afterUpdate() is called
**Then** pruning manager is invoked
**And** pruning operations execute asynchronously or quickly synchronously (NFR4)
**And** reconciliation continues regardless of pruning outcome

**Given** pruning is disabled in configuration
**When** afterUpdate() is called
**Then** pruning manager is not invoked
**And** no pruning operations occur
**And** reconciliation continues normally

**Given** the agent integration
**When** I run integration tests
**Then** tests verify pruning is called after successful reconciliation
**Then** tests verify pruning is not called if disabled
**And** tests verify reconciliation continues even if pruning fails
**And** tests verify pruning doesn't block other AfterUpdate operations

**Given** the pruning manager initialization
**When** the agent starts up
**Then** pruning manager is created with all required dependencies
**And** pruning manager is registered with the agent lifecycle
**And** configuration is read and applied correctly

**Requirements Fulfilled:** FR5 (execute after reconciliation), NFR4 (non-blocking), NFR5 (fail-safe), NFR11 (seamless integration)

## Epic 2: Emergency Storage Recovery

Devices automatically recover from critical disk space alerts by pruning unused images.

**FRs covered:** FR16-19

**User Outcome:** When devices encounter critical disk space alerts, they automatically trigger emergency pruning to free up space without requiring manual intervention or network connectivity.

**Implementation Notes:**
- Integrate with resource manager's alert system
- Agent coordinates alert-triggered pruning
- Pruning manager implements PruneOnAlert() method
- Agent clears alerts after successful pruning

### Story 2.1: Implement Emergency Pruning Method

As a pruning manager,
I want an emergency pruning method that can be triggered on critical disk alerts,
So that devices can recover from storage exhaustion automatically.

**Acceptance Criteria:**

**Given** a critical disk space alert is detected
**When** I call `PruneOnAlert(ctx context.Context) error`
**Then** it executes the same pruning logic as normal pruning (FR17)
**And** it operates without requiring network connectivity (FR18)
**And** it logs that emergency pruning was triggered
**And** it handles errors gracefully (doesn't block alert handling)

**Given** emergency pruning is triggered
**When** pruning operations complete successfully
**Then** it logs the number of images removed
**And** it logs the disk space reclaimed
**And** it returns nil error to indicate success

**Given** emergency pruning encounters errors
**When** some images fail to remove
**Then** it logs warnings for failures
**And** it continues removing other images
**And** it returns nil (fail-safe, doesn't block alert handling)

**Given** the emergency pruning method
**When** I run unit tests
**Then** tests verify emergency pruning executes correctly
**And** tests verify it works without network connectivity
**And** tests verify error handling doesn't block operation

**Requirements Fulfilled:** FR17 (trigger emergency pruning), FR18 (work offline), NFR2 (respond within minutes)

### Story 2.2: Integrate Alert Detection and Pruning Coordination

As an agent,
I want to detect critical disk alerts and trigger emergency pruning,
So that devices can autonomously recover from storage exhaustion.

**Acceptance Criteria:**

**Given** the resource manager detects a critical disk space alert
**When** the alert is triggered (FR16)
**Then** the agent receives the alert notification
**And** the agent calls `pruningManager.PruneOnAlert(ctx)` (FR17)
**And** emergency pruning executes immediately

**Given** emergency pruning completes successfully
**When** disk space is freed
**Then** the agent clears the critical disk alert (FR19)
**And** the alert is removed from the resource manager
**And** normal operations continue

**Given** emergency pruning fails or doesn't free enough space
**When** the alert persists
**Then** the agent logs the situation
**And** the alert remains active for further action
**And** the agent doesn't retry immediately (prevents thrashing)

**Given** the alert integration
**When** I run integration tests
**Then** tests verify alert detection triggers pruning
**And** tests verify successful pruning clears alerts
**And** tests verify failed pruning doesn't clear alerts
**And** tests verify coordination doesn't block other operations

**Requirements Fulfilled:** FR16 (detect alerts), FR17 (trigger pruning), FR19 (clear alerts), NFR14 (coordinate with resource manager)

## Epic 3: Configuration and Control

Fleet operators can enable/disable pruning and configure behavior per device.

**FRs covered:** FR20-23

**User Outcome:** Fleet operators have control over pruning behavior, with the ability to enable/disable pruning and configure settings at the device level, with sensible defaults (enabled by default).

**Implementation Notes:**
- Add PruningConfig struct to agent config
- Implement enable/disable logic in pruning manager
- Support device-level configuration
- Default: pruning enabled

### Story 3.1: Add Pruning Configuration to Agent Config

As a fleet operator,
I want to configure pruning behavior via agent configuration,
So that I can enable or disable pruning per device or fleet.

**Acceptance Criteria:**

**Given** the agent configuration structure
**When** I add PruningConfig to the config
**Then** it includes an `Enabled bool` field (FR20, FR21)
**And** it follows existing config struct patterns
**And** it's readable from the agent config file

**Given** the PruningConfig struct
**When** configuration is not explicitly set
**Then** the default value is `Enabled = true` (FR23)
**And** pruning is enabled by default

**Given** the agent configuration
**When** I set `PruningConfig.Enabled = false`
**Then** pruning operations are disabled (FR21)
**And** no pruning occurs during reconciliation or alerts

**Given** the agent configuration
**When** I set `PruningConfig.Enabled = true`
**Then** pruning operations are enabled (FR20)
**And** pruning executes normally

**Given** the configuration implementation
**When** I run unit tests
**Then** tests verify default is enabled
**And** tests verify enable/disable flags work correctly
**And** tests verify config is read at agent startup

**Requirements Fulfilled:** FR20 (enable), FR21 (disable), FR23 (default enabled)

### Story 3.2: Implement Configuration Application Logic

As a pruning manager,
I want to read and apply configuration settings,
So that pruning behavior can be controlled per device.

**Acceptance Criteria:**

**Given** the pruning manager is initialized
**When** configuration is provided
**Then** it reads the PruningConfig from agent config (FR22)
**And** it applies the settings at initialization
**And** it stores the config for runtime checks

**Given** pruning is disabled in configuration
**When** Prune() or PruneOnAlert() is called
**Then** it checks the configuration first
**And** it returns early without executing pruning
**And** it logs that pruning is disabled

**Given** pruning is enabled in configuration
**When** Prune() or PruneOnAlert() is called
**Then** it proceeds with normal pruning operations
**And** it uses the configuration settings

**Given** the configuration logic
**When** I run unit tests
**Then** tests verify pruning respects enable/disable flag
**And** tests verify config is checked before operations
**And** tests verify default behavior when config is not set

**Requirements Fulfilled:** FR22 (apply config at device level), FR20, FR21, FR23 (configuration control)

## Epic 4: Observability and Verification

Support staff can verify pruning behavior and troubleshoot issues through comprehensive logging.

**FRs covered:** FR24-28

**User Outcome:** Support staff can verify that pruning is working correctly, see what images were removed, how much space was reclaimed, and troubleshoot any issues through structured logs.

**Implementation Notes:**
- Implement structured logging throughout pruning operations
- Log images removed, space reclaimed, operation timing
- Log warnings for errors that don't block operation
- Use existing log.PrefixLogger infrastructure

### Story 4.1: Implement Comprehensive Pruning Operation Logging

As support staff,
I want detailed logs of pruning operations,
So that I can verify pruning is working correctly and troubleshoot issues.

**Acceptance Criteria:**

**Given** pruning operations execute
**When** images are removed
**Then** it logs which images were removed (FR24)
**And** it logs the operation start time (FR26)
**And** it logs the operation completion time (FR26)
**And** it logs the total number of images removed

**Given** pruning operations complete
**When** disk space is reclaimed
**Then** it logs the amount of disk space reclaimed in bytes (FR25)
**And** it logs the operation duration
**And** it uses structured logging format

**Given** pruning operations encounter errors
**When** individual image removals fail
**Then** it logs warnings with image name and error (FR27)
**And** it logs which images failed to remove
**And** it continues logging for successful removals

**Given** the logging implementation
**When** I run unit tests
**Then** tests verify all required information is logged
**And** tests verify log format is consistent
**And** tests verify error cases are logged appropriately

**Requirements Fulfilled:** FR24 (log operations), FR25 (log space reclaimed), FR26 (log timing), FR27 (log warnings)

### Story 4.2: Enable Support Staff Verification Through Logs

As support staff,
I want to verify pruning behavior through device logs,
So that I can confirm pruning is working and diagnose issues.

**Acceptance Criteria:**

**Given** pruning operations have executed
**When** support staff reviews device logs
**Then** they can see pruning operation entries (FR28)
**And** they can identify which images were removed
**And** they can see how much space was reclaimed
**And** they can see operation timing information

**Given** pruning encountered errors
**When** support staff reviews logs
**Then** they can see warning messages for failures (FR27)
**And** they can identify which specific images failed
**And** they can see error details for troubleshooting

**Given** pruning is disabled
**When** support staff reviews logs
**Then** they can see log entries indicating pruning is disabled
**And** they can verify configuration is applied correctly

**Given** emergency pruning is triggered
**When** support staff reviews logs
**Then** they can see emergency pruning was triggered
**And** they can see the results of emergency pruning
**And** they can verify alert clearing

**Given** the logging infrastructure
**When** I run integration tests
**Then** tests verify logs are accessible and readable
**And** tests verify log entries contain all required information
**And** tests verify logs can be used for troubleshooting

**Requirements Fulfilled:** FR28 (support staff verification), FR24-27 (comprehensive logging)

### FR Coverage Map

**Epic 1: Core Automatic Image Pruning**
- FR1: Identify unused container images
- FR2: Identify unused OCI artifacts
- FR3: Remove unused application images
- FR4: Remove unused OCI artifacts
- FR5: Execute pruning after spec reconciliation
- FR6: Determine images in current spec
- FR7: Determine images in previous spec (rollback)
- FR8: Query Podman for all container images
- FR9: Query Podman for all OCI artifacts
- FR10: Preserve current application images
- FR11: Preserve previous application images
- FR12: Preserve current OS images (bootc managed)
- FR13: Preserve rollback OS images (bootc managed)
- FR14: Verify required images exist before pruning
- FR15: Validate rollback capability after pruning

**Epic 2: Emergency Storage Recovery**
- FR16: Detect critical disk space alerts
- FR17: Trigger emergency pruning on alerts
- FR18: Respond to alerts without network connectivity
- FR19: Clear alerts after successful emergency pruning

**Epic 3: Configuration and Control**
- FR20: Enable pruning via agent configuration
- FR21: Disable pruning via agent configuration
- FR22: Apply pruning configuration at device level
- FR23: Use default pruning settings (enabled by default)

**Epic 4: Observability and Verification**
- FR24: Log pruning operations (images removed)
- FR25: Log disk space reclaimed
- FR26: Log operation start and completion times
- FR27: Log warnings when errors occur
- FR28: Support staff can verify through logs

## Epic List

### Epic 1: Core Automatic Image Pruning

Devices automatically remove unused images after successful updates while preserving rollback capability.

**FRs covered:** FR1-15

**User Outcome:** Fleet operators no longer need to manually clean up unused images. Devices automatically prune unused application images and OCI artifacts after successful spec reconciliation, while intelligently preserving current and previous images needed for rollback operations.

**Implementation Notes:**
- Extend Podman client with list/remove methods
- Create pruning manager component following manager pattern
- Implement spec-based image eligibility determination
- Integrate with AfterUpdate hook after specManager.Upgrade()
- Ensure rollback safety through validation

### Epic 2: Emergency Storage Recovery

Devices automatically recover from critical disk space alerts by pruning unused images.

**FRs covered:** FR16-19

**User Outcome:** When devices encounter critical disk space alerts, they automatically trigger emergency pruning to free up space without requiring manual intervention or network connectivity.

**Implementation Notes:**
- Integrate with resource manager's alert system
- Agent coordinates alert-triggered pruning
- Pruning manager implements PruneOnAlert() method
- Agent clears alerts after successful pruning

### Epic 3: Configuration and Control

Fleet operators can enable/disable pruning and configure behavior per device.

**FRs covered:** FR20-23

**User Outcome:** Fleet operators have control over pruning behavior, with the ability to enable/disable pruning and configure settings at the device level, with sensible defaults (enabled by default).

**Implementation Notes:**
- Add PruningConfig struct to agent config
- Implement enable/disable logic in pruning manager
- Support device-level configuration
- Default: pruning enabled

### Epic 4: Observability and Verification

Support staff can verify pruning behavior and troubleshoot issues through comprehensive logging.

**FRs covered:** FR24-28

**User Outcome:** Support staff can verify that pruning is working correctly, see what images were removed, how much space was reclaimed, and troubleshoot any issues through structured logs.

**Implementation Notes:**
- Implement structured logging throughout pruning operations
- Log images removed, space reclaimed, operation timing
- Log warnings for errors that don't block operation
- Use existing log.PrefixLogger infrastructure

