package scan

import (
	"terrasync/log"

	"terrasync/object"
	"time"

	"github.com/IBM/sarama"
)

// KafkaProducer 是Kafka生产者的封装
type KafkaProducer struct {
	producer sarama.SyncProducer
}

// NewKafkaProducer 创建一个新的Kafka生产者
func NewKafkaProducer(brokers []string) (*KafkaProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3
	// 设置连接超时时间为5秒
	config.Net.DialTimeout = 2 * time.Second
	config.Net.ReadTimeout = 2 * time.Second
	config.Net.WriteTimeout = 2 * time.Second

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	return &KafkaProducer{producer: producer}, nil
}

// SendMessage 发送消息到Kafka
func (kp *KafkaProducer) SendMessage(topic string, fileInfo object.FileInfo) error {
	// 创建消息
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(fileInfo.Key()),
	}

	// 发送消息
	_, _, err := kp.producer.SendMessage(msg)
	if err != nil {
		return err
	}

	log.Infof("Successfully sent message to Kafka topic %s: %s", topic, fileInfo.Key())
	return nil
}

// Close 关闭Kafka生产者
func (kp *KafkaProducer) Close() error {
	return kp.producer.Close()
}
