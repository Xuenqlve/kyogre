package mysql

import (
	"database/sql"

	"github.com/xuenqlve/common/data_source/mysql"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/kyogre/internal/data_source"
)

func Config(key string) (mysql.Config, error) {
	ds, err := data_source.GetDataSource(MySQL)
	if err != nil {
		return mysql.Config{}, errors.Trace(err)
	}
	c, err := ds.DataSourceConfig(key)
	if err != nil {
		return mysql.Config{}, errors.Trace(err)
	}
	cfg, ok := c.(mysql.Config)
	if !ok {
		return mysql.Config{}, errors.Errorf("not a valid mysql-row connection")
	}
	return cfg, nil
}

func Connection(key string) (*sql.DB, error) {
	ds, err := data_source.GetDataSource(MySQL)
	if err != nil {
		return nil, errors.Trace(err)
	}
	conn, err := ds.CreateDataSource(key)
	if err != nil {
		return nil, errors.Trace(err)
	}
	db, ok := conn.(*sql.DB)
	if !ok {
		return nil, errors.Errorf("not a valid mysql-row connection")
	}
	return db, nil
}
