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
	KafkaProducer sarama.SyncProducer
	KafkaConsumer sarama.ConsumerGroup
)

func InitKafka() error {
	cfg := config.Global.Kafka

	if len(cfg.Brokers) == 0 {
		logger.Warn("Kafka 未配置，跳过初始化")
		return nil
	}

	// 初始化生产者
	producer, err := kafka.InitProducer(cfg.Brokers)
	if err != nil {
		return fmt.Errorf("初始化 Kafka 生产者失败: %w", err)
	}

	// 初始化消费者 (可选)
	if cfg.GroupID != "" && cfg.Topic != "" {
		consumer, err := kafka.InitConsumerGroup(cfg.Brokers, cfg.GroupID, []string{cfg.Topic, "user-events"})
		if err != nil {
			return fmt.Errorf("初始化 Kafka 消费者失败: %w", err)
		}
		KafkaConsumer = consumer
	}

	logger.Info("Kafka 初始化成功",
		zap.Strings("brokers", cfg.Brokers),
		zap.String("topic", cfg.Topic))
	return nil
}
