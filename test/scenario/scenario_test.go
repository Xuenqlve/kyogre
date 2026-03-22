package scenario

import (
	"fmt"
	"os"
	"testing"

	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/test/test_case"
)

const (
	pipelineName    = "test_scenario"
	mysqlDataSource = "pressure"
)

func mysqlDataSourcePrepare() error {
	return test_case.PrepareMySQLDataSource(pipelineName, mysqlDataSource)
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
