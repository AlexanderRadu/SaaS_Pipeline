package status

import (
	"context"
	"encoding/json"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type InstanceUpdateMessage struct {
	InstanceId    string `json:"instance_id"`
	Status        string `json:"status"`
	ConnectionURL string `json:"connection_url"`
}

type Consumer struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	repo *Repository
	log  *slog.Logger
}

func NewConsumer(amqpURL string, repo *Repository, log *slog.Logger) (*Consumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	_, err = ch.QueueDeclare(
		"instance_updates",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return &Consumer{
		conn: conn,
		ch:   ch,
		repo: repo,
		log:  log,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	msgs, err := c.ch.Consume("instance_updates", "status_service", false, false, false, false, nil)
	if err != nil {
		return err
	}
	c.log.Info("RabbitMQ consumer started, waiting for messages...")

	for {
		select {
		case <-ctx.Done():
			c.log.Info("stopping RabbitMQ consumer...")
			return nil

		case d := <-msgs:
			var msg InstanceUpdateMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				c.log.Error("failed to parse message", slog.Any("error", err))
				d.Nack(false, false)
				continue
			}
			err := c.repo.UpdateInstanceStatus(msg.InstanceId, msg.Status, msg.ConnectionURL)
			if err != nil {
				c.log.Error("failed to update status in DB", slog.Any("error", err))
				d.Nack(false, false)
				continue
			}
			d.Ack(false)
			c.log.Info("instance status updated", slog.String("id", msg.InstanceId), slog.String("statis", msg.Status))

		}

	}

}

func (c *Consumer) Close(ctx context.Context) {
	if c.ch != nil {
		c.ch.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}
