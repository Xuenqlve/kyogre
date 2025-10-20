package mysql

import (
	"fmt"
	"strings"

	"github.com/xuenqlve/kyogre/internal/common/errors"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/timburr/pkg/tool/relational_database/mysql_schema"
	"github.com/xuenqlve/timburr/pkg/tool/sql_tool"
)

func GenerateDeleteSQL(msgBatch []message.Row, tableDef *mysql_schema.Table) (string, []any) {
	var deleteInFlag bool
	var guideKey string
	var sql string
	batchStatement := []string{}
	batchArgs := []any{}

	if len(msgBatch) == 0 {
		return "", nil
	}

	guideKeys := msgBatch[0].GuideKeys
	if len(guideKeys) == 1 {
		deleteInFlag = true
		for key := range guideKeys {
			guideKey = sql_tool.ColumnName(key)
		}
	}
	for _, msgContent := range msgBatch {
		gKeys := msgContent.GuideKeys
		whereStatement := make([]string, 0, len(gKeys))
		args := make([]any, 0, len(gKeys))
		for key, value := range gKeys {
			col := sql_tool.ColumnName(key)
			if deleteInFlag {
				whereStatement = append(whereStatement, "?")
			} else {
				whereStatement = append(whereStatement, fmt.Sprintf("%s = ?", col))
			}
			args = append(args, value)
		}

		if deleteInFlag {
			batchStatement = append(batchStatement, whereStatement...)
		} else {
			batchStatement = append(batchStatement, fmt.Sprintf("(%s)", strings.Join(whereStatement, " AND ")))
		}
		batchArgs = append(batchArgs, args...)

	}

	if deleteInFlag {
		sql = fmt.Sprintf("DELETE FROM `%s`.`%s` WHERE %s in (%s) ", tableDef.Database, tableDef.Table, guideKey, strings.Join(batchStatement, ","))
	} else {
		sql = fmt.Sprintf("DELETE FROM `%s`.`%s` WHERE %s ", tableDef.Database, tableDef.Table, strings.Join(batchStatement, " OR "))
	}

	return sql, batchArgs
}

func GenerateInsertSQL(msgBatch []message.Row, tableDef *mysql_schema.Table) (string, []any, error) {
	batchPlaceHolders, args, err := placeHoldersAndArgsFromEncodedData(msgBatch, tableDef, false)
	if err != nil {
		return "", []any{}, errors.Trace(err)
	}
	return fmt.Sprintf("%s %s", insertSqlPrefix(tableDef), strings.Join(batchPlaceHolders, ",")),
		args, nil
}

func GenerateUpdateSQL(msgBatch []message.Row, tableDef *mysql_schema.Table, isSection bool) (string, []any, error) {
	statements := make([]string, 0, len(msgBatch))
	args := make([]any, 0, len(msgBatch)*len(tableDef.Columns))
	for _, msg := range msgBatch {
		setInfo, conditionInfo, tmpArgs, err := updateSetInfoAndArgsFromEncodedData(msg, tableDef, isSection)
		if err != nil {
			return "", []any{}, errors.Trace(err)
		}
		prefix, err := updateSqlPrefix(tableDef, false)
		if err != nil {
			return "", []any{}, errors.Trace(err)
		}
		statements = append(statements, fmt.Sprintf("%s SET %s WHERE %s", prefix, strings.Join(setInfo, ","), strings.Join(conditionInfo, " AND ")))
		args = append(args, tmpArgs...)
	}
	return strings.Join(statements, ";"), args, nil
}

func GenerateInsertIgnoreSQL(msgBatch []message.Row, tableDef *mysql_schema.Table) (string, []any, error) {
	batchPlaceHolders, args, err := placeHoldersAndArgsFromEncodedData(msgBatch, tableDef, false)
	if err != nil {
		return "", []any{}, errors.Trace(err)
	}

	finalPlaceHolders := strings.Join(batchPlaceHolders, ",")
	s := []string{insertIgnoreSqlPrefix(tableDef), finalPlaceHolders}
	return strings.Join(s, " "), args, nil
}

func GenerateInsertOnDuplicateKeyUpdateSQL(msgBatch []message.Row, tableDef *mysql_schema.Table) (string, []any, error) {
	batchPlaceHolders, args, err := placeHoldersAndArgsFromEncodedData(msgBatch, tableDef, false)
	if err != nil {
		return "", nil, errors.Trace(err)
	}
	return fmt.Sprintf("%s %s %s", insertSqlPrefix(tableDef), strings.Join(batchPlaceHolders, ","), onDuplicateKeyUpdateSQLSuffix(tableDef)),
		args, nil
}

func GenerateInsertUpdateSectionSQL(msgBatch []message.Row, tableDef *mysql_schema.Table) (string, []any, error) {
	msgData := msgBatch[0].Data
	batchPlaceHolders, args, err := placeHoldersAndArgsFromEncodedData(msgBatch, tableDef, true)
	if err != nil {
		return "", nil, errors.Trace(err)
	}
	return fmt.Sprintf("%s %s %s", insertSqlPrefixByMessage(tableDef, msgData), strings.Join(batchPlaceHolders, ","), onDuplicateKeyUpdateSQLSuffixByMessage(tableDef, msgData)),
		args, nil
}

func GenerateReplaceSQL(msgBatch []message.Row, tableDef *mysql_schema.Table) (string, []any, error) {
	batchPlaceHolders, args, err := placeHoldersAndArgsFromEncodedData(msgBatch, tableDef, false)
	if err != nil {
		return "", nil, errors.Trace(err)
	}
	finalPlaceHolders := strings.Join(batchPlaceHolders, ",")
	s := []string{replaceSqlPrefix(tableDef), finalPlaceHolders}
	return strings.Join(s, " "), args, nil
}

func replaceSqlPrefix(tableDef *mysql_schema.Table) string {
	columnNames := make([]string, 0, len(tableDef.Columns))
	for _, column := range tableDef.Columns {
		columnName := column.Name
		columnNames = append(columnNames, fmt.Sprintf("`%s`", columnName))
	}
	return fmt.Sprintf("REPLACE INTO `%s`.`%s` (%s) VALUES", tableDef.Database, tableDef.Table, strings.Join(columnNames, ","))
}

func insertSqlPrefix(tableDef *mysql_schema.Table) string {
	columnNames := make([]string, 0, len(tableDef.Columns))
	for _, column := range tableDef.Columns {
		columnName := column.Name
		columnNames = append(columnNames, fmt.Sprintf("`%s`", columnName))
	}
	return fmt.Sprintf("INSERT INTO `%s`.`%s` (%s) VALUES", tableDef.Database, tableDef.Table, strings.Join(columnNames, ","))
}

func insertIgnoreSqlPrefix(tableDef *mysql_schema.Table) string {
	columnNames := make([]string, 0, len(tableDef.Columns))
	for _, column := range tableDef.Columns {
		columnName := column.Name
		columnNames = append(columnNames, fmt.Sprintf("`%s`", columnName))
	}
	return fmt.Sprintf("INSERT IGNORE INTO `%s`.`%s` (%s) VALUES", tableDef.Database, tableDef.Table, strings.Join(columnNames, ","))
}

func onDuplicateKeyUpdateSQLSuffix(tableDef *mysql_schema.Table) string {
	columnNamesAssign := make([]string, 0, len(tableDef.Columns))
	if len(tableDef.UniqueIndex) == 0 {
		return ""
	}
	for _, column := range tableDef.Columns {
		if column.IsGenerated {
			continue
		}
		columnName := column.Name
		columnNameInSQL := fmt.Sprintf("`%s`", columnName)
		columnNamesAssign = append(columnNamesAssign, fmt.Sprintf("%s = VALUES(%s)", columnNameInSQL, columnNameInSQL))
	}
	return fmt.Sprintf("ON DUPLICATE KEY UPDATE %s", strings.Join(columnNamesAssign, ","))
}

func insertSqlPrefixByMessage(tableDef *mysql_schema.Table, msgData map[string]any) string {
	columnNames := make([]string, 0, len(tableDef.Columns))
	for _, column := range tableDef.Columns {
		columnName := column.Name
		if _, ok := msgData[columnName]; !ok {
			continue
		}
		columnNames = append(columnNames, fmt.Sprintf("`%s`", columnName))
	}
	return fmt.Sprintf("INSERT INTO `%s`.`%s` (%s) VALUES", tableDef.Database, tableDef.Table, strings.Join(columnNames, ","))
}

func onDuplicateKeyUpdateSQLSuffixByMessage(tableDef *mysql_schema.Table, msgData map[string]any) string {
	columnNamesAssign := make([]string, 0, len(tableDef.Columns))

	if len(tableDef.UniqueIndex) == 0 {
		return ""
	}
	for _, column := range tableDef.Columns {
		columnName := column.Name
		if _, ok := msgData[columnName]; !ok {
			continue
		}
		if column.IsGenerated {
			continue
		}
		columnNameInSQL := fmt.Sprintf("`%s`", columnName)
		columnNamesAssign = append(columnNamesAssign, fmt.Sprintf("%s = VALUES(%s)", columnNameInSQL, columnNameInSQL))
	}
	return fmt.Sprintf("ON DUPLICATE KEY UPDATE %s", strings.Join(columnNamesAssign, ","))
}

func placeHoldersAndArgsFromEncodedData(msgBatch []message.Row, tableDef *mysql_schema.Table, bySource bool) ([]string, []any, error) {
	var batchPlaceHolders []string
	var batchArgs []any
	msgLen := len(msgBatch[0].Data)

	for _, msg := range msgBatch {
		if msg.Data == nil {
			return nil, nil, errors.Errorf("Data and MysqlRawBytes are null")
		}

		singleSqlPlaceHolders, singleSqlArgs, err := getSingleSqlPlaceHolderAndArgWithEncodedData(msg, tableDef, bySource)
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
		batchPlaceHolders = append(batchPlaceHolders, singleSqlPlaceHolders)
		if len(singleSqlArgs) != msgLen {
			return nil, nil, fmt.Errorf("single sql args does not match message length")
		}
		batchArgs = append(batchArgs, singleSqlArgs...)
	}
	return batchPlaceHolders, batchArgs, nil
}

func getSingleSqlPlaceHolderAndArgWithEncodedData(msg message.Row, tableDef *mysql_schema.Table, bySource bool) (string, []any, error) {
	if err := validateSchema(msg, tableDef); err != nil && !bySource {
		return "", nil, errors.Trace(err)
	}
	data := msg.Data
	var placeHolders []string
	var args []any

	for _, column := range tableDef.Columns {
		columnName := column.Name
		columnData, ok := data[columnName]
		if !ok {
			if bySource {
				continue
			}
			return "", nil, errors.Errorf("db:%s, table:%s, column:%s missing data", tableDef.Database, tableDef.Table, columnName)
		}
		if column.IsGenerated && msg.IsColumnSetDefault(columnName) {
			placeHolders = append(placeHolders, "DEFAULT")
			continue
		}
		args = append(args, adjustArgs(columnData, &column))
		placeHolders = append(placeHolders, "?")

	}
	singleSqlPlaceHolder := fmt.Sprintf("(%s)", strings.Join(placeHolders, ","))
	return singleSqlPlaceHolder, args, nil
}

func validateSchema(msg message.Row, tableDef *mysql_schema.Table) error {
	columnLenInMsg := len(msg.Data)
	columnLenInTarget := len(tableDef.Columns)

	if columnLenInMsg != columnLenInTarget {
		return errors.Errorf("%s.%s: columnLenInMsg %d columnLenInTarget %d not equal", tableDef.Database, tableDef.Table, columnLenInMsg, columnLenInTarget)
	}
	return nil
}

func adjustArgs(arg any, column *mysql_schema.Column) any {
	if arg == nil {
		return arg
	}
	if column.Type == mysql_schema.TypeDatetime || column.Type == mysql_schema.TypeTimestamp || column.Type == mysql_schema.TypeDate { // datetime is in utc and should ignore location
		v, flag := mysql_schema.ParseTime(arg, column.Type)
		if flag {
			return v.Format("2006-01-02 15:04:05.999999999")
		}
	}
	return arg
}

func updateSetInfoAndArgsFromEncodedData(msgBatch message.Row, tableDef *mysql_schema.Table, isSection bool) ([]string, []string, []any, error) {
	if msgBatch.Data == nil || msgBatch.GuideKeys == nil {
		return nil, nil, nil, fmt.Errorf("data or guideKeys are nil")
	}
	setInfo := make([]string, 0, len(tableDef.Columns))
	conditionInfo := make([]string, 0, len(msgBatch.GuideKeys))
	args := make([]any, 0, len(tableDef.Columns)+len(msgBatch.GuideKeys))
	conditionArgs := make([]any, 0, len(msgBatch.GuideKeys))
	data := msgBatch.Data
	guideKeys := msgBatch.GuideKeys
	oldData := msgBatch.Old

	for _, column := range tableDef.Columns {
		columnName := column.Name
		_, ok := guideKeys[columnName]
		if ok {
			if keyData, exist := oldData[columnName]; exist {
				singleConditionInfo := fmt.Sprintf("`%s` = ?", columnName)
				conditionInfo = append(conditionInfo, singleConditionInfo)
				conditionArgs = append(conditionArgs, adjustArgs(keyData, &column))
			} else {
				return nil, nil, nil, fmt.Errorf("old data not found by key column: %s", columnName)
			}

		}
		columnData, ok := data[columnName]
		if !ok {
			if isSection {
				continue
			}
			return nil, nil, nil, errors.Errorf("db:%s, table:%s, column:%s missing data", tableDef.Database, tableDef.Table, columnName)
		}
		args = append(args, adjustArgs(columnData, &column))
		singleSetInfo := fmt.Sprintf("`%s` = ?", columnName)
		setInfo = append(setInfo, singleSetInfo)
	}
	for _, conditionArg := range conditionArgs {
		args = append(args, conditionArg)
	}
	return setInfo, conditionInfo, args, nil
}

func updateSqlPrefix(tableDef *mysql_schema.Table, updateIgnore bool) (string, error) {
	if updateIgnore {
		return fmt.Sprintf("UPDATE IGNORE `%s`.`%s`", tableDef.Database, tableDef.Table), nil
	} else {
		return fmt.Sprintf("UPDATE `%s`.`%s` ", tableDef.Database, tableDef.Table), nil
	}
}
