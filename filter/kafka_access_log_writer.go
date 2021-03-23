package filter

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Shopify/sarama"
	"github.com/sirupsen/logrus"
)

type kafkaAccessLogWriter struct {
	topic    string
	producer *kafkaProducer
}

func KafkaAccessLogWriter(brokers []string, topic string) AccessLogWriter {
	if len(brokers) == 0 || topic == "" {
		return nil
	}

	producer, err := newKafkaProducer(brokers, topic, func(c *sarama.Config) {
		c.Producer.RequiredAcks = sarama.WaitForLocal       // Only wait for the leader to ack
		c.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms

	})
	if err != nil {
		logrus.WithError(err).Error("Fail to create kafka producer")
		return nil
	}

	return &kafkaAccessLogWriter{
		topic:    topic,
		producer: producer,
	}
}

func (w *kafkaAccessLogWriter) Write(accessLog *AccessLog) {
	if w.producer == nil {
		logrus.Error("Kafka producer is nil")
		return
	}

	if err := w.producer.Send(accessLog); err != nil {
		logrus.WithError(err).Error("Fail to send accesslog to kafka")
		return
	}
}

type kafkaProducer struct {
	topic    string
	producer sarama.AsyncProducer
}

func newKafkaProducer(brokers []string, topic string, options ...func(*sarama.Config)) (*kafkaProducer, error) {
	kafkaConfig := sarama.NewConfig()
	for _, option := range options {
		option(kafkaConfig)
	}

	producer, err := sarama.NewAsyncProducer(brokers, kafkaConfig)
	if err != nil {
		return nil, err
	}

	go func() {
		for err := range producer.Errors() {
			log.Printf("Failed to send log entry to kafka : %v\n", err)
		}
	}()

	return &kafkaProducer{
		topic:    topic,
		producer: producer,
	}, nil
}

func (p *kafkaProducer) Send(v interface{}) error {
	msg, err := json.Marshal(v)
	if err != nil {
		return err
	}

	if p == nil {
		log.Println("Kafka producer is nil")
		return fmt.Errorf("Kafka producer is nil")
	}
	if p.producer == nil {
		log.Println("Kafka producer is nil")
		return fmt.Errorf("Kafka producer is nil")
	}

	p.producer.Input() <- &sarama.ProducerMessage{
		Topic: p.topic,
		Value: sarama.ByteEncoder(msg),
	}

	return nil
}
func (p *kafkaProducer) SendWithKey(v interface{}, key string) error {
	msg, err := json.Marshal(v)
	if err != nil {
		return err
	}

	if p.producer == nil {
		log.Println("Kafka producer is nil")
		return fmt.Errorf("Kafka producer is nil")
	}

	if key == "" {
		log.Println("producer Key is empty")
		return fmt.Errorf("producer Key is empty")
	}

	p.producer.Input() <- &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(msg),
	}

	return nil
}

func (p *kafkaProducer) Close() error {
	return p.producer.Close()
}
