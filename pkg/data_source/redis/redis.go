package redis

import (
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/kyogre/internal/data_source"
)

func Config(key string) (RedisConfig, error) {
	ds, err := data_source.GetDataSource(REDIS)
	if err != nil {
		return RedisConfig{}, errors.Trace(err)
	}
	c, err := ds.DataSourceConfig(key)
	if err != nil {
		return RedisConfig{}, errors.Trace(err)
	}
	cfg, ok := c.(RedisConfig)
	if !ok {
		return RedisConfig{}, errors.Errorf("not a valid redis connection")
	}
	return cfg, nil
}

func Connection(key string) (any, error) {
	ds, err := data_source.GetDataSource(REDIS)
	if err != nil {
		return nil, errors.Trace(err)
	}
	conn, err := ds.CreateDataSource(key)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return conn, nil
}
