package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
	kafakDs "github.com/xuenqlve/common/data_source/kafka"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/kyogre/internal/data_source"
)

func Config(key string) (kafakDs.Config, error) {
	datasource, err := data_source.GetDataSource(KafKa)
	if err != nil {
		return kafakDs.Config{}, errors.Trace(err)
	}
	cfg, err := datasource.DataSourceConfig(key)
	if err != nil {
		return kafakDs.Config{}, errors.Trace(err)
	}
	config, ok := cfg.(kafakDs.Config)
	if !ok {
		return kafakDs.Config{}, errors.Trace(err)
	}
	return config, nil
}

func QueryPartition(key, topic string) ([]kafka.Partition, error) {
	config, err := Config(key)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return config.QueryPartition(topic)
}

type ReaderClient struct {
	Reader    *kafka.Reader
	Config    kafakDs.Config
	Topic     string
	Partition int
}

func (r *ReaderClient) Close() error {
	return r.Reader.Close()
}

func NewReaderClient(key, topic string, partition int) (*ReaderClient, error) {
	config, err := Config(key)
	if err != nil {
		return nil, errors.Trace(err)
	}
	reader, err := config.CreateReader(topic, partition)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &ReaderClient{
		Reader:    reader,
		Config:    config,
		Topic:     topic,
		Partition: partition,
	}, nil
}

type WriterClient struct {
	Writer *kafka.Writer
	Config kafakDs.Config
}

func NewWriterClient(key string, balancer kafka.Balancer) (*WriterClient, error) {
	config, err := Config(key)
	if err != nil {
		return nil, errors.Trace(err)
	}
	writer, err := config.CreateWriterCustomBalancer(balancer)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &WriterClient{
		Writer: writer,
		Config: config,
	}, nil
}

// WriteMessage 写入单条消息
func (c *WriterClient) WriteMessage(ctx context.Context, topic string, key, value []byte) error {
	if c.Writer == nil {
		return errors.New("kafka-docker writer is not initialized")
	}

	msg := kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	}

	return c.Writer.WriteMessages(ctx, msg)
}

// WriteMessages 批量写入消息
func (c *WriterClient) WriteMessages(ctx context.Context, messages ...kafka.Message) error {
	if c.Writer == nil {
		return errors.New("kafka-docker writer is not initialized")
	}
	return c.Writer.WriteMessages(ctx, messages...)
}

func (c *WriterClient) Close() error {
	var err error
	if c.Writer != nil {
		if closeErr := c.Writer.Close(); closeErr != nil {
			err = closeErr
		}
	}
	return err
}
