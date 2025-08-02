package rendered_version

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flightctl/flightctl/internal/kvstore"
	"github.com/flightctl/flightctl/pkg/queues"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const queueName = "rendered_version_notifier"

type Broadcaster interface {
	Broadcast(ctx context.Context, orgId uuid.UUID, name string, renderedVersion string) error
}

type Subscriber interface {
	Subscribe(ctx context.Context, handler func(ctx context.Context, orgId uuid.UUID, name string, renderedVersion string) error) error
}

func NewBroadcaster(queuesProvider queues.Provider, kvStore kvstore.KVStore) (Broadcaster, error) {
	queuesPublisher, err := queuesProvider.NewBroadcaster(queueName)
	if err != nil {
		return nil, err
	}
	return &broadcaster{
		broadcaster: queuesPublisher,
		kvStore:     kvStore,
	}, nil
}

func NewSubscriber(queuesProvider queues.Provider) (Subscriber, error) {
	subscriber, err := queuesProvider.NewSubscriber(queueName)
	if err != nil {
		return nil, err
	}
	return &consumer{
		subscriber: subscriber,
	}, nil
}

type serializedResourceId struct {
	OrgId         uuid.UUID `json:"org_id"`
	Name          string    `json:"name"`
	RenderVersion string    `json:"render_version,omitempty"`
}
type broadcaster struct {
	broadcaster queues.Broadcaster
	kvStore     kvstore.KVStore
}

func (p *broadcaster) storeRenderedVersion(ctx context.Context, kvStore kvstore.KVStore, orgId uuid.UUID, name, renderedVersion string) error {
	if name == "" {
		return fmt.Errorf("device name is required to store rendered version")
	}
	kvKey := kvstore.DeviceKey{
		OrgID:      orgId,
		DeviceName: name,
	}

	if renderedVersion == "" {
		return fmt.Errorf("device rendered version is required to store rendered version")
	}
	_, err := kvStore.SetNX(ctx, kvKey.ComposeKey(), []byte(renderedVersion))
	return err
}

func (p *broadcaster) Broadcast(ctx context.Context, orgId uuid.UUID, name string, renderVersion string) error {
	if err := p.storeRenderedVersion(ctx, p.kvStore, orgId, name, renderVersion); err != nil {
		return fmt.Errorf("failed to store rendered version: %w", err)
	}
	s := serializedResourceId{
		OrgId:         orgId,
		Name:          name,
		RenderVersion: renderVersion,
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return p.broadcaster.Broadcast(ctx, b)
}

type consumer struct {
	subscriber queues.Subscriber
}

func (c *consumer) Subscribe(ctx context.Context, handler func(ctx context.Context, orgId uuid.UUID, name string, renderedVersion string) error) error {
	var s serializedResourceId
	queuesHandler := func(ctx context.Context, payload []byte, log logrus.FieldLogger) error {
		if err := json.Unmarshal(payload, &s); err != nil {
			log.WithError(err).Error("failed to unmarshal payload")
			return err
		}
		return handler(ctx, s.OrgId, s.Name, s.RenderVersion)
	}

	_, err := c.subscriber.Subscribe(ctx, queuesHandler)
	return err
}
