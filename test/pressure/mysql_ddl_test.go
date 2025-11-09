package pressure

import (
	"context"
	"testing"
	"time"

	"github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin"
	gh_ost "github.com/xuenqlve/kyogre/pkg/pressure/mysql-ddl"
)

var ddlLoader = mysql.NewDDLLoader()

var database = "kyogre"

func MockCreateDatabaseMessage() *message.MySQLDDLMessage {
	sql := "CREATE DATABASE IF NOT EXISTS " + database + ";"
	parse, err := ddlLoader.Parse(database, sql)
	if err != nil {
		panic(err)
	}
	stmt := parse[0]
	return &message.MySQLDDLMessage{
		Metadata: message.Metadata{
			Operation: schema_store.CREATE_DATABASE.String(),
			Database:  database,
			StartTime: time.Now(),
		},
		DDLStatement: stmt,
	}
}

func MockDropDatabaseDDLMessage() *message.MySQLDDLMessage {
	sql := "drop database if exists " + database + ";"
	parse, err := ddlLoader.Parse(database, sql)
	if err != nil {
		panic(err)
	}
	stmt := parse[0]
	return &message.MySQLDDLMessage{
		Metadata: message.Metadata{
			Operation: schema_store.DROP_DATABASE.String(),
			Database:  database,
			StartTime: time.Now(),
		},
		DDLStatement: stmt,
	}
}

func MockCreateDDLMessage() *message.MySQLDDLMessage {
	sql := "CREATE TABLE `test` (\n  `id` bigint(11) unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',\n  `name` varchar(245) NOT NULL DEFAULT '' COMMENT 'name',\n  PRIMARY KEY (`id`)\n) ENGINE=InnoDB AUTO_INCREMENT=1003 DEFAULT CHARSET=utf8 COMMENT='单一主键测试表'"
	parse, err := ddlLoader.Parse(database, sql)
	if err != nil {
		panic(err)
	}
	stmt := parse[0]
	return &message.MySQLDDLMessage{
		Metadata: message.Metadata{
			Operation: schema_store.Create_Table.String(),
			Database:  database,
			Table:     "test",
			StartTime: time.Now(),
		},
		DDLStatement: stmt,
	}
}

func MockDropTableDDLMessage() *message.MySQLDDLMessage {
	sql := "DROP TABLE IF EXISTS `test`;"
	parse, err := ddlLoader.Parse(database, sql)
	if err != nil {
		panic(err)
	}
	stmt := parse[0]
	return &message.MySQLDDLMessage{
		Metadata: message.Metadata{
			Operation: schema_store.DROP_TABLE.String(),
			Database:  database,
			Table:     "test",
			StartTime: time.Now(),
		},
		DDLStatement: stmt,
	}
}

func MockRenameTableDDLMessage(table, newTable string) *message.MySQLDDLMessage {
	sql := "RENAME TABLE `" + table + "` TO `" + newTable + "`;"
	parse, err := ddlLoader.Parse(database, sql)
	if err != nil {
		panic(err)
	}
	stmt := parse[0]
	return &message.MySQLDDLMessage{
		Metadata: message.Metadata{
			Operation: schema_store.RENAME_TABLE.String(),
			Database:  database,
			Table:     table,
			StartTime: time.Now(),
		},
		DDLStatement: stmt,
	}
}

func MockAlterRenameTableDDLMessage(table, newTable string) *message.MySQLDDLMessage {
	sql := "alter table `" + table + "` rename `" + newTable + "`;"
	parse, err := ddlLoader.Parse(database, sql)
	if err != nil {
		panic(err)
	}
	stmt := parse[0]
	return &message.MySQLDDLMessage{
		Metadata: message.Metadata{
			Operation: schema_store.ALTER_TABLE.String(),
			Database:  database,
			Table:     table,
			StartTime: time.Now(),
		},
		DDLStatement: stmt,
	}
}

func MockAlterAddColumnDDLMessage() *message.MySQLDDLMessage {
	sql := "alter table test add column `a` smallint unsigned NOT NULL DEFAULT '0' COMMENT 'a'"
	parse, err := ddlLoader.Parse(database, sql)
	if err != nil {
		panic(err)
	}
	stmt := parse[0]
	return &message.MySQLDDLMessage{
		Metadata: message.Metadata{
			Operation: schema_store.ALTER_TABLE.String(),
			Database:  database,
			Table:     "test",
			StartTime: time.Now(),
		},
		DDLStatement: stmt,
	}
}

func MockAlterChangeColumnDDLMessage() *message.MySQLDDLMessage {
	sql := "alter table test change column `a` `a` int(10) NOT NULL DEFAULT '0' COMMENT 'a' "
	parse, err := ddlLoader.Parse(database, sql)
	if err != nil {
		panic(err)
	}
	stmt := parse[0]
	return &message.MySQLDDLMessage{
		Metadata: message.Metadata{
			Operation: schema_store.ALTER_TABLE.String(),
			Database:  database,
			Table:     "test",
			StartTime: time.Now(),
		},
		DDLStatement: stmt,
	}
}

func MockAlterModifyColumnDDLMessage() *message.MySQLDDLMessage {
	sql := "alter table test modify column `a` smallint NOT NULL DEFAULT '1' COMMENT 'a'"
	parse, err := ddlLoader.Parse(database, sql)
	if err != nil {
		panic(err)
	}
	stmt := parse[0]
	return &message.MySQLDDLMessage{
		Metadata: message.Metadata{
			Operation: schema_store.ALTER_TABLE.String(),
			Database:  database,
			Table:     "test",
			StartTime: time.Now(),
		},
		DDLStatement: stmt,
	}

}

func MockAlterAddIndexDDLMessage() *message.MySQLDDLMessage {
	sql := "alter table test add index idx_a(`a`)"
	parse, err := ddlLoader.Parse(database, sql)
	if err != nil {
		panic(err)
	}
	stmt := parse[0]
	return &message.MySQLDDLMessage{
		Metadata: message.Metadata{
			Operation: schema_store.ALTER_TABLE.String(),
			Database:  database,
			Table:     "test",
			StartTime: time.Now(),
		},
		DDLStatement: stmt,
	}
}

func MockAddIndexDDLMessage() *message.MySQLDDLMessage {
	sql := "create index idx_a1 on test(`a`)"
	parse, err := ddlLoader.Parse(database, sql)
	if err != nil {
		panic(err)
	}
	stmt := parse[0]
	return &message.MySQLDDLMessage{
		Metadata: message.Metadata{
			Operation: schema_store.Create_Indexes.String(),
			Database:  database,
			Table:     "test",
			StartTime: time.Now(),
		},
		DDLStatement: stmt,
	}
}

func MockDropIndexDDLMessage() *message.MySQLDDLMessage {
	sql := "DROP INDEX idx_a1 on test;"
	parse, err := ddlLoader.Parse(database, sql)
	if err != nil {
		panic(err)
	}
	stmt := parse[0]
	return &message.MySQLDDLMessage{
		Metadata: message.Metadata{
			Operation: schema_store.DROP_INDEX.String(),
			Database:  database,
			Table:     "test",
			StartTime: time.Now(),
		},
		DDLStatement: stmt,
	}
}

func MockAlterDropColumnDDLMessage() *message.MySQLDDLMessage {
	sql := "alter table test drop column `a`"
	parse, err := ddlLoader.Parse(database, sql)
	if err != nil {
		panic(err)
	}
	stmt := parse[0]
	return &message.MySQLDDLMessage{
		Metadata: message.Metadata{
			Operation: schema_store.DROP_TABLE.String(),
			Database:  database,
			Table:     "test",
			StartTime: time.Now(),
		},
		DDLStatement: stmt,
	}
}

func TestGHost(t *testing.T) {
	ctx := context.Background()
	pressure, err := plugin.GetPressure(gh_ost.MySQLDDL)
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

	t.Run("ddl create database", func(t *testing.T) {
		msg := MockCreateDatabaseMessage()
		pressure.Execute(msg)
		if err = pressure.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
		t.Logf("DDL Message: %v", msg)
	})

	t.Run("ddl drop database", func(t *testing.T) {
		msg := MockDropDatabaseDDLMessage()
		pressure.Execute(msg)
		if err = pressure.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
		t.Logf("DDL Message: %v", msg)
	})

	t.Run("ddl create table", func(t *testing.T) {
		msg := MockCreateDatabaseMessage()
		pressure.Execute(msg)
		msg1 := MockCreateDDLMessage()
		pressure.Execute(msg1)
		if err = pressure.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
		t.Logf("DDL Message: %v", msg)
	})

	t.Run("ddl drop table", func(t *testing.T) {
		msg := MockDropTableDDLMessage()
		pressure.Execute(msg)
		if err = pressure.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
		t.Logf("DDL Message: %v", msg)
	})

	t.Run("rename table", func(t *testing.T) {
		msg := MockCreateDDLMessage()
		pressure.Execute(msg)
		msg2 := MockRenameTableDDLMessage("test", "test1")
		pressure.Execute(msg2)
		if err = pressure.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
	})

	t.Run("rename table2", func(t *testing.T) {
		msg := MockCreateDDLMessage()
		pressure.Execute(msg)
		msg2 := MockAlterRenameTableDDLMessage("test", "test1")
		pressure.Execute(msg2)
		if err = pressure.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
	})

	t.Run("alter add column", func(t *testing.T) {
		msg := MockCreateDDLMessage()
		pressure.Execute(msg)
		msg2 := MockAlterAddColumnDDLMessage()
		pressure.Execute(msg2)
		if err = pressure.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
	})

	t.Run("change column", func(t *testing.T) {
		msg := MockCreateDDLMessage()
		pressure.Execute(msg)
		msg2 := MockAlterAddColumnDDLMessage()
		pressure.Execute(msg2)
		msg3 := MockAlterChangeColumnDDLMessage()
		pressure.Execute(msg3)
		if err = pressure.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
	})

	t.Run("alter modify column", func(t *testing.T) {
		msg := MockCreateDDLMessage()
		pressure.Execute(msg)
		msg2 := MockAlterAddColumnDDLMessage()
		pressure.Execute(msg2)
		msg3 := MockAlterModifyColumnDDLMessage()
		pressure.Execute(msg3)
		if err = pressure.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
	})

	t.Run("alter add index", func(t *testing.T) {
		msg := MockCreateDDLMessage()
		pressure.Execute(msg)
		msg2 := MockAlterAddColumnDDLMessage()
		pressure.Execute(msg2)
		msg3 := MockAlterModifyColumnDDLMessage()
		pressure.Execute(msg3)
		msg4 := MockAlterAddIndexDDLMessage()
		pressure.Execute(msg4)
		if err = pressure.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
	})

	t.Run("alter add index", func(t *testing.T) {
		msg := MockCreateDDLMessage()
		pressure.Execute(msg)
		msg2 := MockAlterAddColumnDDLMessage()
		t.Logf("alter add column: %v", msg)
		pressure.Execute(msg2)
		msg3 := MockAlterModifyColumnDDLMessage()
		t.Logf("alter modify column: %v", msg)
		pressure.Execute(msg3)
		msg4 := MockAddIndexDDLMessage()
		t.Logf("add index: %v", msg)
		pressure.Execute(msg4)
		if err = pressure.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
	})

	t.Run("drop index ddl", func(t *testing.T) {
		msg := MockCreateDDLMessage()
		pressure.Execute(msg)
		msg2 := MockAlterAddColumnDDLMessage()
		pressure.Execute(msg2)
		msg3 := MockAlterModifyColumnDDLMessage()
		pressure.Execute(msg3)
		msg4 := MockAddIndexDDLMessage()
		pressure.Execute(msg4)
		msg5 := MockDropIndexDDLMessage()
		pressure.Execute(msg5)
		if err = pressure.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
	})

	t.Run("drop column", func(t *testing.T) {
		msg := MockCreateDDLMessage()
		pressure.Execute(msg)
		msg2 := MockAlterAddColumnDDLMessage()
		pressure.Execute(msg2)
		msg3 := MockAlterModifyColumnDDLMessage()
		pressure.Execute(msg3)
		msg4 := MockAddIndexDDLMessage()
		pressure.Execute(msg4)
		msg5 := MockDropIndexDDLMessage()
		pressure.Execute(msg5)
		msg6 := MockAlterDropColumnDDLMessage()
		pressure.Execute(msg6)
		if err = pressure.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
	})

}
