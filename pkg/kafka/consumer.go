package kafka

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
	"go.uber.org/zap"

	"github.com/VennLe/charlotte/internal/model"
	"github.com/VennLe/charlotte/pkg/logger"
)

// ConsumerGroupHandler 消费者组处理器
type ConsumerGroupHandler struct {
	ready chan bool
}

func NewConsumerGroupHandler() *ConsumerGroupHandler {
	return &ConsumerGroupHandler{
		ready: make(chan bool),
	}
}

func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			h.handleMessage(message.Topic, message.Value)
			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

func (h *ConsumerGroupHandler) handleMessage(topic string, data []byte) {
	logger.Info("收到 Kafka 消息",
		zap.String("topic", topic),
		zap.String("data", string(data)))

	switch topic {
	case "user-events":
		var event model.UserEvent
		if err := json.Unmarshal(data, &event); err != nil {
			logger.Error("解析用户事件失败", zap.Error(err))
			return
		}
		h.handleUserEvent(event)
	default:
		logger.Warn("未知 topic", zap.String("topic", topic))
	}
}

func (h *ConsumerGroupHandler) handleUserEvent(event model.UserEvent) {
	logger.Info("处理用户事件",
		zap.String("event_type", event.EventType),
		zap.Uint("user_id", event.UserID),
		zap.String("username", event.Username))

	// 这里可以添加具体的业务逻辑
	// 例如：发送邮件通知、更新搜索引擎索引、清理缓存等

	switch event.EventType {
	case "user_created":
		// 发送欢迎邮件
		logger.Info("新用户注册，发送欢迎邮件", zap.String("email", event.Email))
	case "user_updated":
		// 更新缓存
		logger.Info("用户信息更新，清理缓存", zap.Uint("user_id", event.UserID))
	case "user_deleted":
		// 清理相关数据
		logger.Info("用户删除，清理相关数据", zap.Uint("user_id", event.UserID))
	}
}

// InitConsumerGroup 初始化消费者组
func InitConsumerGroup(brokers []string, groupID string, topics []string) (sarama.ConsumerGroup, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, err
	}

	// 启动消费
	go func() {
		handler := NewConsumerGroupHandler()
		for {
			if err := consumerGroup.Consume(context.Background(), topics, handler); err != nil {
				logger.Error("Kafka 消费错误", zap.Error(err))
			}
		}
	}()

	return consumerGroup, nil
}
