package postgres

import (
	"fmt"
	"strings"
)

func cleanupSchemaName(tableName string) string {
	splitTableName := strings.Split(tableName, ".")

	if len(splitTableName) == 1 {
		return splitTableName[0]
	}

	return splitTableName[1]
}

func generateInternalName(host, dbName, schema, name string) string {
	return fmt.Sprintf("%s/%s-%s@%s", host, dbName, schema, name)
}
