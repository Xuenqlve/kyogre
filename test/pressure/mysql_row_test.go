package pressure

import (
	"context"
	"testing"

	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin"
	"github.com/xuenqlve/kyogre/pkg/pressure/mysql-row"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
)

func MockInsertMySQLRowMessage(writeType string) *message.MySQLRowMessage {
	return &message.MySQLRowMessage{
		SQLRows: message.SQLRows{
			Metadata: message.Metadata{
				Database:  "test",
				Table:     "test",
				Operation: schema_store.Insert.String(),
				WriteType: writeType,
			},
			Contents: []mysql_schema.RowData{
				{
					Key: "1001",
					Data: map[string]interface{}{
						"id":   "1001",
						"name": "name_1001",
					},
				}, {
					Key: "1002",
					Data: map[string]interface{}{
						"id":   "1002",
						"name": "name_1002",
					},
				},
			},
		},
	}
}

func MockUpdateMySQLRowMessage(writeType string) *message.MySQLRowMessage {
	return &message.MySQLRowMessage{
		SQLRows: message.SQLRows{
			Metadata: message.Metadata{
				Database:  "test",
				Table:     "test",
				Operation: schema_store.Update.String(),
				WriteType: writeType,
			},
			Contents: []mysql_schema.RowData{
				{
					Key: "1001",
					Old: map[string]any{
						"id":   "1001",
						"name": "name_1001",
					},
					Data: map[string]any{
						"id":   "1001",
						"name": "name_1001_new",
					},
					GuideKeys: map[string]any{
						"id": "1001",
					},
				}, {
					Key: "1002",
					Old: map[string]any{
						"id":   "1002",
						"name": "name_1002",
					},
					Data: map[string]any{
						"id":   "1002",
						"name": "name_1002_new",
					},
					GuideKeys: map[string]any{
						"id": "1002",
					},
				},
			},
		},
	}
}

func MockDeleteMySQLRowMessage(writeType string) *message.MySQLRowMessage {
	return &message.MySQLRowMessage{
		SQLRows: message.SQLRows{
			Metadata: message.Metadata{
				Database:  "test",
				Table:     "test",
				Operation: schema_store.Delete.String(),
				WriteType: writeType,
			},
			Contents: []mysql_schema.RowData{
				{
					Key: "1001",
					GuideKeys: map[string]any{
						"id": "1001",
					},
				}, {
					Key: "1002",
					GuideKeys: map[string]any{
						"id": "1002",
					},
				},
			},
		},
	}
}

func TestMySQL(t *testing.T) {
	ctx := context.Background()
	pressure, err := plugin.GetPressure(mysql_row.MySQLDML)
	if err != nil {
		t.Fatalf("GetPressure() failed: %v", err)
		return
	}
	err = pressure.Configure(pipelineName, map[string]any{
		"data-source": mysqlDataSource,
	})
	if err != nil {
		t.Fatalf("Configure() failed: %v", err)
		return
	}
	if err = pressure.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
		return
	}

	t.Run("insert", func(t *testing.T) {
		msg := MockInsertMySQLRowMessage("insert")
		pressure.Execute(msg)
		if err = pressure.Close(); err != nil {
			t.Errorf("pressure.Close() failed: %v", err)
		}
		t.Log("pressure closed")
	})

	t.Run("update", func(t *testing.T) {
		msg := MockUpdateMySQLRowMessage("update")
		pressure.Execute(msg)
		if err = pressure.Close(); err != nil {
			t.Errorf("pressure.Close() failed: %v", err)
		}
		t.Log("pressure closed")
	})

	t.Run("delete", func(t *testing.T) {
		msg := MockDeleteMySQLRowMessage("delete")
		pressure.Execute(msg)
		if err = pressure.Close(); err != nil {
			t.Errorf("pressure.Close() failed: %v", err)
		}
		t.Log("pressure closed")
	})

}
