package data_source

import (
	"github.com/xuenqlve/kyogre/internal/common/errors"
	"github.com/xuenqlve/kyogre/internal/data_source"
	"github.com/xuenqlve/kyogre/pkg/data_source/kafka"
	"github.com/xuenqlve/kyogre/pkg/data_source/mongodb"
	"github.com/xuenqlve/kyogre/pkg/data_source/mysql"
	"github.com/xuenqlve/kyogre/pkg/data_source/redis"
)

func MySQLConfig(key string) (mysql.Config, error) {
	ds, err := data_source.GetDataSource(mysql.MySQL)
	if err != nil {
		return mysql.Config{}, errors.Trace(err)
	}
	c, err := ds.DataSourceConfig(key)
	if err != nil {
		return mysql.Config{}, errors.Trace(err)
	}
	cfg, ok := c.(mysql.Config)
	if !ok {
		return mysql.Config{}, errors.Errorf("not a valid mysql connection")
	}
	return cfg, nil
}

func RedisConfig(key string) (redis.Config, error) {
	ds, err := data_source.GetDataSource(redis.REDIS)
	if err != nil {
		return redis.Config{}, errors.Trace(err)
	}
	c, err := ds.DataSourceConfig(key)
	if err != nil {
		return redis.Config{}, errors.Trace(err)
	}
	cfg, ok := c.(redis.Config)
	if !ok {
		return redis.Config{}, errors.Errorf("not a valid redis connection")
	}
	return cfg, nil
}

func MongoDBConfig(key string) (mongodb.Config, error) {
	ds, err := data_source.GetDataSource(mongodb.MongoDB)
	if err != nil {
		return mongodb.Config{}, errors.Trace(err)
	}
	c, err := ds.DataSourceConfig(key)
	if err != nil {
		return mongodb.Config{}, errors.Trace(err)
	}
	cfg, ok := c.(mongodb.Config)
	if !ok {
		return mongodb.Config{}, errors.Errorf("not a valid mongodb connection")
	}
	return cfg, nil
}

func KafkaConfig(key string) (kafka.Config, error) {
	ds, err := data_source.GetDataSource(kafka.KafKa)
	if err != nil {
		return kafka.Config{}, errors.Trace(err)
	}
	cfg, err := ds.DataSourceConfig(key)
	if err != nil {
		return kafka.Config{}, errors.Trace(err)
	}
	config, ok := cfg.(kafka.Config)
	if !ok {
		return kafka.Config{}, errors.Trace(err)
	}
	return config, nil
}
