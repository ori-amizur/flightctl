package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/flightctl/flightctl/api/v1beta1"
	"github.com/flightctl/flightctl/internal/agent/client"
	"github.com/flightctl/flightctl/internal/agent/device/dependency"
	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	"github.com/flightctl/flightctl/pkg/log"
	"github.com/samber/lo"
)

// CacheEntry represents cached nested targets extracted from image-based applications.
type CacheEntry struct {
	Name string
	// Parent is the parent image from which child OCI targets were extracted.
	Parent dependency.OCIPullTarget
	// Children are OCI targets extracted from parent image.
	Children []dependency.OCIPullTarget
}

// OCITargetCache caches child OCI targets extracted from parents.
type OCITargetCache struct {
	entries map[string]CacheEntry
}

// NewOCITargetCache creates a new cache instance
func NewOCITargetCache() *OCITargetCache {
	return &OCITargetCache{
		entries: make(map[string]CacheEntry),
	}
}

// Get retrieves cached nested targets for the given entity name.
// Returns the entry and true if found, empty entry and false otherwise.
func (c *OCITargetCache) Get(name string) (CacheEntry, bool) {
	entry, found := c.entries[name]
	return entry, found
}

// Set stores a cache entry.
func (c *OCITargetCache) Set(entry CacheEntry) {
	c.entries[entry.Name] = entry
}

// GC removes cache entries for entities not in the activeNames list.
// This prevents unbounded cache growth as entities are added/removed.
func (c *OCITargetCache) GC(activeNames []string) {
	// build set of active names for O(1) lookup
	active := make(map[string]struct{}, len(activeNames))
	for _, name := range activeNames {
		active[name] = struct{}{}
	}

	// remove entries not in active set
	for name := range c.entries {
		if _, isActive := active[name]; !isActive {
			delete(c.entries, name)
		}
	}
}

// Len returns the number of entries in the cache
func (c *OCITargetCache) Len() int {
	return len(c.entries)
}

// Clear removes all entries from the cache
func (c *OCITargetCache) Clear() {
	c.entries = make(map[string]CacheEntry)
}

// GetOrExtractNestedTargets gets nested targets from cache or extracts them if not cached.
// It checks if the image/artifact exists locally, gets its digest, and uses cache if digest matches.
// Returns the nested targets, AppData (nil on cache hit, non-nil when extracted), and error.
// The AppData should be stored in appDataCache by the caller if it's not nil.
func (c *OCITargetCache) GetOrExtractNestedTargets(
	ctx context.Context,
	log *log.PrefixLogger,
	podman *client.Podman,
	readWriter fileio.ReadWriter,
	appSpec *v1beta1.ApplicationProviderSpec,
	imageSpec *v1beta1.ImageApplicationProviderSpec,
	pullSecret *client.PullSecret,
) ([]dependency.OCIPullTarget, *AppData, error) {
	appName, err := ResolveImageAppName(appSpec)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving app name: %w", err)
	}

	imageRef := imageSpec.Image

	// Detect if reference is an artifact or image and check if it exists locally
	var digest string
	var ociType dependency.OCIType
	var exists bool

	// Check if it's an image first (most common case)
	if podman.ImageExists(ctx, imageRef) {
		ociType = dependency.OCITypeImage
		exists = true
		digest, err = podman.ImageDigest(ctx, imageRef)
		if err != nil {
			return nil, nil, fmt.Errorf("getting image digest for %s: %w", imageRef, err)
		}
	} else if podman.ArtifactExists(ctx, imageRef) {
		ociType = dependency.OCITypeArtifact
		exists = true
		digest, err = podman.ArtifactDigest(ctx, imageRef)
		if err != nil {
			return nil, nil, fmt.Errorf("getting artifact digest for %s: %w", imageRef, err)
		}
	}

	if !exists {
		return nil, nil, fmt.Errorf("reference %s for app %s not available locally", imageRef, appName)
	}

	// Check cache
	if cachedEntry, found := c.Get(appName); found {
		if cachedEntry.Parent.Digest == digest {
			// cache hit - parent digest matches
			log.Debugf("Using cached nested targets for app %s (digest: %s)", appName, digest)
			return cachedEntry.Children, nil, nil // AppData is nil on cache hit
		}
		log.Debugf("Cache invalidated for app %s - digest changed from %s to %s", appName, cachedEntry.Parent.Digest, digest)
	}

	// cache miss or invalid - extract nested targets for this image
	appData, err := ExtractNestedTargetsFromImage(
		ctx,
		log,
		podman,
		readWriter,
		appSpec,
		imageSpec,
		pullSecret,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("extracting nested targets for app %s: %w", appName, err)
	}

	// Update cache
	cacheEntry := CacheEntry{
		Name: appName,
		Parent: dependency.OCIPullTarget{
			Type:      ociType,
			Reference: imageRef,
			Digest:    digest,
		},
		Children: appData.Targets,
	}
	c.Set(cacheEntry)
	log.Debugf("Cached %d nested targets for app %s (type: %s, digest: %s)", len(appData.Targets), appName, ociType, digest)

	return appData.Targets, appData, nil // Return AppData so caller can store it in appDataCache
}

// CollectNestedTargetsFromSpec collects nested OCI targets from all image-based applications in a device spec.
// It uses caching to avoid re-extracting targets when the parent image digest hasn't changed.
// Returns all nested targets, AppData map (keyed by app name, only contains entries that were extracted),
// whether requeue is needed (if any images aren't available yet), and active app names.
func (c *OCITargetCache) CollectNestedTargetsFromSpec(
	ctx context.Context,
	log *log.PrefixLogger,
	podman *client.Podman,
	readWriter fileio.ReadWriter,
	deviceSpec *v1beta1.DeviceSpec,
	pullSecret *client.PullSecret,
) ([]dependency.OCIPullTarget, map[string]*AppData, bool, []string, error) {
	if deviceSpec == nil || deviceSpec.Applications == nil {
		return []dependency.OCIPullTarget{}, make(map[string]*AppData), false, []string{}, nil
	}

	var allNestedTargets []dependency.OCIPullTarget
	appDataMap := make(map[string]*AppData)
	var activeAppNames []string
	needsRequeue := false

	for _, appSpec := range lo.FromPtr(deviceSpec.Applications) {
		appName := lo.FromPtr(appSpec.Name)
		activeAppNames = append(activeAppNames, appName)

		providerType, err := appSpec.Type()
		if err != nil {
			return nil, nil, false, nil, fmt.Errorf("getting provider type for app %s: %w", appName, err)
		}

		// only image-based apps have nested targets extracted from parent images
		if providerType != v1beta1.ImageApplicationProviderType {
			continue
		}

		imageSpec, err := appSpec.AsImageApplicationProviderSpec()
		if err != nil {
			return nil, nil, false, nil, fmt.Errorf("getting image spec for app %s: %w", appName, err)
		}

		// Get or extract nested targets (with caching)
		nestedTargets, appData, err := c.GetOrExtractNestedTargets(
			ctx,
			log,
			podman,
			readWriter,
			&appSpec,
			&imageSpec,
			pullSecret,
		)
		if err != nil {
			// Check if error indicates image doesn't exist locally (not available yet)
			// GetOrExtractNestedTargets returns this specific error when image doesn't exist
			if strings.Contains(err.Error(), "not available locally") {
				log.Debugf("Reference %s for app %s not available yet, skipping nested extraction", imageSpec.Image, appName)
				needsRequeue = true
				continue
			}
			return nil, nil, false, nil, err
		}

		// Store AppData in map if it was extracted (not from cache)
		// AppData is nil on cache hit, non-nil when extracted
		if appData != nil {
			appDataMap[appName] = appData
		}

		allNestedTargets = append(allNestedTargets, nestedTargets...)
	}

	return allNestedTargets, appDataMap, needsRequeue, activeAppNames, nil
}

// CollectNestedTargetReferencesFromSpec collects nested OCI target references (as strings) from all image-based applications.
// This is a convenience method for use cases that only need the reference strings (e.g., pruning).
// It uses the same caching logic as CollectNestedTargetsFromSpec.
// Errors during extraction are logged but don't block collection (best-effort).
func (c *OCITargetCache) CollectNestedTargetReferencesFromSpec(
	ctx context.Context,
	log *log.PrefixLogger,
	podman *client.Podman,
	readWriter fileio.ReadWriter,
	deviceSpec *v1beta1.DeviceSpec,
	pullSecret *client.PullSecret,
) ([]string, error) {
	targets, _, _, _, err := c.CollectNestedTargetsFromSpec(ctx, log, podman, readWriter, deviceSpec, pullSecret)
	if err != nil {
		// For pruning, we want to continue even if some extractions fail
		// Log the error but return what we have
		log.Debugf("Some nested target extractions failed (continuing with available targets): %v", err)
		// If we got no targets, return the error; otherwise return what we have
		if len(targets) == 0 {
			return nil, err
		}
	}

	refsSeen := make(map[string]struct{})
	var references []string
	for _, target := range targets {
		if target.Reference != "" {
			if _, seen := refsSeen[target.Reference]; !seen {
				refsSeen[target.Reference] = struct{}{}
				references = append(references, target.Reference)
			}
		}
	}

	return references, nil
}
