package pruning

import (
	"context"
	"fmt"

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

// Manager provides the public API for managing image pruning operations.
type Manager interface {
	// Prune removes unused container images and OCI artifacts after successful spec reconciliation.
	// It preserves images required for current and desired operations.
	Prune(ctx context.Context) error
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
}

// NewManager creates a new pruning manager instance.
//
// Dependencies:
//   - podmanClient: Podman client for image/artifact operations
//   - specManager: Spec manager for reading current and rollback specs
//   - readWriter: File I/O interface for any file operations
//   - log: Logger for structured logging
//   - config: Pruning configuration (enabled flag, etc.)
func NewManager(
	podmanClient *client.Podman,
	specManager spec.Manager,
	readWriter fileio.ReadWriter,
	log *log.PrefixLogger,
	config PruningConfig,
) Manager {
	return &manager{
		podmanClient: podmanClient,
		specManager:  specManager,
		readWriter:   readWriter,
		log:          log,
		config:       config,
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
		return nil
	}

	m.log.Infof("Starting pruning of %d eligible images and %d eligible artifacts", len(eligible.Images), len(eligible.Artifacts))

	// Remove eligible images and artifacts separately
	removedImages, err := m.removeEligibleImages(ctx, eligible.Images)
	if err != nil {
		m.log.Warnf("Error during image removal: %v", err)
		// Continue with artifact removal even if image removal failed
	}

	removedArtifacts, err := m.removeEligibleArtifacts(ctx, eligible.Artifacts)
	if err != nil {
		m.log.Warnf("Error during artifact removal: %v", err)
		// Continue with validation even if some removals failed
	}

	m.log.Infof("Pruning complete: removed %d of %d eligible images, %d of %d eligible artifacts", removedImages, len(eligible.Images), removedArtifacts, len(eligible.Artifacts))

	// Validate capability after pruning
	if err := m.validateCapability(ctx); err != nil {
		m.log.Warnf("Capability validation failed after pruning: %v", err)
		// Log warning but don't block reconciliation
	}

	return nil
}

// getImageReferencesFromSpecs extracts image references from current and desired device specs.
// It includes both explicit references and nested targets extracted from image-based applications.
// It uses the spec manager's Read() method to read specs and returns a combined unique list.
func (m *manager) getImageReferencesFromSpecs(ctx context.Context) ([]string, error) {
	imagesSeen := make(map[string]struct{})
	var allImages []string

	// Read current spec
	currentDevice, err := m.specManager.Read(spec.Current)
	if err != nil {
		return nil, fmt.Errorf("reading current spec: %w", err)
	}
	if currentDevice != nil {
		currentImages, err := m.extractImageReferences(ctx, currentDevice)
		if err != nil {
			return nil, fmt.Errorf("extracting images from current spec: %w", err)
		}
		for _, img := range currentImages {
			if _, seen := imagesSeen[img]; !seen {
				imagesSeen[img] = struct{}{}
				allImages = append(allImages, img)
			}
		}

		// Extract nested targets from image-based applications
		nestedImages, err := m.extractNestedTargetsFromSpec(ctx, currentDevice)
		if err != nil {
			// Log warning but don't fail - nested extraction is best-effort
			m.log.Warnf("Failed to extract nested targets from current spec: %v", err)
		} else {
			for _, img := range nestedImages {
				if _, seen := imagesSeen[img]; !seen {
					imagesSeen[img] = struct{}{}
					allImages = append(allImages, img)
				}
			}
		}
	}

	// Read desired spec (may not exist)
	desiredDevice, err := m.specManager.Read(spec.Desired)
	if err != nil {
		// Desired spec may not exist - this is acceptable
		m.log.Debugf("Desired spec not available: %v", err)
	} else if desiredDevice != nil {
		desiredImages, err := m.extractImageReferences(ctx, desiredDevice)
		if err != nil {
			return nil, fmt.Errorf("extracting images from desired spec: %w", err)
		}
		for _, img := range desiredImages {
			if _, seen := imagesSeen[img]; !seen {
				imagesSeen[img] = struct{}{}
				allImages = append(allImages, img)
			}
		}

		// Extract nested targets from image-based applications
		nestedImages, err := m.extractNestedTargetsFromSpec(ctx, desiredDevice)
		if err != nil {
			// Log warning but don't fail - nested extraction is best-effort
			m.log.Warnf("Failed to extract nested targets from desired spec: %v", err)
		} else {
			for _, img := range nestedImages {
				if _, seen := imagesSeen[img]; !seen {
					imagesSeen[img] = struct{}{}
					allImages = append(allImages, img)
				}
			}
		}
	}

	return allImages, nil
}

// getOSImageReferences extracts OS image references from current and desired device specs.
// OS images are managed by bootc and should never be pruned.
func (m *manager) getOSImageReferences(ctx context.Context) ([]string, error) {
	var osImages []string
	osImagesSeen := make(map[string]struct{})

	// Read current spec
	currentDevice, err := m.specManager.Read(spec.Current)
	if err != nil {
		return nil, fmt.Errorf("reading current spec: %w", err)
	}
	if currentDevice != nil && currentDevice.Spec != nil && currentDevice.Spec.Os != nil && currentDevice.Spec.Os.Image != "" {
		osImage := currentDevice.Spec.Os.Image
		if _, seen := osImagesSeen[osImage]; !seen {
			osImagesSeen[osImage] = struct{}{}
			osImages = append(osImages, osImage)
		}
	}

	// Read desired spec (may not exist)
	desiredDevice, err := m.specManager.Read(spec.Desired)
	if err != nil {
		// Desired spec may not exist - this is acceptable
		m.log.Debugf("Desired spec not available for OS images: %v", err)
	} else if desiredDevice != nil && desiredDevice.Spec != nil && desiredDevice.Spec.Os != nil && desiredDevice.Spec.Os.Image != "" {
		osImage := desiredDevice.Spec.Os.Image
		if _, seen := osImagesSeen[osImage]; !seen {
			osImagesSeen[osImage] = struct{}{}
			osImages = append(osImages, osImage)
		}
	}

	return osImages, nil
}

// EligibleItems holds separate lists of eligible images and artifacts for pruning.
type EligibleItems struct {
	Images    []string
	Artifacts []string
}

// determineEligibleImages determines which images and artifacts are eligible for pruning.
// It queries Podman for all images/artifacts separately and compares against required images from specs.
// OS images are excluded as they are managed by bootc.
// Returns separate lists for images and artifacts.
func (m *manager) determineEligibleImages(ctx context.Context) (*EligibleItems, error) {
	m.log.Debug("Determining eligible images and artifacts for pruning")

	// Get all images from Podman
	allImages, err := m.podmanClient.ListImages(ctx)
	if err != nil {
		m.log.Warnf("Failed to list container images: %v", err)
		// Continue with partial results
		allImages = []string{}
	}

	// Get all artifacts from Podman
	allArtifacts, err := m.podmanClient.ListArtifacts(ctx)
	if err != nil {
		m.log.Warnf("Failed to list OCI artifacts: %v", err)
		// Continue with partial results
		allArtifacts = []string{}
	}

	// Get required images from specs (application images)
	requiredImages, err := m.getImageReferencesFromSpecs(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting image references from specs: %w", err)
	}

	// Get OS images from specs (bootc manages these)
	osImages, err := m.getOSImageReferences(ctx)
	if err != nil {
		m.log.Warnf("Failed to get OS image references: %v", err)
		// Continue with partial results
		osImages = []string{}
	}

	// Build set of images/artifacts that must be preserved
	preserveSet := make(map[string]struct{})
	for _, img := range requiredImages {
		preserveSet[img] = struct{}{}
	}
	for _, img := range osImages {
		preserveSet[img] = struct{}{}
	}

	// Find eligible images (all images minus preserved images)
	var eligibleImages []string
	for _, img := range allImages {
		if _, preserved := preserveSet[img]; !preserved {
			eligibleImages = append(eligibleImages, img)
		}
	}

	// Find eligible artifacts (all artifacts minus preserved artifacts)
	var eligibleArtifacts []string
	for _, artifact := range allArtifacts {
		if _, preserved := preserveSet[artifact]; !preserved {
			eligibleArtifacts = append(eligibleArtifacts, artifact)
		}
	}

	m.log.Debugf("Found %d eligible images and %d eligible artifacts for pruning", len(eligibleImages), len(eligibleArtifacts))
	return &EligibleItems{
		Images:    eligibleImages,
		Artifacts: eligibleArtifacts,
	}, nil
}

// extractImageReferences extracts all container image and OCI artifact references from a device spec.
// It returns a unique list of image/artifact identifiers.
func (m *manager) extractImageReferences(ctx context.Context, device *v1beta1.Device) ([]string, error) {
	if device == nil || device.Spec == nil {
		return []string{}, nil
	}

	imagesSeen := make(map[string]struct{})
	var images []string

	// Extract images from applications
	if device.Spec.Applications != nil {
		for _, appSpec := range lo.FromPtr(device.Spec.Applications) {
			appImages, err := m.extractImagesFromApplication(ctx, &appSpec)
			if err != nil {
				return nil, fmt.Errorf("extracting images from application %s: %w", lo.FromPtr(appSpec.Name), err)
			}
			for _, img := range appImages {
				if _, seen := imagesSeen[img]; !seen {
					imagesSeen[img] = struct{}{}
					images = append(images, img)
				}
			}
		}
	}

	return images, nil
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
// It checks that current and desired application images still exist, and that OS images are still available.
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
// It returns the count of successfully removed images and any error encountered.
// Errors during individual removals are logged but don't stop the process.
func (m *manager) removeEligibleImages(ctx context.Context, eligibleImages []string) (int, error) {
	var removedCount int
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
			m.log.Debugf("Removed image: %s", imageRef)
		} else {
			m.log.Warnf("Failed to remove image %s: %v", imageRef, err)
			removalErrors = append(removalErrors, fmt.Errorf("failed to remove image %s: %w", imageRef, err))
		}
	}

	// Return error only if all removals failed
	if len(removalErrors) == len(eligibleImages) && len(eligibleImages) > 0 {
		return removedCount, fmt.Errorf("all image removals failed: %d errors", len(removalErrors))
	}

	// Log summary if there were any failures
	if len(removalErrors) > 0 {
		m.log.Warnf("Image pruning completed with %d failures out of %d attempts", len(removalErrors), len(eligibleImages))
	}

	return removedCount, nil
}

// removeEligibleArtifacts removes the list of eligible artifacts from Podman storage.
// It returns the count of successfully removed artifacts and any error encountered.
// Errors during individual removals are logged but don't stop the process.
func (m *manager) removeEligibleArtifacts(ctx context.Context, eligibleArtifacts []string) (int, error) {
	var removedCount int
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
			m.log.Debugf("Removed artifact: %s", artifactRef)
		} else {
			m.log.Warnf("Failed to remove artifact %s: %v", artifactRef, err)
			removalErrors = append(removalErrors, fmt.Errorf("failed to remove artifact %s: %w", artifactRef, err))
		}
	}

	// Return error only if all removals failed
	if len(removalErrors) == len(eligibleArtifacts) && len(eligibleArtifacts) > 0 {
		return removedCount, fmt.Errorf("all artifact removals failed: %d errors", len(removalErrors))
	}

	// Log summary if there were any failures
	if len(removalErrors) > 0 {
		m.log.Warnf("Artifact pruning completed with %d failures out of %d attempts", len(removalErrors), len(eligibleArtifacts))
	}

	return removedCount, nil
}
