# Image and Artifact Pruning Design Document

## Overview

The pruning functionality automatically removes unused container images and OCI artifacts from device storage after successful spec reconciliation. It ensures that only images and artifacts that are no longer referenced by the current or desired device specifications are removed, preserving all required resources for device operations.

## Goals

1. **Automatic Cleanup**: Remove unused images and artifacts to free up disk space
2. **Safety First**: Never remove images/artifacts that are still referenced in specs
3. **Non-Blocking**: Pruning failures must not block device reconciliation
4. **State Tracking**: Track which images/artifacts were previously referenced to enable incremental pruning
5. **Fleet Management**: Support opt-in/opt-out via file dropins for device or fleet-level control

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Device Agent                             │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         Device Reconciliation Loop                  │  │
│  │                                                      │  │
│  │  1. Sync device spec                                │  │
│  │  2. Apply changes (applications, OS, etc.)          │  │
│  │  3. Upgrade spec (mark as applied)                  │  │
│  │  4. → Prune unused images/artifacts ←              │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                  │
│                          ▼                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │            Pruning Manager                           │  │
│  │                                                      │  │
│  │  • Reads current/desired specs                      │  │
│  │  • Compares with previous references                │  │
│  │  • Identifies lost references                       │  │
│  │  • Removes eligible images/artifacts                │  │
│  │  • Records new references                           │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                  │
│         ┌────────────────┼────────────────┐                │
│         ▼                ▼                ▼                │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐            │
│  │  Spec    │    │ Podman   │    │  File    │            │
│  │ Manager  │    │ Client   │    │  I/O     │            │
│  └──────────┘    └──────────┘    └──────────┘            │
└─────────────────────────────────────────────────────────────┘
```

### Key Dependencies

- **Spec Manager**: Reads current and desired device specifications
- **Podman Client**: Lists and removes images/artifacts from local storage
- **OCITargetCache**: Extracts nested OCI targets from image-based applications
- **File I/O**: Reads/writes reference tracking file

## Design Decisions

### 1. "Lost References" Pruning Strategy

**Decision**: Only prune images/artifacts that were previously referenced but are no longer referenced.

**Rationale**:
- **Safety**: Never removes images that were never tracked, avoiding accidental deletion of manually pulled images
- **Incremental**: Only removes what was explicitly lost, not everything unreferenced
- **First Run Protection**: On first run (no previous references file), nothing is pruned

**Implementation**:
- Maintains a JSON file (`image-artifact-references.json`) recording all previously referenced images/artifacts
- Compares previous references with current/desired specs
- Only items in the "lost references" set are eligible for pruning

### 2. Separate Handling of Images and Artifacts

**Decision**: Treat container images and OCI artifacts as separate entities with distinct removal paths.

**Rationale**:
- **Different APIs**: Podman has separate commands for images vs artifacts
- **Different Use Cases**: Artifacts (SBOMs, signatures) may be referenced independently
- **Clearer Logic**: Separate lists make the code more maintainable

**Implementation**:
- Maintains separate lists for images and artifacts eligible for removal
- Images and artifacts are removed through separate operations
- Categorization happens during reference recording and eligibility determination

### 3. Nested Target Extraction

**Decision**: Extract and preserve nested OCI targets (images/artifacts referenced inside other images).

**Rationale**:
- **Compose Files**: A Compose file inside an image may reference other images/artifacts
- **Dependencies**: Artifacts (e.g., SBOMs) may be referenced within image manifests
- **Completeness**: Ensures all dependencies are preserved, not just top-level references

**Implementation**:
- Uses the OCI target cache to extract nested targets from image-based applications
- Extracts from both current and desired specs
- Best-effort: failures during extraction are logged but don't block pruning

### 4. OS Image Pruning

**Decision**: OS images can be pruned if they lose their references, just like application images.

**Rationale**:
- **Consistency**: OS images should follow the same pruning rules as application images
- **Lost References**: If an OS image is no longer referenced in specs, it should be removed
- **Disk Space**: Old OS images that are no longer needed should be cleaned up

**Implementation**:
- OS images are included in the image reference extraction along with application images
- OS images are treated the same as application images in eligibility determination
- OS images that lose their references are eligible for pruning

### 5. Non-Blocking Error Handling

**Decision**: Pruning errors are logged but never block device reconciliation.

**Rationale**:
- **Availability**: Device operations must continue even if pruning fails
- **Graceful Degradation**: Partial pruning failures shouldn't prevent successful reconciliation
- **Observability**: Errors are logged for debugging but don't fail the operation

**Implementation**:
- All pruning operations may encounter errors, but these are logged as warnings and processing continues
- Individual removal failures are logged but don't stop processing of other items
- Validation failures are logged but don't block reconciliation

### 6. Post-Upgrade Invocation

**Decision**: Pruning is invoked only after successful spec upgrade (after all managers have applied changes).

**Rationale**:
- **Consistency**: Ensures pruning sees the final state after all changes are applied
- **Safety**: Only prunes after confirming all managers successfully applied the spec
- **Correctness**: The references file is updated with the final state

**Implementation**:
- Pruning is invoked in the device reconciliation loop after the spec upgrade succeeds
- This ensures the spec files reflect the applied state before pruning runs

### 7. File Dropin Configuration

**Decision**: Pruning enabled flag can be configured via file dropins (similar to cert dropins).

**Rationale**:
- **Fleet Management**: Allows enabling/disabling pruning per device or fleet via ConfigProviderSpec
- **Flexibility**: Users can opt-in or opt-out without modifying main config
- **Consistency**: Follows the same pattern as certificate configuration

**Implementation**:
- Base file: `/etc/flightctl/pruning.yaml`
- Dropins: `/etc/flightctl/pruning.d/*.yaml` (applied in lexical order)
- Dropins override base file, later dropins override earlier ones
- Can be managed via `ConfigProviderSpec` in device spec

## Pruning Algorithm

### High-Level Flow

```
1. Check if pruning is enabled
   └─> If disabled, return early

2. Determine eligible images/artifacts
   ├─> Read previous references from file
   │   └─> If no file exists (first run), return empty list
   ├─> Extract current references from specs
   │   ├─> Application images (explicit + nested)
   │   └─> OS images (included with application images)
   ├─> Find lost references (previously referenced but not current)
   └─> Verify items exist locally (images vs artifacts)

3. Remove eligible items
   ├─> Remove eligible images
   └─> Remove eligible artifacts

4. Validate capability
   └─> Verify required images still exist after pruning

5. Record new references
   └─> Write current/desired references to file for next run
```

### Detailed Steps

#### Step 1: Eligibility Determination

The eligibility determination process:

1. **Read previous references**: Load the previously recorded references from the tracking file. If no file exists (first run), return an empty list.

2. **Get current references from specs**: Extract all image and artifact references from the current and desired device specifications. This includes application images, OS images, and nested targets.

3. **Find lost references**: Compare previous references with current references. Items that were previously referenced but are no longer in the current specs are considered "lost references" and eligible for pruning.

4. **Verify existence and categorize**: For each lost reference, verify it exists locally and categorize it as either an image or artifact. Only items that exist locally can be pruned.

#### Step 2: Reference Extraction

**Explicit References**:
- Application images (Image provider)
- Compose service images
- Quadlet container images
- Volume images (image-based volumes)

**Nested References**:
- Images/artifacts referenced inside image-based applications
- Extracted via the OCI target cache from already-pulled images
- Includes Compose files, manifests, etc.

**OS References**:
- Extracted from the OS image field in the device specification
- Included in the same reference extraction as application images
- Treated the same as application images for pruning eligibility

#### Step 3: Removal

**Images**:
- Removes images via Podman client operations
- Checks existence before removal
- Continues on individual failures

**Artifacts**:
- Removes artifacts via Podman client operations
- Checks existence before removal
- Continues on individual failures

#### Step 4: Validation

After removal, validates that:
- All current application images still exist
- All desired application images still exist
- All current OS images still exist
- All desired OS images still exist

Logs warnings for missing images but doesn't block reconciliation.

#### Step 5: Reference Recording

Records all current references to `image-artifact-references.json`:
- Timestamp of recording
- List of images (categorized)
- List of artifacts (categorized)
- Used for next pruning cycle

## Data Structures

### Image and Artifact References

The reference tracking file stores:
- **Timestamp**: When the references were recorded (RFC3339 format)
- **Images**: List of container image references
- **Artifacts**: List of OCI artifact references

**Location**: `{dataDir}/image-artifact-references.json`

**Purpose**: Tracks which images/artifacts were referenced in the previous reconciliation cycle.

### Eligible Items

The eligibility determination produces separate lists:
- **Images**: List of images eligible for removal
- **Artifacts**: List of artifacts eligible for removal

**Purpose**: Separates eligible items by type for distinct removal paths.

### Pruning Configuration

The pruning configuration contains:
- **Enabled**: Boolean flag controlling whether pruning is enabled (default: true)

**Purpose**: Controls whether pruning is enabled.

**Configuration Sources** (in order of precedence):
1. File dropins: `/etc/flightctl/pruning.d/*.yaml` (lexical order)
2. Base file: `/etc/flightctl/pruning.yaml`
3. Config file: `pruning.enabled` field
4. Default: `true`

## Integration Points

### Device Reconciliation Loop

**Integration Point**: Device reconciliation loop

**Invocation Flow**:
1. Sync and apply device specification changes
2. Upgrade the spec (mark as successfully applied)
3. If upgrade succeeds, invoke pruning
4. If pruning encounters errors, log warnings but continue reconciliation

**Key Points**:
- Only invoked after successful spec upgrade
- Errors are logged but don't block reconciliation
- Optional (pruning may be disabled via configuration)

### Spec Manager Integration

**Reads**:
- Current spec: Currently applied device specification
- Desired spec: Target device specification

**Uses**:
- Reads the current device specification to determine currently required images/artifacts
- Reads the desired device specification to determine target required images/artifacts

### Podman Client Integration

**Operations**:
- List all local container images
- List all local OCI artifacts
- Check if a specific image exists locally
- Check if a specific artifact exists locally
- Remove a container image from local storage
- Remove an OCI artifact from local storage

### OCITargetCache Integration

**Purpose**: Extract nested OCI targets from image-based applications.

**Usage**:
- Extracts nested targets from both current and desired specs
- Best-effort: failures are logged but don't block pruning
- Used to discover artifacts referenced inside images (e.g., Compose files)

## Configuration

### File Dropin Support

Pruning can be enabled/disabled via file dropins, allowing fleet or device-level control via ConfigProviderSpec in the device specification.

**Base File** (optional):
```yaml
# /etc/flightctl/pruning.yaml
enabled: true
```

**Dropin Files** (optional, applied in lexical order):
```yaml
# /etc/flightctl/pruning.d/01-enable.yaml
enabled: true
```

```yaml
# /etc/flightctl/pruning.d/99-disable.yaml
enabled: false  # Overrides earlier dropins
```

**Config File** (fallback):
```yaml
# /etc/flightctl/config.yaml
pruning:
  enabled: true
```

**Precedence** (highest to lowest):
1. Dropin files (lexical order, later overrides earlier)
2. Base file (`pruning.yaml`)
3. Config file (`pruning.enabled`)
4. Default (`true`)

### Example: Fleet-Level Configuration

```yaml
# Device spec with ConfigProviderSpec
spec:
  config:
    - name: pruning-config
      inline:
        - path: /etc/flightctl/pruning.d/01-fleet-enable.yaml
          content: |
            enabled: true
```

## Error Handling

### Error Categories

1. **Eligibility Determination Errors**:
   - Spec read failures → Logged, return empty list
   - Image listing failures → Logged, continue with partial results
   - Nested extraction failures → Logged, continue without nested targets

2. **Removal Errors**:
   - Individual image removal failures → Logged, continue with others
   - Individual artifact removal failures → Logged, continue with others
   - All removals fail → Return error (but caller logs and continues)

3. **Validation Errors**:
   - Missing required images → Logged as warning, don't block reconciliation
   - Validation function errors → Logged, don't block reconciliation

4. **Recording Errors**:
   - File write failures → Logged, don't block reconciliation
   - JSON marshaling failures → Logged, don't block reconciliation

### Error Recovery

- **Partial Failures**: Pruning continues with remaining items
- **Complete Failures**: Logged but reconciliation continues
- **State Consistency**: Reference file is only updated on successful recording

## Edge Cases

### First Run

**Scenario**: No previous references file exists.

**Behavior**: 
- Previous references file is not found
- Eligibility determination returns an empty list
- Nothing is pruned
- References are recorded for next run

### Missing References File

**Scenario**: References file was deleted or corrupted.

**Behavior**:
- Treated as first run
- Nothing is pruned
- New references file is created

### Spec Read Failures

**Scenario**: Cannot read current or desired spec.

**Behavior**:
- Error is logged
- Pruning continues with available specs
- If both fail, eligibility determination returns error (logged, no pruning)

### Image/Artifact Already Removed

**Scenario**: Item in lost references list doesn't exist locally.

**Behavior**:
- Existence check before removal
- Skipped if doesn't exist
- No error logged (may have been removed manually)

### Nested Extraction Failures

**Scenario**: Cannot extract nested targets from image.

**Behavior**:
- Error is logged
- Pruning continues without nested targets
- Explicit references are still preserved

## Performance Considerations

### Reference File Size

- Grows with number of referenced images/artifacts
- Typically small (KB range) even for many references
- JSON format for human readability and debugging

### Pruning Frequency

- Invoked once per successful reconciliation
- Only runs if spec was successfully upgraded
- Skips if no lost references found

### Extraction Overhead

- Nested target extraction may be expensive
- Uses caching via `OCITargetCache` to minimize redundant extractions
- Best-effort: failures don't block pruning