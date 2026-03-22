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

	DefaultPressureMySQLPort = 3307
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

func MySQLPressureDataSourceTestConfig(name string) map[string]any {
	return map[string]any{
		name: commonMySQL.Config{
			Host:     DefaultMySQLHost,
			Username: DefaultMySQLUsername,
			Password: DefaultMySQLPassword,
			Port:     DefaultPressureMySQLPort,
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

func MySQLDataSource(pipeline, source, pressure string) error {
	dataSource, err := data_source.GetDataSource(ds.MySQL)
	if err != nil {
		return err
	}
	return dataSource.Configure(pipeline, map[string]any{
		source: commonMySQL.Config{
			Host:     DefaultMySQLHost,
			Username: DefaultMySQLUsername,
			Password: DefaultMySQLPassword,
			Port:     DefaultPressureMySQLPort,
		},
		pressure: commonMySQL.Config{
			Host:     DefaultMySQLHost,
			Username: DefaultMySQLUsername,
			Password: DefaultMySQLPassword,
			Port:     DefaultPressureMySQLPort,
		},
	})
}
