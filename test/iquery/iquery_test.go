package iquery

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/test/test_case"
)

const (
	pipeline        = "test_pressure"
	mysqlDataSource = "pressure"
)

func mysqlDataSourcePrepare() error {
	return test_case.PrepareMySQLDataSource(pipeline, mysqlDataSource)
}

func TestMain(t *testing.M) {
	dir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("get work dir fail err:%v", err))
	}
	parentDir := filepath.Dir(dir)
	log.Init(log.DebugLevel, parentDir)
	if err = mysqlDataSourcePrepare(); err != nil {
		panic(fmt.Sprintf("mysqlDataSourcePrepare err:%v", err))
	}
	exitCode := t.Run()
	os.Exit(exitCode)
}
