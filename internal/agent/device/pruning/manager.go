package pruning

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/flightctl/flightctl/api/v1beta1"
	"github.com/flightctl/flightctl/internal/agent/client"
	"github.com/flightctl/flightctl/internal/agent/config"
	"github.com/flightctl/flightctl/internal/agent/device/applications/provider"
	"github.com/flightctl/flightctl/internal/agent/device/errors"
	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	"github.com/flightctl/flightctl/internal/agent/device/spec"
	"github.com/flightctl/flightctl/internal/api/common"
	"github.com/flightctl/flightctl/pkg/log"
	"github.com/samber/lo"
)

var _ Manager = (*manager)(nil)

const (
	// ReferencesFileName is the name of the file that stores image and artifact references
	ReferencesFileName = "image-artifact-references.json"
)

// Manager provides the public API for managing image pruning operations.
type Manager interface {
	// Prune removes unused container images and OCI artifacts after successful spec reconciliation.
	// It preserves images required for current and desired operations.
	Prune(ctx context.Context) error
	// RecordReferences records all image and artifact references from current and desired specs to a file.
	// This ensures the references file exists and is kept up to date even when pruning is disabled,
	// so that when pruning is later enabled, it has accurate historical data to work with.
	RecordReferences(ctx context.Context) error
}

// PruningConfig holds configuration for pruning operations.
// This type is defined in internal/agent/config/config.go as config.Pruning
// and is aliased here for backward compatibility and clarity.
type PruningConfig = config.Pruning

// manager implements the Manager interface for image pruning operations.
type manager struct {
	podmanClient   *client.Podman
	specManager    spec.Manager
	readWriter     fileio.ReadWriter
	log            *log.PrefixLogger
	config         PruningConfig
	ociTargetCache *provider.OCITargetCache
	dataDir        string
}

// NewManager creates a new pruning manager instance.
//
// Dependencies:
//   - podmanClient: Podman client for image/artifact operations
//   - specManager: Spec manager for reading current and desired specs
//   - readWriter: File I/O interface for any file operations
//   - log: Logger for structured logging
//   - config: Pruning configuration (enabled flag, etc.)
//   - dataDir: Directory where data files are stored
func NewManager(
	podmanClient *client.Podman,
	specManager spec.Manager,
	readWriter fileio.ReadWriter,
	log *log.PrefixLogger,
	config PruningConfig,
	dataDir string,
) Manager {
	return &manager{
		podmanClient: podmanClient,
		specManager:  specManager,
		readWriter:   readWriter,
		log:          log,
		config:       config,
		dataDir:      dataDir,
	}
}

// Prune removes unused container images and OCI artifacts after successful spec reconciliation.
// It preserves images required for current and desired operations.
func (m *manager) Prune(ctx context.Context) error {
	if !m.config.Enabled {
		m.log.Debug("Pruning is disabled, skipping")
		return nil
	}

	// Determine eligible images and artifacts for pruning
	// This function handles all validation: it only considers images/artifacts that exist locally,
	// and builds a preserve set from required images in specs. Missing required images
	// cannot be pruned anyway, so they don't need explicit protection.
	eligible, err := m.determineEligibleImages(ctx)
	if err != nil {
		m.log.Warnf("Failed to determine eligible images: %v", err)
		// Don't block reconciliation on pruning errors
		return nil
	}

	totalEligible := len(eligible.Images) + len(eligible.Artifacts)
	if totalEligible == 0 {
		m.log.Debug("No images or artifacts eligible for pruning")
		// Still record current references even if nothing to prune
		// This ensures the file is created on first run and updated on subsequent runs
		if err := m.recordImageArtifactReferences(ctx); err != nil {
			m.log.Warnf("Failed to record image/artifact references: %v", err)
			// Don't block reconciliation on recording errors
		}
		return nil
	}

	m.log.Infof("Starting pruning of %d eligible images and %d eligible artifacts", len(eligible.Images), len(eligible.Artifacts))

	// Remove eligible images and artifacts separately, tracking which ones were successfully removed
	removedImages, removedImageRefs, err := m.removeEligibleImages(ctx, eligible.Images)
	if err != nil {
		m.log.Warnf("Error during image removal: %v", err)
		// Continue with artifact removal even if image removal failed
	}

	removedArtifacts, removedArtifactRefs, err := m.removeEligibleArtifacts(ctx, eligible.Artifacts)
	if err != nil {
		m.log.Warnf("Error during artifact removal: %v", err)
		// Continue with validation even if some removals failed
	}

	m.log.Infof("Pruning complete: removed %d of %d eligible images, %d of %d eligible artifacts", removedImages, len(eligible.Images), removedArtifacts, len(eligible.Artifacts))

	// Remove successfully pruned items from the references file
	// This ensures the accumulated file only contains items that still exist or haven't been pruned yet
	if len(removedImageRefs) > 0 || len(removedArtifactRefs) > 0 {
		if err := m.removePrunedReferencesFromFile(removedImageRefs, removedArtifactRefs); err != nil {
			m.log.Warnf("Failed to remove pruned references from file: %v", err)
			// Don't block reconciliation on file update errors
		}
	}

	// Validate capability after pruning
	if err := m.validateCapability(ctx); err != nil {
		m.log.Warnf("Capability validation failed after pruning: %v", err)
		// Log warning but don't block reconciliation
	}

	// Record all image and artifact references to file
	// This accumulates new references from current specs for the next pruning cycle
	if err := m.recordImageArtifactReferences(ctx); err != nil {
		m.log.Warnf("Failed to record image/artifact references: %v", err)
		// Don't block reconciliation on recording errors
	}

	return nil
}

// RecordReferences records all image and artifact references from current and desired specs to a file.
// This ensures the references file exists and is kept up to date even when pruning is disabled,
// so that when pruning is later enabled, it has accurate historical data to work with.
// This is called on every successful sync, regardless of pruning enabled status.
func (m *manager) RecordReferences(ctx context.Context) error {
	if err := m.recordImageArtifactReferences(ctx); err != nil {
		m.log.Warnf("Failed to record image/artifact references: %v", err)
		return err
	}

	return nil
}

// getImageReferencesFromSpecs extracts image references from current and desired device specs.
// It includes both explicit references and nested targets extracted from image-based applications.
// It uses the spec manager's Read() method to read specs and returns a combined unique list.
func (m *manager) getImageReferencesFromSpecs(ctx context.Context) ([]string, error) {
	var images []string

	// Process both Current and Desired specs
	specs := []struct {
		specType spec.Type
		name     string
		required bool // Current is required, Desired is optional
	}{
		{spec.Current, "current", true},
		{spec.Desired, "desired", false},
	}

	for _, s := range specs {
		device, err := m.specManager.Read(s.specType)
		if err != nil {
			if s.required {
				return nil, fmt.Errorf("reading %s spec: %w", s.name, err)
			}
			// Desired spec may not exist - this is acceptable
			m.log.Debugf("%s spec not available: %v", s.name, err)
			continue
		}

		if device == nil {
			continue
		}

		// Extract explicit image references
		explicitImages, err := m.extractImageReferences(ctx, device)
		if err != nil {
			return nil, fmt.Errorf("extracting images from %s spec: %w", s.name, err)
		}
		images = append(images, explicitImages...)

		// Extract nested targets from image-based applications
		nestedImages, err := m.extractNestedTargetsFromSpec(ctx, device)
		if err != nil {
			// Log warning but don't fail - nested extraction is best-effort
			m.log.Warnf("Failed to extract nested targets from %s spec: %v", s.name, err)
		} else {
			images = append(images, nestedImages...)
		}
	}

	return lo.Uniq(images), nil
}

// ImageArtifactReferences holds all image and artifact references from specs.
type ImageArtifactReferences struct {
	Timestamp string   `json:"timestamp"`
	Images    []string `json:"images"`
	Artifacts []string `json:"artifacts"`
}

// categorizeReference checks if a reference exists as an image or artifact and returns the category.
// Returns "image", "artifact", or "unknown" if it doesn't exist locally.
func (m *manager) categorizeReference(ctx context.Context, ref string) string {
	if m.podmanClient.ImageExists(ctx, ref) {
		return "image"
	}
	if m.podmanClient.ArtifactExists(ctx, ref) {
		return "artifact"
	}
	return "unknown"
}

// readPreviousReferences reads the previous image/artifact references from the file.
// Returns nil if the file doesn't exist (first run) or if there's an error reading it.
func (m *manager) readPreviousReferences() *ImageArtifactReferences {
	// Use filepath.Join to create the full path, then readWriter will handle it correctly
	filePath := filepath.Join(m.dataDir, ReferencesFileName)
	data, err := m.readWriter.ReadFile(filePath)
	if err != nil {
		// File doesn't exist or can't be read - this is expected on first run
		m.log.Debugf("Previous references file not found or unreadable: %v", err)
		return nil
	}

	var refs ImageArtifactReferences
	if err := json.Unmarshal(data, &refs); err != nil {
		m.log.Warnf("Failed to unmarshal previous references file: %v", err)
		return nil
	}

	return &refs
}

// recordImageArtifactReferences records all image and artifact references from current and desired specs to a file.
// It accumulates references: reads existing file (if any), adds new references from current specs, and writes back.
// References are only removed when they are successfully pruned (see removePrunedReferencesFromFile).
func (m *manager) recordImageArtifactReferences(ctx context.Context) error {
	// Read existing references file (if it exists) to accumulate with new references
	existingRefs := m.readPreviousReferences()
	
	// Build sets from existing references for efficient lookup
	existingImagesSet := make(map[string]struct{})
	existingArtifactsSet := make(map[string]struct{})
	if existingRefs != nil {
		for _, img := range existingRefs.Images {
			existingImagesSet[img] = struct{}{}
		}
		for _, artifact := range existingRefs.Artifacts {
			existingArtifactsSet[artifact] = struct{}{}
		}
	}

	// Get all image references from specs using the shared function
	allRefs, err := m.getImageReferencesFromSpecs(ctx)
	if err != nil {
		// Log warning but continue - we still want to record what we can
		// This is more lenient than getImageReferencesFromSpecs which requires Current spec
		m.log.Warnf("Failed to get image references from specs for recording: %v", err)
		allRefs = []string{} // Continue with empty list
	}

	// Categorize and accumulate new references
	for _, ref := range allRefs {
		category := m.categorizeReference(ctx, ref)
		switch category {
		case "image":
			if _, exists := existingImagesSet[ref]; !exists {
				existingImagesSet[ref] = struct{}{}
			}
		case "artifact":
			if _, exists := existingArtifactsSet[ref]; !exists {
				existingArtifactsSet[ref] = struct{}{}
			}
		default:
			// For unknown references, add to both lists since we can't determine
			// They might be pulled later or might not exist locally yet
			if _, exists := existingImagesSet[ref]; !exists {
				existingImagesSet[ref] = struct{}{}
			}
			if _, exists := existingArtifactsSet[ref]; !exists {
				existingArtifactsSet[ref] = struct{}{}
			}
		}
	}

	// Build final accumulated references
	refs := ImageArtifactReferences{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Images:    lo.Keys(existingImagesSet),
		Artifacts: lo.Keys(existingArtifactsSet),
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(refs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling image/artifact references: %w", err)
	}

	// Write to file
	filePath := filepath.Join(m.dataDir, ReferencesFileName)
	if err := m.readWriter.WriteFile(filePath, jsonData, fileio.DefaultFilePermissions); err != nil {
		return fmt.Errorf("writing image/artifact references to file: %w", err)
	}

	m.log.Debugf("Recorded image/artifact references to %s (accumulated: %d images, %d artifacts)", filePath, len(refs.Images), len(refs.Artifacts))
	return nil
}

// removePrunedReferencesFromFile removes successfully pruned images and artifacts from the references file.
// This ensures the accumulated file only contains items that still exist or haven't been pruned yet.
func (m *manager) removePrunedReferencesFromFile(removedImages []string, removedArtifacts []string) error {
	if len(removedImages) == 0 && len(removedArtifacts) == 0 {
		return nil // Nothing to remove
	}

	// Read existing references file
	existingRefs := m.readPreviousReferences()
	if existingRefs == nil {
		// File doesn't exist - nothing to remove
		return nil
	}

	// Build sets of removed items for efficient lookup
	removedImagesSet := make(map[string]struct{})
	for _, img := range removedImages {
		removedImagesSet[img] = struct{}{}
	}
	removedArtifactsSet := make(map[string]struct{})
	for _, artifact := range removedArtifacts {
		removedArtifactsSet[artifact] = struct{}{}
	}

	// Filter out removed items
	var filteredImages []string
	for _, img := range existingRefs.Images {
		if _, removed := removedImagesSet[img]; !removed {
			filteredImages = append(filteredImages, img)
		}
	}

	var filteredArtifacts []string
	for _, artifact := range existingRefs.Artifacts {
		if _, removed := removedArtifactsSet[artifact]; !removed {
			filteredArtifacts = append(filteredArtifacts, artifact)
		}
	}

	// Build updated references
	refs := ImageArtifactReferences{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Images:    filteredImages,
		Artifacts: filteredArtifacts,
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(refs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling updated image/artifact references: %w", err)
	}

	// Write to file
	filePath := filepath.Join(m.dataDir, ReferencesFileName)
	if err := m.readWriter.WriteFile(filePath, jsonData, fileio.DefaultFilePermissions); err != nil {
		return fmt.Errorf("writing updated image/artifact references to file: %w", err)
	}

	m.log.Debugf("Removed %d images and %d artifacts from references file (remaining: %d images, %d artifacts)", len(removedImages), len(removedArtifacts), len(filteredImages), len(filteredArtifacts))
	return nil
}

// EligibleItems holds separate lists of eligible images and artifacts for pruning.
type EligibleItems struct {
	Images    []string
	Artifacts []string
}

// determineEligibleImages determines which images and artifacts are eligible for pruning.
// It only prunes items that were previously referenced (according to the references file)
// but are no longer referenced in the current specs.
// OS images are included and can be pruned if they lose their references.
// Returns separate lists for images and artifacts.
func (m *manager) determineEligibleImages(ctx context.Context) (*EligibleItems, error) {
	m.log.Debug("Determining eligible images and artifacts for pruning")

	// Read previous references from file
	previousRefs := m.readPreviousReferences()
	if previousRefs == nil {
		// No previous references file - this is the first run, so nothing to prune
		m.log.Debug("No previous references file found - skipping pruning on first run")
		return &EligibleItems{
			Images:    []string{},
			Artifacts: []string{},
		}, nil
	}

	// Get all images and artifacts from Podman first (needed for categorization)
	allImages, err := m.podmanClient.ListImages(ctx)
	if err != nil {
		m.log.Warnf("Failed to list container images: %v", err)
		// Continue with partial results
		allImages = []string{}
	}

	allArtifacts, err := m.podmanClient.ListArtifacts(ctx)
	if err != nil {
		m.log.Warnf("Failed to list OCI artifacts: %v", err)
		// Continue with partial results
		allArtifacts = []string{}
	}

	allImages = lo.Uniq(allImages)
	allArtifacts = lo.Uniq(allArtifacts)

	// Get current required references from specs (includes application images, OS images, and artifacts)
	currentRequiredRefs, err := m.getImageReferencesFromSpecs(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting image references from specs: %w", err)
	}

	currentRequiredRefs = lo.Uniq(currentRequiredRefs)

	currentImages := lo.Intersect(allImages, currentRequiredRefs)
	currentArtifacts := lo.Intersect(allArtifacts, currentRequiredRefs)
	remaining := lo.Without(currentRequiredRefs, lo.Union(currentImages, currentArtifacts)...)
	currentImages = append(currentImages, remaining...)
	currentArtifacts = append(currentArtifacts, remaining...)

	eligibleImages := lo.Without(lo.Uniq(previousRefs.Images), currentImages...)
	eligibleArtifacts := lo.Without(lo.Uniq(previousRefs.Artifacts), currentArtifacts...)

	if len(eligibleImages) == 0 && len(eligibleArtifacts) == 0 {
		m.log.Debug("No previously referenced items have lost their references - nothing to prune")
	} else {
		m.log.Debugf("Found %d eligible images and %d eligible artifacts for pruning (previously referenced but no longer)", len(eligibleImages), len(eligibleArtifacts))
	}
	return &EligibleItems{
		Images:    eligibleImages,
		Artifacts: eligibleArtifacts,
	}, nil
}

// extractImageReferences extracts all container image and OCI artifact references from a device spec.
// It includes both application images and OS images.
// It returns a unique list of image/artifact identifiers.
func (m *manager) extractImageReferences(ctx context.Context, device *v1beta1.Device) ([]string, error) {
	if device == nil || device.Spec == nil {
		return []string{}, nil
	}

	var images []string

	// Extract images from applications
	if device.Spec.Applications != nil {
		for _, appSpec := range lo.FromPtr(device.Spec.Applications) {
			appImages, err := m.extractImagesFromApplication(ctx, &appSpec)
			if err != nil {
				return nil, fmt.Errorf("extracting images from application %s: %w", lo.FromPtr(appSpec.Name), err)
			}
			images = append(images, appImages...)
		}
	}

	// Extract OS image (if present)
	if device.Spec.Os != nil && device.Spec.Os.Image != "" {
		images = append(images, device.Spec.Os.Image)
	}

	return lo.Uniq(images), nil
}

// extractImagesFromApplication extracts image references from a single application spec.
func (m *manager) extractImagesFromApplication(ctx context.Context, appSpec *v1beta1.ApplicationProviderSpec) ([]string, error) {
	var images []string

	providerType, err := appSpec.Type()
	if err != nil {
		return nil, fmt.Errorf("determining provider type: %w", err)
	}

	switch providerType {
	case v1beta1.ImageApplicationProviderType:
		imageSpec, err := appSpec.AsImageApplicationProviderSpec()
		if err != nil {
			return nil, fmt.Errorf("getting image provider spec: %w", err)
		}
		images = append(images, imageSpec.Image)

		// Extract volume images
		if imageSpec.Volumes != nil {
			volImages, err := m.extractVolumeImages(*imageSpec.Volumes)
			if err != nil {
				return nil, fmt.Errorf("extracting volume images: %w", err)
			}
			images = append(images, volImages...)
		}

	case v1beta1.InlineApplicationProviderType:
		inlineSpec, err := appSpec.AsInlineApplicationProviderSpec()
		if err != nil {
			return nil, fmt.Errorf("getting inline provider spec: %w", err)
		}

		// Extract images from inline content based on app type
		switch appSpec.AppType {
		case v1beta1.AppTypeCompose:
			composeImages, err := m.extractComposeImages(inlineSpec.Inline)
			if err != nil {
				return nil, fmt.Errorf("extracting compose images: %w", err)
			}
			images = append(images, composeImages...)

		case v1beta1.AppTypeQuadlet:
			quadletImages, err := m.extractQuadletImages(inlineSpec.Inline)
			if err != nil {
				return nil, fmt.Errorf("extracting quadlet images: %w", err)
			}
			images = append(images, quadletImages...)

		case v1beta1.AppTypeContainer:
			// Container type cannot use inline provider (validated by API)
			return nil, fmt.Errorf("%w: container applications cannot use inline provider", errors.ErrUnsupportedAppType)

		default:
			return nil, fmt.Errorf("%w: %s", errors.ErrUnsupportedAppType, appSpec.AppType)
		}

		// Extract volume images
		if inlineSpec.Volumes != nil {
			volImages, err := m.extractVolumeImages(*inlineSpec.Volumes)
			if err != nil {
				return nil, fmt.Errorf("extracting volume images: %w", err)
			}
			images = append(images, volImages...)
		}

	default:
		return nil, fmt.Errorf("unsupported application provider type: %s", providerType)
	}

	return images, nil
}

// extractNestedTargetsFromSpec extracts nested OCI targets (images and artifacts) from image-based applications.
// This ensures that artifacts referenced inside images (e.g., in Compose files) are preserved during pruning.
// Returns a list of image/artifact references found in nested targets.
// It uses the OCITargetCache which handles extraction and caching logic.
func (m *manager) extractNestedTargetsFromSpec(ctx context.Context, device *v1beta1.Device) ([]string, error) {
	if device == nil || device.Spec == nil {
		return []string{}, nil
	}

	// Use the cache's method to collect nested target references
	// Pass nil for pullSecret since we're only extracting from already-pulled images
	// Errors during extraction are logged but don't block collection (best-effort for pruning)
	return m.ociTargetCache.CollectNestedTargetReferencesFromSpec(
		ctx,
		m.log,
		m.podmanClient,
		m.readWriter,
		device.Spec,
		nil, // pullSecret not needed for extraction from local images
	)
}

// extractComposeImages extracts image references from Compose inline content.
func (m *manager) extractComposeImages(contents []v1beta1.ApplicationContent) ([]string, error) {
	spec, err := client.ParseComposeFromSpec(contents)
	if err != nil {
		return nil, fmt.Errorf("parsing compose spec: %w", err)
	}

	var images []string
	for _, svc := range spec.Services {
		if svc.Image != "" {
			images = append(images, svc.Image)
		}
	}

	return images, nil
}

// extractQuadletImages extracts image references from Quadlet inline content.
func (m *manager) extractQuadletImages(contents []v1beta1.ApplicationContent) ([]string, error) {
	quadlets, err := client.ParseQuadletReferencesFromSpec(contents)
	if err != nil {
		return nil, fmt.Errorf("parsing quadlet spec: %w", err)
	}

	var images []string
	for _, quad := range quadlets {
		// Extract images from service/container quadlets
		if quad.Image != nil {
			images = append(images, *quad.Image)
		}
		// Extract images from volume quadlets
		if quad.Type == common.QuadletTypeVolume && quad.Image != nil {
			images = append(images, *quad.Image)
		}
	}

	return images, nil
}

// extractVolumeImages extracts image references from application volumes.
func (m *manager) extractVolumeImages(volumes []v1beta1.ApplicationVolume) ([]string, error) {
	var images []string

	for _, vol := range volumes {
		volType, err := vol.Type()
		if err != nil {
			return nil, fmt.Errorf("determining volume type: %w", err)
		}

		switch volType {
		case v1beta1.ImageApplicationVolumeProviderType:
			provider, err := vol.AsImageVolumeProviderSpec()
			if err != nil {
				return nil, fmt.Errorf("getting image volume provider spec: %w", err)
			}
			images = append(images, provider.Image.Reference)

		case v1beta1.ImageMountApplicationVolumeProviderType:
			provider, err := vol.AsImageMountVolumeProviderSpec()
			if err != nil {
				return nil, fmt.Errorf("getting image mount volume provider spec: %w", err)
			}
			images = append(images, provider.Image.Reference)

		case v1beta1.MountApplicationVolumeProviderType:
			// Mount volumes don't have images
			continue

		default:
			return nil, fmt.Errorf("%w: %s", errors.ErrUnsupportedVolumeType, volType)
		}
	}

	return images, nil
}

// validateCapability verifies that capability is maintained after pruning operations.
// It checks that current and desired application images and OS images still exist.
func (m *manager) validateCapability(ctx context.Context) error {
	m.log.Debug("Validating capability after pruning")

	// Read current spec
	currentDevice, err := m.specManager.Read(spec.Current)
	if err != nil {
		return fmt.Errorf("reading current spec for validation: %w", err)
	}

	// Read desired spec (may not exist)
	desiredDevice, err := m.specManager.Read(spec.Desired)
	if err != nil {
		m.log.Debugf("Desired spec not available for validation: %v", err)
		// Desired spec may not exist - this is acceptable
		desiredDevice = nil
	}

	var missingImages []string

	// Validate current application images
	if currentDevice != nil && currentDevice.Spec != nil {
		currentImages, err := m.extractImageReferences(ctx, currentDevice)
		if err != nil {
			return fmt.Errorf("extracting current images for validation: %w", err)
		}
		for _, img := range currentImages {
			exists := m.podmanClient.ImageExists(ctx, img)
			if !exists {
				exists = m.podmanClient.ArtifactExists(ctx, img)
			}
			if !exists {
				missingImages = append(missingImages, img)
				m.log.Warnf("Current application image missing after pruning: %s", img)
			}
		}

		// Validate current OS image
		if currentDevice.Spec.Os != nil && currentDevice.Spec.Os.Image != "" {
			osImage := currentDevice.Spec.Os.Image
			exists := m.podmanClient.ImageExists(ctx, osImage)
			if !exists {
				missingImages = append(missingImages, osImage)
				m.log.Warnf("Current OS image missing after pruning: %s", osImage)
			}
		}
	}

	// Validate desired application images
	if desiredDevice != nil && desiredDevice.Spec != nil {
		desiredImages, err := m.extractImageReferences(ctx, desiredDevice)
		if err != nil {
			return fmt.Errorf("extracting desired images for validation: %w", err)
		}
		for _, img := range desiredImages {
			exists := m.podmanClient.ImageExists(ctx, img)
			if !exists {
				exists = m.podmanClient.ArtifactExists(ctx, img)
			}
			if !exists {
				missingImages = append(missingImages, img)
				m.log.Warnf("Desired application image missing after pruning: %s", img)
			}
		}

		// Validate desired OS image
		if desiredDevice.Spec.Os != nil && desiredDevice.Spec.Os.Image != "" {
			osImage := desiredDevice.Spec.Os.Image
			exists := m.podmanClient.ImageExists(ctx, osImage)
			if !exists {
				missingImages = append(missingImages, osImage)
				m.log.Warnf("Desired OS image missing after pruning: %s", osImage)
			}
		}
	}

	if len(missingImages) > 0 {
		return fmt.Errorf("capability compromised - missing images: %v", missingImages)
	}

	m.log.Debug("Capability validated successfully")
	return nil
}

// removeEligibleImages removes the list of eligible images from Podman storage.
// It returns the count of successfully removed images, the list of successfully removed image references, and any error encountered.
// Errors during individual removals are logged but don't stop the process.
func (m *manager) removeEligibleImages(ctx context.Context, eligibleImages []string) (int, []string, error) {
	var removedCount int
	var removedRefs []string
	var removalErrors []error

	for _, imageRef := range eligibleImages {
		// Check if image exists before attempting removal
		imageExists := m.podmanClient.ImageExists(ctx, imageRef)
		if !imageExists {
			// Image doesn't exist - skip it (may have been removed already or never existed)
			continue
		}

		err := m.podmanClient.RemoveImage(ctx, imageRef)
		if err == nil {
			removedCount++
			removedRefs = append(removedRefs, imageRef)
			m.log.Debugf("Removed image: %s", imageRef)
		} else {
			m.log.Warnf("Failed to remove image %s: %v", imageRef, err)
			removalErrors = append(removalErrors, fmt.Errorf("failed to remove image %s: %w", imageRef, err))
		}
	}

	// Return error only if all removals failed
	if len(removalErrors) == len(eligibleImages) && len(eligibleImages) > 0 {
		return removedCount, removedRefs, fmt.Errorf("all image removals failed: %d errors", len(removalErrors))
	}

	// Log summary if there were any failures
	if len(removalErrors) > 0 {
		m.log.Warnf("Image pruning completed with %d failures out of %d attempts", len(removalErrors), len(eligibleImages))
	}

	return removedCount, removedRefs, nil
}

// removeEligibleArtifacts removes the list of eligible artifacts from Podman storage.
// It returns the count of successfully removed artifacts, the list of successfully removed artifact references, and any error encountered.
// Errors during individual removals are logged but don't stop the process.
func (m *manager) removeEligibleArtifacts(ctx context.Context, eligibleArtifacts []string) (int, []string, error) {
	var removedCount int
	var removedRefs []string
	var removalErrors []error

	for _, artifactRef := range eligibleArtifacts {
		// Check if artifact exists before attempting removal
		artifactExists := m.podmanClient.ArtifactExists(ctx, artifactRef)
		if !artifactExists {
			// Artifact doesn't exist - skip it (may have been removed already or never existed)
			continue
		}

		err := m.podmanClient.RemoveArtifact(ctx, artifactRef)
		if err == nil {
			removedCount++
			removedRefs = append(removedRefs, artifactRef)
			m.log.Debugf("Removed artifact: %s", artifactRef)
		} else {
			m.log.Warnf("Failed to remove artifact %s: %v", artifactRef, err)
			removalErrors = append(removalErrors, fmt.Errorf("failed to remove artifact %s: %w", artifactRef, err))
		}
	}

	// Return error only if all removals failed
	if len(removalErrors) == len(eligibleArtifacts) && len(eligibleArtifacts) > 0 {
		return removedCount, removedRefs, fmt.Errorf("all artifact removals failed: %d errors", len(removalErrors))
	}

	// Log summary if there were any failures
	if len(removalErrors) > 0 {
		m.log.Warnf("Artifact pruning completed with %d failures out of %d attempts", len(removalErrors), len(eligibleArtifacts))
	}

	return removedCount, removedRefs, nil
}
