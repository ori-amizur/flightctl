package queues

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/flightctl/flightctl/pkg/log"
	"github.com/flightctl/flightctl/pkg/reqid"
	"github.com/go-chi/chi/v5/middleware"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

type amqpProvider struct {
	ctx     context.Context
	cancel  context.CancelFunc
	url     string
	log     logrus.FieldLogger
	wg      *sync.WaitGroup
	queues  []*amqpQueue
	stopped bool
}

func NewAmqpProvider(url string, log logrus.FieldLogger) Provider {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
		}
	}()
	return &amqpProvider{
		ctx:    ctx,
		cancel: cancel,
		url:    url,
		log:    log,
		wg:     &wg,
	}
}

func (r *amqpProvider) newQueue(queueName string) (*amqpQueue, error) {
	var (
		err        error
		connection *amqp.Connection
		channel    *amqp.Channel
	)
	if r.stopped {
		return nil, errors.New("queue is stopped")
	}
	defer func() {
		if err != nil {
			if channel != nil {
				channel.Close()
			}
			if connection != nil {
				connection.Close()
			}
		}
	}()
	connection, err = amqp.Dial(r.url)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}
	channel, err = connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}
	_, err = channel.QueueDeclare(queueName,
		true,
		false,
		false,
		false,
		nil)

	if err != nil {
		return nil, fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}
	ret := &amqpQueue{
		name:       queueName,
		connection: connection,
		channel:    channel,
		log:        r.log,
		ctx:        r.ctx,
		wg:         r.wg,
	}
	r.queues = append(r.queues, ret)
	return ret, nil
}

func (a *amqpProvider) NewConsumer(queueName string) (Consumer, error) {
	q, err := a.newQueue(queueName)
	return q, err
}

func (a *amqpProvider) NewPublisher(queueName string) (Publisher, error) {
	q, err := a.newQueue(queueName)
	return q, err
}

func (a *amqpProvider) Stop() {
	a.cancel()
	for _, q := range a.queues {
		q.close()
	}
}

func (a *amqpProvider) Wait() {
	a.wg.Wait()
}

type amqpQueue struct {
	ctx        context.Context
	connection *amqp.Connection
	channel    *amqp.Channel
	name       string
	wg         *sync.WaitGroup
	log        logrus.FieldLogger
	closed     bool
}

func (r *amqpQueue) Publish(payload []byte) error {
	if r.closed {
		return errors.New("queue is closed")
	}
	return r.channel.Publish("",
		r.name,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         payload,
		})
}

func (r *amqpQueue) Consume(handler ConsumeHandler) error {
	msgs, err := r.channel.ConsumeWithContext(r.ctx,
		r.name,
		"",
		false,
		false,
		false,
		false,
		nil)
	if err != nil {
		return err
	}
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		var err error
		for {
			select {
			case <-r.ctx.Done():
				return
			case d, ok := <-msgs:
				requestID := reqid.NextRequestID()
				reqCtx := context.WithValue(r.ctx, middleware.RequestIDKey, requestID)
				log := log.WithReqIDFromCtx(reqCtx, r.log)
				if !ok {
					if !r.closed {
						log.Fatal("channel was closed by AMQP provider")
					}
					return
				}
				if err = handler(reqCtx, d.Body, log); err != nil {
					log.WithError(err).Errorf("failed to consume message: %s", string(d.Body))
				}
				if err = d.Ack(false); err != nil {
					log.WithError(err).Errorf("failed to acknowledge message")
				}
			}

		}
	}()
	return nil
}

func (a *amqpQueue) close() {
	a.closed = true
	if a.channel != nil {
		a.channel.Close()
	}
	if a.connection != nil {
		a.connection.Close()
	}
}
