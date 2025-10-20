package redis

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/kyogre/internal/common/errors"
	"github.com/xuenqlve/kyogre/internal/data_source"
)

const REDIS data_source.DataSourceType = "redis"

const (
	standalone = "standalone"
	cluster    = "cluster"
)

type DataSource struct {
	pipelineName  string
	dataSourceMap map[string]Config
}

type Config struct {
	Type     string `json:"type"`
	Address  string `json:"address"`
	Username string `json:"username"`
	Password string `json:"password"`
	IsTls    bool   `json:"is_tls" mapstructure:"is_tls"`
}

func init() {
	data_source.RegisterPlugin(REDIS, &DataSource{}, true)
}

func (ds *DataSource) Configure(pipelineName string, data map[string]any) error {
	ds.pipelineName = pipelineName
	ds.dataSourceMap = map[string]Config{}
	if err := mapstructure.Decode(data, &ds.dataSourceMap); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (ds *DataSource) CreateDataSource(dataSourceName string) (any, error) {
	cfg, ok := ds.dataSourceMap[dataSourceName]
	if !ok {
		return nil, errors.Errorf("dataSource %s is not exist", dataSourceName)
	}

	switch cfg.Type {
	case standalone:
		return CreateRedisConnection(cfg.Address, cfg.Username, cfg.Password, cfg.IsTls)
	case cluster:
		return CreateRedisClusterConnection(cfg.Address, cfg.Username, cfg.Password, cfg.IsTls)
	}
	return nil, fmt.Errorf("[DataSource] Unknown parameter type:%v", cfg.Type)
}

func (ds *DataSource) DataSourceConfig(dataSourceName string) (any, error) {
	dataSourceConfig, ok := ds.dataSourceMap[dataSourceName]
	if !ok {
		return nil, errors.Errorf("dataSource %s is not exist", dataSourceName)
	}
	return dataSourceConfig, nil
}
