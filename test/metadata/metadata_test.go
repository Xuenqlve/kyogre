package metadata

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xuenqlve/common/log"
	_ "github.com/xuenqlve/kyogre/pkg/metadata_template"
	"github.com/xuenqlve/kyogre/test/test_case"
)

const (
	pipeline        = "metadata-test"
	mysqlDataSource = "source"
)

func mysqlDataSourcePrepare() error {
	return test_case.PrepareMySQLDataSource(pipeline, mysqlDataSource)
}

func TestMain(t *testing.M) {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	parentDir := filepath.Dir(dir)
	log.Init(log.DebugLevel, parentDir)
	err = mysqlDataSourcePrepare()
	if err != nil {
		panic(err)
	}
	exitCode := t.Run()
	os.Exit(exitCode)
}
