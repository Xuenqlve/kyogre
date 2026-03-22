package test_case

import (
	commonMySQL "github.com/xuenqlve/common/data_source/mysql"
	"github.com/xuenqlve/kyogre/internal/data_source"
	ds "github.com/xuenqlve/kyogre/pkg/data_source/mysql"
)

const (
	DefaultMySQLHost     = "127.0.0.1"
	DefaultMySQLPort     = 3306
	DefaultMySQLUsername = "root"
	DefaultMySQLPassword = "root"
)

func MySQLDataSourceTestConfig(name string) map[string]any {
	return map[string]any{
		name: commonMySQL.Config{
			Host:     DefaultMySQLHost,
			Username: DefaultMySQLUsername,
			Password: DefaultMySQLPassword,
			Port:     DefaultMySQLPort,
		},
	}
}

func PrepareMySQLDataSource(pipeline, name string) error {
	dataSource, err := data_source.GetDataSource(ds.MySQL)
	if err != nil {
		return err
	}
	return dataSource.Configure(pipeline, MySQLDataSourceTestConfig(name))
}
