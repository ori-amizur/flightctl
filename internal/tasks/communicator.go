package tasks

import (
	"context"
	"encoding/json"

	"github.com/flightctl/flightctl/pkg/queues"
	"github.com/sirupsen/logrus"
)

const FleetQueue = "fleet-queue"

type Communicator interface {
	NewReceiver() (Receiver, error)
	NewSender() (Sender, error)
	Stop()
	Wait()
}

type Receiver interface {
	Receive(handler ReceiveHandler) error
}

type Sender interface {
	Send(reference *ResourceReference) error
}

type communicator struct {
	provider queues.Provider
}

func NewCommunicator(url string, log logrus.FieldLogger) Communicator {
	return &communicator{
		provider: queues.NewAmqpProvider(url, log),
	}
}

func (c *communicator) NewReceiver() (Receiver, error) {
	consumer, err := c.provider.NewConsumer(FleetQueue)
	if err != nil {
		return nil, err
	}
	return &receiver{
		consumer: consumer,
	}, nil
}

func (c *communicator) NewSender() (Sender, error) {
	publisher, err := c.provider.NewPublisher(FleetQueue)
	if err != nil {
		return nil, err
	}
	return &sender{
		publisher: publisher,
	}, nil
}

func (c *communicator) Stop() {
	c.provider.Stop()
}

func (c *communicator) Wait() {
	c.provider.Wait()
}

type receiver struct {
	consumer queues.Consumer
}

func parseAndDispatch(handler ReceiveHandler) queues.ConsumeHandler {
	return func(ctx context.Context, payload []byte, log logrus.FieldLogger) error {
		var resourceReference ResourceReference
		if err := json.Unmarshal(payload, &resourceReference); err != nil {
			return err
		}
		return handler(ctx, &resourceReference, log)
	}
}

type ReceiveHandler func(ctx context.Context, reference *ResourceReference, log logrus.FieldLogger) error

func (c *receiver) Receive(handler ReceiveHandler) error {
	return c.consumer.Consume(parseAndDispatch(handler))
}

type sender struct {
	publisher queues.Publisher
}

func (s *sender) Send(reference *ResourceReference) error {
	b, err := json.Marshal(reference)
	if err != nil {
		return err
	}
	return s.publisher.Publish(b)
}
