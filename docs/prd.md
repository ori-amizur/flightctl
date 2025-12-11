---
stepsCompleted: [1, 2, 3, 4, 7, 8, 9, 10]
inputDocuments:
  - docs/index.md
  - docs/project-overview.md
  - docs/developer/architecture/architecture.md
documentCounts:
  briefs: 0
  research: 0
  brainstorming: 0
  projectDocs: 1
workflowType: 'prd'
lastStep: 11
project_name: 'flightctl'
user_name: 'Ori'
date: '2025-12-08T18:38:40+02:00'
---

# Product Requirements Document - flightctl

**Author:** Ori
**Date:** 2025-12-08T18:38:40+02:00

## Executive Summary

This PRD defines the implementation of automatic image pruning for FlightCtl-managed devices. The feature addresses a critical operational gap where devices accumulate unused container images and OCI artifacts over time, leading to storage exhaustion. The solution automatically removes unused images while intelligently preserving those required for rollback operations, ensuring devices can recover from failed updates even without network connectivity.

### What Makes This Special

This feature introduces intelligent, automated storage management to FlightCtl's device management capabilities. Unlike manual cleanup approaches, this solution:

- **Proactively prevents storage exhaustion** by automatically removing unused images after successful spec reconciliation
- **Preserves rollback capability** by retaining current and previous application/OS images, ensuring recovery works offline
- **Responds to critical alerts** by triggering pruning when disk space becomes critical, preventing device failures
- **Maintains operational safety** through configurable controls and comprehensive testing of rollback scenarios

The key differentiator is the balance between aggressive cleanup (to prevent storage issues) and conservative retention (to ensure rollback reliability), all while operating autonomously on edge devices that may have intermittent network connectivity.

## Project Classification

**Technical Type:** Backend Service Enhancement (Agent Functionality)
**Domain:** Device Management / Infrastructure Automation
**Complexity:** Medium
**Project Context:** Brownfield - extending existing FlightCtl agent system

This enhancement integrates with existing FlightCtl agent components including spec reconciliation, container/OCI image management, rollback mechanisms (greenboot for OS, application rollback), and the alert system. It extends the agent's lifecycle management capabilities to include storage management while maintaining compatibility with existing device management workflows.

## Success Criteria

### User Success

**Fleet Operators:**
- Zero manual intervention required for image cleanup - operators no longer need to manually remove unused images
- Devices never fail due to storage exhaustion - automatic pruning prevents critical disk space issues
- Rollback always works when needed - previous images are preserved, enabling reliable recovery even without network access
- Reduced operational overhead - automated storage management reduces manual maintenance tasks
- Reduced risk of storage exhaustion - proactive pruning prevents devices from running out of space

**Success Moment:** Operators see devices automatically clean up unused images while maintaining rollback capability, confirming that autonomous storage management is working.

**Outcome:** Operators gain confidence that devices self-manage storage without manual intervention, reducing operational burden and preventing storage-related failures.

### Business Success

**Primary Metric:** Zero unpruned files that should have been cleaned
- Success = no unused images remain on devices after pruning runs
- Measurable: audit shows all eligible images are removed while required images are retained

**Operational Impact:**
- Reduced support tickets related to storage exhaustion
- Improved device reliability and uptime
- Lower operational costs from reduced manual maintenance

**Success Indicator:** Fleet operators report they never see storage-related issues or need to manually clean up images.

### Technical Success

**Performance Metrics:**
- **Pruning Time:** Pruning completes within acceptable timeframes (to be defined based on device capabilities and image counts)
- **Disk Space Reclaimed:** Measurable reduction in disk usage after pruning operations
- **Pruning Efficiency:** System successfully identifies and removes unused images without impacting active workloads

**Reliability Metrics:**
- **Rollback Success Rate:** 100% rollback success after pruning (validated through testing)
- **Zero Data Loss:** No required images are accidentally removed during pruning
- **Alert Response:** Critical disk alerts trigger pruning successfully

**Functional Requirements (from Definition of Done):**
- Agent prunes application images no longer referenced by current or previous spec
- OS images retain current booted image plus one rollback image (bootc default behavior)
- Application images retain current plus previous version per application
- Pruning runs after successful spec reconciliation and when critical disk alert is firing
- Unit tests cover pruning logic and edge cases
- Tests validate rollback works after pruning
- Documentation updated with pruning behavior
- Agent configuration exposes knob to disable pruning

### Measurable Outcomes

**Immediate (Post-Deployment):**
- All devices in fleet have automatic pruning enabled
- Zero manual cleanup tasks required by operators
- Pruning runs successfully after each spec reconciliation

**Short-term (3 months):**
- Zero storage exhaustion incidents across fleet
- 100% rollback success rate after pruning operations
- Measurable disk space reclaimed (target TBD based on fleet analysis)

**Long-term (12 months):**
- Storage management fully autonomous across all devices
- Operational overhead reduction measurable in support ticket reduction
- Zero unpruned files that should have been cleaned (business success metric)

## Product Scope

### MVP - Minimum Viable Product

**Core Functionality:**
- Automatic pruning of unused application images after successful spec reconciliation
- Retention of current + previous application images per application
- Retention of current + previous OS images (leveraging bootc default behavior)
- Pruning triggered on critical disk alerts
- Agent configuration knob to disable pruning
- Unit tests for pruning logic
- Rollback validation tests

**Success Criteria:**
- Pruning works correctly without breaking rollback
- Configurable via agent settings
- Basic performance acceptable (pruning completes without device impact)

### Growth Features (Post-MVP)

**Enhanced Monitoring:**
- Metrics and observability for pruning operations (pruning time, space reclaimed)
- Dashboard visibility into pruning effectiveness across fleet
- Alerting on pruning failures or anomalies

**Advanced Configuration:**
- Per-fleet pruning policies
- Customizable retention policies (beyond current + previous)
- Pruning schedules/configurable triggers

**Optimization:**
- Performance improvements based on real-world usage
- Intelligent pruning strategies (e.g., prioritize larger images)
- Pruning analytics and reporting

### Vision (Future)

**Intelligent Storage Management:**
- Predictive pruning based on usage patterns
- Integration with storage quota management
- Multi-layer storage optimization (images, logs, temporary files)
- Cross-device learning for optimal pruning strategies

**Advanced Safety:**
- Machine learning-based image importance scoring
- Enhanced rollback scenarios (multiple rollback points)
- Storage forecasting and capacity planning

## User Journeys

### Journey 1: Sarah Chen - Fleet Operator (Happy Path)

Sarah manages a fleet of 500 edge devices deployed across multiple factory sites. She regularly updates application versions to deploy new features and security patches. Previously, she had to manually clean up unused images every few weeks, which was time-consuming and sometimes missed devices.

**The Story:**
After the image pruning feature is deployed, Sarah updates a fleet spec to version 2.3.0. Devices automatically pull the new application image and reconcile successfully. The agent automatically prunes unused images from version 2.1.0 (skipping 2.2.0, which is kept for rollback). Sarah checks her fleet dashboard and sees storage usage normalized across all devices without any manual intervention.

**The Breakthrough:**
Three months later, Sarah reviews her operational metrics and realizes she hasn't had a single storage-related incident. Devices are self-managing their storage, and when she tests rollback scenarios, she confirms previous images are always preserved. She no longer needs to allocate time for manual cleanup tasks.

### Journey 2: Marcus Rodriguez - Fleet Operator (Critical Alert Response)

Marcus manages devices in remote locations with limited connectivity. One morning, he receives an alert that a device has triggered a critical disk space warning at 95% capacity.

**The Story:**
The agent detects the critical disk alert and immediately triggers emergency pruning. It intelligently removes unused images while preserving current and previous versions needed for rollback. Within minutes, disk usage drops to 78%, the alert clears, and the device continues operating normally.

**The Breakthrough:**
The device recovered autonomously without requiring manual intervention or network access. Marcus receives a notification that pruning successfully resolved the alert, and he can verify that rollback capability remains intact for future updates.

### Journey 3: David Kim - Device Administrator (Configuration)

David needs to configure pruning for a new fleet with specific requirements. Some devices have limited storage capacity, while others need longer retention periods for compliance reasons.

**The Story:**
David opens the fleet configuration interface and enables automatic pruning. He reviews the default settings (current + previous image retention) and confirms they meet his requirements. For a compliance-sensitive fleet, he verifies the disable option is available if needed. He saves the configuration and monitors the first pruning cycle to ensure it works as expected.

**The Breakthrough:**
David sees pruning working correctly across different fleet types with appropriate configurations. He has control when needed while benefiting from sensible defaults that work for most use cases.

### Journey 4: Lisa Thompson - Support Staff (Troubleshooting)

A customer reports that a device failed to rollback after an update. Lisa needs to investigate whether pruning accidentally removed a required image.

**The Story:**
Lisa checks the device logs and sees that pruning ran successfully, retaining both current and previous images as designed. She verifies the rollback failure is unrelated to pruning - the previous image exists and is valid. She confirms pruning behavior is correct and focuses her troubleshooting on the actual rollback issue.

**The Breakthrough:**
Lisa can quickly verify pruning behavior through logs and metrics, ruling out pruning as the cause and focusing on the real issue. The observability features help her provide accurate support.

### Journey Requirements Summary

These journeys reveal requirements for:

- **Automatic Pruning Execution** - Pruning runs automatically after successful spec reconciliation without operator intervention
- **Alert-Triggered Pruning** - Critical disk alerts trigger immediate pruning to prevent storage exhaustion
- **Configuration Management** - Fleet/device-level configuration for enabling/disabling pruning and retention policies
- **Observability & Logging** - Comprehensive logs and metrics to verify pruning behavior, space reclaimed, and retained images
- **Rollback Preservation** - Guaranteed retention of current + previous images for reliable rollback capability
- **Dashboard Visibility** - Fleet operators can see pruning status and storage metrics across their fleet
- **Notification System** - Alerts when pruning successfully resolves critical disk issues

## Backend Service Enhancement (Agent Functionality) - Specific Requirements

### Project-Type Overview

This feature extends the FlightCtl agent's lifecycle management capabilities to include automatic image pruning. The agent follows a reconciliation-based architecture where:
- Specs are managed through `desired.json`, `current.json`, and `rollback.json` files
- Reconciliation happens in `syncDeviceSpec()` → `sync()` → `afterUpdate()` flow
- Managers implement hooks: `BeforeUpdate()`, `Sync()`, `AfterUpdate()`
- Images are managed via Podman client for both container images and OCI artifacts

### Technical Architecture Considerations

**Integration Points:**
1. **AfterUpdate Hook** - Pruning should trigger after successful spec reconciliation, specifically after `specManager.Upgrade()` completes successfully
2. **Podman Client Extension** - Need to add methods to `internal/agent/client/podman.go`:
   - `ListImages(ctx context.Context) ([]string, error)` - List all images
   - `ListArtifacts(ctx context.Context) ([]string, error)` - List all artifacts
   - `RemoveImage(ctx context.Context, image string) error` - Remove specific image
   - `RemoveArtifact(ctx context.Context, artifact string) error` - Remove specific artifact
3. **Applications Manager** - Extend `internal/agent/device/applications/manager.go` to track image usage:
   - Determine which images are referenced by current spec
   - Determine which images are referenced by previous spec (for rollback)
   - Identify unused images eligible for pruning
4. **OS Manager** - Leverage existing `internal/agent/device/os/os.go`:
   - OS images are managed by bootc (handles rollback automatically)
   - Pruning should respect bootc's rollback image retention

**State Management:**
- Track images referenced in `current.json` and `rollback.json` specs
- For applications: track current + previous image per application
- For OS: rely on bootc's built-in rollback mechanism (current + one rollback)
- No persistent state needed - determine eligible images on each pruning run

**Error Handling:**
- Pruning failures should not block reconciliation
- Log warnings but continue operation
- Implement retry logic for transient failures
- Validate rollback capability after pruning (test that required images still exist)

### Implementation Considerations

**Pruning Logic:**
1. **Trigger Points:**
   - After successful `specManager.Upgrade()` (normal reconciliation)
   - On critical disk alert from resource manager
2. **Image Identification:**
   - Query Podman for all images/artifacts
   - Extract image references from `current.json` and `rollback.json`
   - For each application: identify current and previous image versions
   - Mark images not in either set as eligible for pruning
3. **Safety Checks:**
   - Verify required images exist before pruning
   - Never prune images referenced in current or rollback specs
   - For OS: rely on bootc's management (don't prune OS images manually)
4. **Configuration:**
   - Add agent config flag to enable/disable pruning
   - Default: enabled
   - Configurable retention policy (current + previous is default)

**Testing Requirements:**
- Unit tests for pruning logic (identifying eligible images)
- Integration tests for pruning after reconciliation
- Rollback validation tests (ensure rollback works after pruning)
- Edge case tests (concurrent updates, partial failures)

**Observability:**
- Log pruning operations (images removed, space reclaimed)
- Metrics for pruning time, space reclaimed, images removed
- Alert on pruning failures

## Project Scoping & Phased Development

### MVP Strategy & Philosophy

**MVP Approach:** Problem-Solving MVP - Solve the core storage exhaustion problem with minimal features that deliver immediate operational value.

**Resource Requirements:** 
- Medium scope project requiring focused development team
- Core skills: Go backend development, container/OCI image management, agent lifecycle integration
- Estimated team: 1-2 developers for MVP implementation

**Strategic Rationale:**
The MVP focuses on solving the critical operational problem (storage exhaustion) while maintaining system safety (rollback capability). This approach delivers immediate value to fleet operators while establishing a foundation for future enhancements.

### MVP Feature Set (Phase 1)

**Core User Journeys Supported:**
- Fleet Operator Happy Path (Journey 1) - Automatic pruning after reconciliation
- Fleet Operator Critical Alert (Journey 2) - Emergency pruning on disk alerts
- Device Administrator Configuration (Journey 3) - Basic enable/disable configuration

**Must-Have Capabilities:**
1. Automatic pruning of unused application images after successful spec reconciliation
2. Retention of current + previous application images per application (rollback safety)
3. Retention of current + previous OS images (leveraging bootc default behavior)
4. Pruning triggered on critical disk alerts from resource manager
5. Agent configuration flag to enable/disable pruning (default: enabled)
6. Unit tests for pruning logic and edge cases
7. Rollback validation tests to ensure pruning doesn't break rollback
8. Basic logging of pruning operations

**MVP Exclusions (Deferred to Post-MVP):**
- Advanced monitoring/metrics dashboard
- Per-fleet pruning policies
- Customizable retention policies beyond current + previous
- Pruning analytics and reporting
- Alerting on pruning failures (basic logging only)

### Post-MVP Features

**Phase 2 (Growth - Post-MVP):**
- Enhanced monitoring and observability:
  - Metrics for pruning operations (pruning time, space reclaimed)
  - Dashboard visibility into pruning effectiveness across fleet
  - Alerting on pruning failures or anomalies
- Advanced configuration:
  - Per-fleet pruning policies
  - Customizable retention policies (beyond current + previous)
  - Pruning schedules/configurable triggers
- Optimization:
  - Performance improvements based on real-world usage
  - Intelligent pruning strategies (e.g., prioritize larger images)

**Phase 3 (Expansion - Future):**
- Intelligent Storage Management:
  - Predictive pruning based on usage patterns
  - Integration with storage quota management
  - Multi-layer storage optimization (images, logs, temporary files)
  - Cross-device learning for optimal pruning strategies
- Advanced Safety:
  - Machine learning-based image importance scoring
  - Enhanced rollback scenarios (multiple rollback points)
  - Storage forecasting and capacity planning

### Risk Mitigation Strategy

**Technical Risks:**
- **Risk:** Pruning logic complexity and edge cases
- **Mitigation:** Comprehensive unit and integration testing, rollback validation tests, phased rollout with monitoring
- **Fallback:** Disable pruning via configuration if issues arise

**Market Risks:**
- **Risk:** Low - addresses known operational problem with clear user need
- **Mitigation:** MVP focuses on proven use case (storage exhaustion prevention)
- **Validation:** Early adopter feedback on pruning effectiveness

**Resource Risks:**
- **Risk:** Medium scope may require more resources than available
- **Mitigation:** Strict MVP boundaries, defer non-essential features to Phase 2
- **Contingency:** Can launch with even more minimal feature set (pruning only, no alert triggering) if needed

## Functional Requirements

### Image Pruning Management

- FR1: The agent can automatically identify unused container images that are no longer referenced by current or previous device specs
- FR2: The agent can automatically identify unused OCI artifacts that are no longer referenced by current or previous device specs
- FR3: The agent can remove unused application images from device storage
- FR4: The agent can remove unused OCI artifacts from device storage
- FR5: The agent can execute pruning operations after successful spec reconciliation
- FR6: The agent can determine which images are referenced by the current device spec
- FR7: The agent can determine which images are referenced by the previous device spec (for rollback)
- FR8: The agent can query Podman for all container images stored on the device
- FR9: The agent can query Podman for all OCI artifacts stored on the device

### Rollback Safety

- FR10: The agent can preserve the current application image for each application during pruning operations
- FR11: The agent can preserve the previous application image for each application during pruning operations
- FR12: The agent can preserve the current OS image during pruning operations (managed by bootc)
- FR13: The agent can preserve the rollback OS image during pruning operations (managed by bootc)
- FR14: The agent can verify that required images for rollback exist before executing pruning
- FR15: The agent can validate that rollback capability is maintained after pruning operations

### Alert Response

- FR16: The agent can detect critical disk space alerts from the resource manager
- FR17: The agent can trigger emergency pruning operations when critical disk alerts are detected
- FR18: The agent can respond to critical disk alerts without requiring network connectivity
- FR19: The agent can clear critical disk alerts after successful emergency pruning

### Configuration Management

- FR20: Fleet operators can enable automatic image pruning via agent configuration
- FR21: Fleet operators can disable automatic image pruning via agent configuration
- FR22: The agent can apply pruning configuration settings at the device level
- FR23: The agent can use default pruning settings when configuration is not explicitly set (default: enabled)

### Observability

- FR24: The agent can log pruning operations including which images were removed
- FR25: The agent can log the amount of disk space reclaimed by pruning operations
- FR26: The agent can log pruning operation start and completion times
- FR27: The agent can log warnings when pruning operations encounter errors
- FR28: Support staff can verify pruning behavior through device logs

## Non-Functional Requirements

### Performance

- NFR1: Pruning operations must complete within acceptable timeframes that do not block or significantly delay spec reconciliation
- NFR2: Emergency pruning triggered by critical disk alerts must respond and begin execution within minutes of alert detection
- NFR3: Pruning operations must not significantly impact device performance or active workloads during execution
- NFR4: Pruning operations should be designed to execute asynchronously to avoid blocking the main reconciliation loop

### Reliability

- NFR5: Pruning failures must not block spec reconciliation - failures should be logged and reconciliation should continue
- NFR6: Pruning must never remove images required for rollback - system must verify required images exist before pruning
- NFR7: Pruning operations must be idempotent - safe to retry if interrupted or if partial failures occur
- NFR8: Rollback capability must be validated after pruning operations to ensure required images are still available
- NFR9: Pruning must handle edge cases gracefully (concurrent updates, partial failures, network interruptions) without causing system instability
- NFR10: Pruning must preserve system integrity even if Podman operations fail or return unexpected results

### Integration

- NFR11: Pruning must integrate seamlessly with existing agent lifecycle hooks (AfterUpdate) without disrupting current workflows
- NFR12: Pruning must work correctly with Podman's image and artifact management APIs
- NFR13: Pruning must respect bootc's OS image management - no manual pruning of OS images (bootc handles rollback automatically)
- NFR14: Pruning must coordinate properly with resource manager's critical disk alert system

