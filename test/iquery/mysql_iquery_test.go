package iquery

import (
	"context"
	"testing"

	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
	"github.com/xuenqlve/kyogre/pkg/lookup"
	"github.com/xuenqlve/kyogre/pkg/metadata/mysql"
	//"github.com/xuenqlve/kyogre/pkg/metadata_template"
)

func templateMySQLMetadata(template, datasource string) (md metadata.Metadata, err error) {
	if md, err = metadata.GetMetadata(mysql.MySQL); err != nil {
		return
	}
	cfg := map[string]interface{}{
		"data-source": datasource,
		"template":    template,
	}
	if err = md.Configure(pipeline, cfg); err != nil {
		return
	}
	if err = md.Initialize(context.Background()); err != nil {
		return
	}
	return md, nil
}

func TestMySQLIQuery(t *testing.T) {
	lk, err := iquery.GetIQueryModule(lookup.MySQL)
	if err != nil {
		t.Error(err)
		return
	}
	defer lk.Close()

}
