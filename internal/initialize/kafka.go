package initialize

import (
	"fmt"
	"github.com/IBM/sarama"
	"go.uber.org/zap"

	"github.com/VennLe/charlotte/internal/config"
	"github.com/VennLe/charlotte/pkg/kafka"
	"github.com/VennLe/charlotte/pkg/logger"
)

var (
	KafkaProducer *sarama.SyncProducer
	KafkaConsumer sarama.ConsumerGroup
)

func InitKafka() error {
	cfg := config.Global.Kafka

	if len(cfg.Brokers) == 0 {
		logger.Warn("Kafka 未配置，跳过初始化")
		return nil
	}

	// 初始化生产者
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Retry.Max = 3
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true

	producer, err := sarama.NewSyncProducer(cfg.Brokers, saramaConfig)
	if err != nil {
		return fmt.Errorf("初始化 Kafka 生产者失败: %w", err)
	}
	KafkaProducer = &producer

	// 初始化消费者 (可选)
	consumerTopics := []string{cfg.Topic}
	if cfg.GroupID != "" && len(consumerTopics) > 0 && consumerTopics[0] != "" {
		consumer, err := kafka.InitConsumerGroup(cfg.Brokers, cfg.GroupID, consumerTopics)
		if err != nil {
			return fmt.Errorf("初始化 Kafka 消费者失败: %w", err)
		}
		KafkaConsumer = consumer
		logger.Info("Kafka 消费者初始化成功",
			zap.String("group_id", cfg.GroupID),
			zap.Strings("topics", consumerTopics))
	}

	logger.Info("Kafka 初始化成功",
		zap.Strings("brokers", cfg.Brokers),
		zap.String("topic", cfg.Topic))
	return nil
}

// CloseKafka 关闭 Kafka 连接
func CloseKafka() {
	if KafkaProducer != nil {
		if err := (*KafkaProducer).Close(); err != nil {
			logger.Error("关闭 Kafka 生产者失败", zap.Error(err))
		} else {
			logger.Info("Kafka 生产者已关闭")
		}
	}

	if KafkaConsumer != nil {
		if err := KafkaConsumer.Close(); err != nil {
			logger.Error("关闭 Kafka 消费者失败", zap.Error(err))
		} else {
			logger.Info("Kafka 消费者已关闭")
		}
	}
}
