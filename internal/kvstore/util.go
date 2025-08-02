package kvstore

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func StoreRenderedVersion(ctx context.Context, kvStore KVStore, orgId uuid.UUID, name, renderedVersion string) error {
	if name == "" {
		return fmt.Errorf("device name is required to store rendered version")
	}
	kvKey := DeviceKey{
		OrgID:      orgId,
		DeviceName: name,
	}

	if renderedVersion == "" {
		return fmt.Errorf("device rendered version is required to store rendered version")
	}
	_, err := kvStore.SetNX(ctx, kvKey.ComposeKey(), []byte(renderedVersion))
	return err
}
