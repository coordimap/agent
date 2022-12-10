package postgres

import "strings"

func cleanupSchemaName(tableName string) string {
	splitTableName := strings.Split(tableName, ".")

	if len(splitTableName) == 1 {
		return splitTableName[0]
	}

	return splitTableName[1]
}
