package tool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xuenqlve/common/log"
)

func TestMain(t *testing.M) {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	parentDir := filepath.Dir(dir)
	log.Init(log.DebugLevel, parentDir)
	exitCode := t.Run()
	os.Exit(exitCode)
}
