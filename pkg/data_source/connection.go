package data_source

import (
	"database/sql"

	"github.com/xuenqlve/kyogre/internal/common/errors"
	"github.com/xuenqlve/kyogre/internal/data_source"
	"github.com/xuenqlve/kyogre/pkg/data_source/mongodb"
	"github.com/xuenqlve/kyogre/pkg/data_source/mysql"
	"github.com/xuenqlve/kyogre/pkg/data_source/redis"
	"go.mongodb.org/mongo-driver/mongo"
)

func MySQLConnection(key string) (*sql.DB, error) {
	ds, err := data_source.GetDataSource(mysql.MySQL)
	if err != nil {
		return nil, errors.Trace(err)
	}
	conn, err := ds.CreateDataSource(key)
	if err != nil {
		return nil, errors.Trace(err)
	}
	db, ok := conn.(*sql.DB)
	if !ok {
		return nil, errors.Errorf("not a valid mysql connection")
	}
	return db, nil
}

func MongoDBConnection(key string) (*mongo.Client, error) {
	ds, err := data_source.GetDataSource(mongodb.MongoDB)
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

func RedisConnection(key string) (any, error) {
	ds, err := data_source.GetDataSource(redis.REDIS)
	if err != nil {
		return nil, errors.Trace(err)
	}
	conn, err := ds.CreateDataSource(key)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return conn, nil
}

//func KafkaWriterConnection(key string) (*kafka-docker.Client, error) {
//	ds, err := data_source.GetDataSource(kafka-docker.KafKa)
//	if err != nil {
//		return nil, errors.Trace(err)
//	}
//	conn, err := ds.CreateDataSource(key)
//	if err != nil {
//		return nil, errors.Trace(err)
//	}
//	client, ok := conn.(*kafka-docker.Client)
//	if !ok {
//		return nil, errors.Trace(err)
//	}
//	return client, nil
//}
