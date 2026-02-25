package kafka

import (
	"github.com/IBM/sarama"
	"go.uber.org/zap"

	"github.com/VennLe/charlotte/pkg/logger"
)

// Producer Kafka 生产者接口
type Producer interface {
	SendMessage(topic string, message string) error
	SendMessageWithKey(topic string, key string, message string) error
	Close() error
}

type kafkaProducer struct {
	producer sarama.SyncProducer
}

var defaultProducer Producer

// InitProducer 初始化生产者
func InitProducer(brokers []string) (Producer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	defaultProducer = &kafkaProducer{producer: producer}
	return defaultProducer, nil
}

// GetProducer 获取默认生产者
func GetProducer() Producer {
	return defaultProducer
}

func (p *kafkaProducer) SendMessage(topic string, message string) error {
	return p.SendMessageWithKey(topic, "", message)
}

func (p *kafkaProducer) SendMessageWithKey(topic string, key string, message string) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(message),
	}

	if key != "" {
		msg.Key = sarama.StringEncoder(key)
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		logger.Error("Kafka 发送消息失败",
			zap.String("topic", topic),
			zap.Error(err))
		return err
	}

	logger.Debug("Kafka 消息已发送",
		zap.String("topic", topic),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset))

	return nil
}

func (p *kafkaProducer) Close() error {
	return p.producer.Close()
}
