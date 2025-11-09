package clickhouse

import (
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/xuenqlve/common/data_source/clickhouse"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/kyogre/internal/data_source"
)

func Config(key string) (clickhouse.Config, error) {
	ds, err := data_source.GetDataSource(ClickHouse)
	if err != nil {

		return clickhouse.Config{}, errors.Trace(err)
	}
	c, err := ds.DataSourceConfig(key)
	if err != nil {
		return clickhouse.Config{}, errors.Trace(err)
	}
	cfg, ok := c.(clickhouse.Config)
	if !ok {
		return clickhouse.Config{}, errors.Errorf("not a valid mysql-row connection")
	}
	return cfg, nil
}

func Connection(key string) (driver.Conn, error) {
	ds, err := data_source.GetDataSource(ClickHouse)
	if err != nil {
		return nil, errors.Trace(err)
	}
	conn, err := ds.CreateDataSource(key)
	if err != nil {
		return nil, errors.Trace(err)
	}
	db, ok := conn.(driver.Conn)
	if !ok {
		return nil, errors.Errorf("not a valid mysql-row connection")
	}
	return db, nil
}
