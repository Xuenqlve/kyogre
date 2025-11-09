package mongodb

import (
	"github.com/xuenqlve/common/data_source/mongodb"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/kyogre/internal/data_source"
	"go.mongodb.org/mongo-driver/mongo"
)

func Config(key string) (mongodb.Config, error) {
	ds, err := data_source.GetDataSource(MongoDB)
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

func Connection(key string) (*mongo.Client, error) {
	ds, err := data_source.GetDataSource(MongoDB)
	if err != nil {
		return nil, errors.Trace(err)
	}
	conn, err := ds.CreateDataSource(key)
	if err != nil {
		return nil, errors.Trace(err)
	}
	client, ok := conn.(*mongo.Client)
	if !ok {
		return nil, errors.Errorf("not a valid mongodb connection")
	}
	return client, nil
}
