package mysql

import (
	"fmt"
	"strings"

	"github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/common/schema_store"
	genctx "github.com/xuenqlve/kyogre/pkg/generator_context"
)

func buildDDLStatement(spec genctx.MySQLDDLSpec) (mysql.DDLStatement, error) {
	if spec.Schema == nil {
		return mysql.DDLStatement{}, fmt.Errorf("ddl schema is nil")
	}
	ddlType := schema_store.DDLMap[strings.ToUpper(spec.DDLType)]
	if ddlType == "" || ddlType == schema_store.UNKNOWN {
		return mysql.DDLStatement{}, fmt.Errorf("unsupported ddl type: %s", spec.DDLType)
	}
	sql, err := buildDDLSQL(spec, ddlType)
	if err != nil {
		return mysql.DDLStatement{}, err
	}
	loader := mysql.NewDDLLoader()
	list, err := loader.Parse(spec.Schema.Database, sql)
	if err != nil {
		return mysql.DDLStatement{}, err
	}
	if len(list) == 0 {
		return mysql.DDLStatement{}, fmt.Errorf("ddl parse result is empty")
	}
	return list[0], nil
}

func buildDDLSQL(spec genctx.MySQLDDLSpec, ddlType schema_store.DDL) (string, error) {
	db := spec.Schema.Database
	table := spec.Schema.Table
	switch ddlType {
	case schema_store.CREATE_DATABASE:
		return fmt.Sprintf("CREATE DATABASE `%s`", db), nil
	case schema_store.DROP_DATABASE:
		return fmt.Sprintf("DROP DATABASE `%s`", db), nil
	case schema_store.CREATE_TABLE:
		if sql := strings.TrimSpace(spec.Schema.CreateTableSql()); sql != "" {
			return sql, nil
		}
		cols, err := ddlColumnsForCreate(spec)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("CREATE TABLE `%s`.`%s` (%s)", db, table, strings.Join(cols, ",")), nil
	case schema_store.ALTER_TABLE:
		clauses, err := ddlAlterClauses(spec)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("ALTER TABLE `%s`.`%s` %s", db, table, strings.Join(clauses, ",")), nil
	case schema_store.DROP_TABLE:
		return fmt.Sprintf("DROP TABLE `%s`.`%s`", db, table), nil
	case schema_store.RENAME_TABLE:
		newDB, newTable, err := ddlRenameTarget(spec)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("RENAME TABLE `%s`.`%s` TO `%s`.`%s`", db, table, newDB, newTable), nil
	case schema_store.TRUNCATE_TABLE:
		return fmt.Sprintf("TRUNCATE TABLE `%s`.`%s`", db, table), nil
	case schema_store.CREATE_INDEX:
		indexName, columns, err := ddlIndexSpec(spec)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("CREATE INDEX `%s` ON `%s`.`%s` (%s)", indexName, db, table, strings.Join(columns, ",")), nil
	case schema_store.DROP_INDEX:
		indexName, _, err := ddlIndexSpec(spec)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("DROP INDEX `%s` ON `%s`.`%s`", indexName, db, table), nil
	default:
		return "", fmt.Errorf("unsupported ddl type: %s", spec.DDLType)
	}
}

func ddlColumnsForCreate(spec genctx.MySQLDDLSpec) ([]string, error) {
	if len(spec.Columns) > 0 {
		cols := make([]string, 0, len(spec.Columns))
		for _, col := range spec.Columns {
			if col.Name == "" || col.Type == "" {
				return nil, fmt.Errorf("create table column name/type is empty")
			}
			cols = append(cols, fmt.Sprintf("`%s` %s", col.Name, col.Type))
		}
		return cols, nil
	}
	if len(spec.Schema.Columns) == 0 {
		return nil, fmt.Errorf("create table schema columns are empty")
	}
	cols := make([]string, 0, len(spec.Schema.Columns))
	for _, col := range spec.Schema.Columns {
		colType := col.RawType
		if colType == "" {
			colType = col.DataType
		}
		if colType == "" {
			colType = "varchar(255)"
		}
		cols = append(cols, fmt.Sprintf("`%s` %s", col.Name, colType))
	}
	return cols, nil
}

func ddlAlterClauses(spec genctx.MySQLDDLSpec) ([]string, error) {
	if len(spec.Columns) == 0 {
		return nil, fmt.Errorf("alter table columns are empty")
	}
	clauses := make([]string, 0, len(spec.Columns))
	for _, col := range spec.Columns {
		switch strings.ToLower(col.Change) {
		case "add":
			if col.Name == "" || col.Type == "" {
				return nil, fmt.Errorf("alter add column name/type is empty")
			}
			clauses = append(clauses, fmt.Sprintf("ADD COLUMN `%s` %s", col.Name, col.Type))
		case "drop":
			if col.Name == "" {
				return nil, fmt.Errorf("alter drop column name is empty")
			}
			clauses = append(clauses, fmt.Sprintf("DROP COLUMN `%s`", col.Name))
		case "modify":
			if col.Name == "" || col.Type == "" {
				return nil, fmt.Errorf("alter modify column name/type is empty")
			}
			clauses = append(clauses, fmt.Sprintf("MODIFY COLUMN `%s` %s", col.Name, col.Type))
		case "rename":
			if col.Name == "" || col.Type == "" {
				return nil, fmt.Errorf("alter rename column old/new name is empty")
			}
			clauses = append(clauses, fmt.Sprintf("RENAME COLUMN `%s` TO `%s`", col.Name, col.Type))
		case "":
			if col.Name == "" || col.Type == "" {
				return nil, fmt.Errorf("alter column name/type is empty")
			}
			clauses = append(clauses, fmt.Sprintf("ADD COLUMN `%s` %s", col.Name, col.Type))
		default:
			return nil, fmt.Errorf("unsupported alter change: %s", col.Change)
		}
	}
	return clauses, nil
}

func ddlRenameTarget(spec genctx.MySQLDDLSpec) (string, string, error) {
	if len(spec.Columns) == 0 || spec.Columns[0].Name == "" {
		return "", "", fmt.Errorf("rename table target is empty")
	}
	newDB := spec.Schema.Database
	newTable := spec.Columns[0].Name
	if spec.Columns[0].Type != "" {
		newDB = spec.Columns[0].Type
	}
	return newDB, newTable, nil
}

func ddlIndexSpec(spec genctx.MySQLDDLSpec) (string, []string, error) {
	if len(spec.Columns) == 0 || spec.Columns[0].Name == "" {
		return "", nil, fmt.Errorf("index name is empty")
	}
	indexName := spec.Columns[0].Name
	if len(spec.Columns) < 2 {
		return indexName, nil, fmt.Errorf("index columns are empty")
	}
	cols := make([]string, 0, len(spec.Columns)-1)
	for _, col := range spec.Columns[1:] {
		if col.Name == "" {
			return "", nil, fmt.Errorf("index column name is empty")
		}
		cols = append(cols, fmt.Sprintf("`%s`", col.Name))
	}
	return indexName, cols, nil
}
