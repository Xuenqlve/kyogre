package mongodb

import (
	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/data_source/mongodb"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/kyogre/internal/data_source"
)

const MongoDB data_source.DataSourceType = "mongodb"

type DataSource struct {
	pipelineName  string
	dataSourceMap map[string]mongodb.Config
}

func init() {
	data_source.RegisterPlugin(MongoDB, &DataSource{}, true)
}

func (ds *DataSource) Configure(pipelineName string, data map[string]any) error {
	ds.pipelineName = pipelineName
	ds.dataSourceMap = map[string]mongodb.Config{}
	if err := mapstructure.Decode(data, &ds.dataSourceMap); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (ds *DataSource) CreateDataSource(dataSourceName string) (any, error) {
	dsCfg, ok := ds.dataSourceMap[dataSourceName]
	if !ok {
		return nil, errors.Errorf("data source %s not exist", dataSourceName)
	}
	return dsCfg.Connect()
}

func (ds *DataSource) DataSourceConfig(dataSourceName string) (any, error) {
	dsCfg, ok := ds.dataSourceMap[dataSourceName]
	if !ok {
		return nil, errors.Errorf("data source %s not exist", dataSourceName)
	}
	return dsCfg, nil
}
