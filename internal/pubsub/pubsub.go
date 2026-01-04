package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	SimpleQueueDurable SimpleQueueType = iota
	SimpleQueueTransient
)

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange, queueName, key string,
	queueType SimpleQueueType) (*amqp.Channel, amqp.Queue, error) {

	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	queue, err := ch.QueueDeclare(
		queueName,
		queueType == SimpleQueueDurable,
		queueType != SimpleQueueDurable,
		queueType != SimpleQueueDurable,
		false,
		amqp.Table{"x-dead-letter-exchange": "peril_dlx"})
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	err = ch.QueueBind(
		queueName,
		key,
		exchange,
		false,
		nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	return ch, queue, nil
}

func PublishJson[T any](ch *amqp.Channel, exchange, key string, val T) error {
	jsonBytes, err := json.Marshal(val)
	if err != nil {
		return err
	}

	ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        jsonBytes,
	})

	return nil
}

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var data bytes.Buffer
	enc := gob.NewEncoder(&data)
	err := enc.Encode(val)
	if err != nil {
		return err
	}

	ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/gob",
			Body:        data.Bytes(),
		},
	)

	return nil
}

func Subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
	unmarshall func([]byte) (T, error),
) error {
	//make sure queue exists and it's bound to exchange
	ch, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}
	err = ch.Qos(10, 0, false)
	if err != nil {
		return err
	}
	msgs, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	go func() {
		defer ch.Close()
		for msg := range msgs {
			arg, err := unmarshall(msg.Body)
			if err != nil {
				log.Println(err)
				continue
			}

			ackType := handler(arg)
			switch ackType {
			case Ack:
				msg.Ack(false)
				log.Println("Message acknowledged:", msg.MessageId)
			case NackRequeue:
				msg.Nack(false, true)
				log.Println("Message reque, negative acknowledged:", msg.MessageId)
			case NackDiscard:
				msg.Nack(false, false)
				log.Println("Message discarded, negative acknowledged:", msg.MessageId)
			default:
				log.Println("Unknown ack type", msg.MessageId)
			}
		}
	}()
	return nil
}

func UnmarshallJSON[T any](data []byte) (T, error) {
	var arg T
	err := json.Unmarshal(data, &arg)
	return arg, err
}

func UnmarshallGob[T any](data []byte) (T, error) {
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	var arg T
	err := dec.Decode(&arg)
	return arg, err
}
