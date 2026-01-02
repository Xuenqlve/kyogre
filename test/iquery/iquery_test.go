package iquery

import (
	"fmt"
	"os"
	"testing"

	"github.com/xuenqlve/common/data_source/mysql"
	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/internal/data_source"
	ds "github.com/xuenqlve/kyogre/pkg/data_source/mysql"
)

const (
	pipeline        = "test_pressure"
	mysqlDataSource = "pressure"
)

func mysqlDataSourceTestConfig() map[string]any {
	return map[string]any{
		mysqlDataSource: mysql.Config{
			Host:     "127.0.0.1",
			Username: "root",
			Password: "root",
			Port:     3306,
		},
	}
}

func mysqlDataSourcePrepare() error {
	dataSource, err := data_source.GetDataSource(ds.MySQL)
	if err != nil {
		return err
	}
	if err = dataSource.Configure(pipeline, mysqlDataSourceTestConfig()); err != nil {
		return err
	}
	return nil
}

func TestMain(t *testing.M) {
	dir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("get work dir fail err:%v", err))
	}
	log.Init(log.DebugLevel, dir)
	if err = mysqlDataSourcePrepare(); err != nil {
		panic(fmt.Sprintf("mysqlDataSourcePrepare err:%v", err))
	}
	exitCode := t.Run()
	os.Exit(exitCode)
}
