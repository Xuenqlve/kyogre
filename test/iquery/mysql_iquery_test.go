package iquery

import (
	"context"
	"testing"

	iquery2 "github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
	"github.com/xuenqlve/kyogre/pkg/iquery"
	"github.com/xuenqlve/kyogre/pkg/metadata/mysql"
	"github.com/xuenqlve/kyogre/pkg/metadata_template"
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
	module, err := iquery2.GetIQueryModule(iquery.MySQL)
	if err != nil {
		t.Error(err)
		return
	}
	ctx := context.Background()
	md, err := templateMySQLMetadata(metadata_template.EcommerceTemplate, mysql.MockDataSource)
	if err != nil {
		t.Error(err)
		return
	}
	keys := md.SchemaKeys()
	t.Run("BatchQuery", func(t *testing.T) {
		result, err := module.BatchQuery(ctx, keys)
		if err != nil {
			t.Error(err)
			return
		}
		for _, info := range result {
			t.Logf("%+v", info)
		}
	})

	t.Run("QueryMaxValue", func(t *testing.T) {
		module.QueryMaxValue()
	})
	t.Run("QueryRowCount", func(t *testing.T) {
		module.QueryRowCount()
	})
	t.Run("GetQueryResult", func(t *testing.T) {
		module.GetQueryResult()
	})
}
