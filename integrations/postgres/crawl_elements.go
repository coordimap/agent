package postgres

import (
	"fmt"

	databasemodels "dev.azure.com/bloopi/bloopi/_git/shared_models.git/database_models"
	post_model "dev.azure.com/bloopi/bloopi/_git/shared_models.git/postgres"
	"github.com/rs/zerolog/log"
)

func (postCrawler *postgresCrawler) getSchemaNames() ([]string, error) {
	schemaNames := []string{}
	sqlStatement := `SELECT schema_name FROM information_schema.schemata WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast', '_timescaledb_cache', '_timescaledb_catalog', '_timescaledb_internal', '_timescaledb_config', 'timescaledb_experimental', 'timescaledb_information')`
	rows, err := postCrawler.dbConn.Query(sqlStatement)
	if err != nil {
		return schemaNames, err
	}

	defer rows.Close()

	for rows.Next() {
		var schemaName string
		if err := rows.Scan(&schemaName); err != nil {
			return schemaNames, err
		}
		schemaNames = append(schemaNames, schemaName)
	}

	return schemaNames, nil
}

func (postCrawler *postgresCrawler) getSchemaTables(schemaName string) ([]string, error) {
	tableNames := []string{}
	sqlStatement := `select table_name from information_schema.tables where table_schema not in ('pg_catalog', 'information_schema') and table_schema = $1`
	rows, err := postCrawler.dbConn.Query(sqlStatement, schemaName)
	if err != nil {
		return tableNames, err
	}

	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return tableNames, err
		}
		tableNames = append(tableNames, tableName)
	}

	return tableNames, nil
}

func (postCrawler *postgresCrawler) getTableData(schemaName, tableName string) (databasemodels.Table, error) {
	table := databasemodels.Table{
		Name:        tableName,
		Columns:     []databasemodels.Column{},
		Constraints: []databasemodels.Constraint{},
		Schema:      schemaName,
		Indexes:     []string{},
	}

	columns, errColumns := postCrawler.getTableColumns(schemaName, tableName)
	if errColumns != nil {
		log.Warn().Msgf("Something happened while trying to get the columns of %s.%s due to %s", schemaName, tableName, errColumns.Error())
	}
	table.Columns = columns

	constraints, errConstraints := postCrawler.getTableConstraints(schemaName, tableName)
	if errConstraints != nil {
		log.Warn().Msgf("Something happened while trying to get the constraints of %s.%s due to %s", schemaName, tableName, errConstraints.Error())
	}
	table.Constraints = constraints

	return table, nil
}

func (postCrawler *postgresCrawler) getTableConstraints(schemaName, tableName string) ([]databasemodels.Constraint, error) {
	constraints := []databasemodels.Constraint{}
	tableNameCleaned := cleanupSchemaName(tableName)

	// Get all constraint names of table
	sqlTableConstraints := `select constraint_name from information_schema.key_column_usage where table_schema = $1 and table_name = $2`
	resTableConstraints, errTableConstraints := postCrawler.dbConn.Query(sqlTableConstraints, schemaName, tableNameCleaned)
	if errTableConstraints != nil {
		log.Error().Msgf("Could not get all the constraint names for %s.%s", schemaName, tableName)
		return constraints, errTableConstraints
	}

	defer resTableConstraints.Close()

	for resTableConstraints.Next() {
		var constraintName, constraintType string
		if err := resTableConstraints.Scan(&constraintName); err != nil {
			log.Error().Msg("Could not bind the constraint to the variable.")
			return constraints, err
		}

		constraint := databasemodels.Constraint{
			Name:         constraintName,
			Type:         "",
			Sources:      []databasemodels.Column{},
			Destinations: []databasemodels.Column{},
		}

		// Get all columns of the constraint
		sqlConstraintsColumns := `
			select
				kcu.ordinal_position as position,
				kcu.column_name as key_column,
				'postgres.' || LOWER(REPLACE(tco.constraint_type, ' ', '_')) AS constraint_type
			from information_schema.table_constraints tco
			join information_schema.key_column_usage kcu
				on kcu.constraint_name = tco.constraint_name
				and kcu.constraint_schema = tco.constraint_schema
				and kcu.constraint_name = tco.constraint_name
			where kcu.table_schema = $1 and kcu.table_name = $2 and kcu.constraint_name = $3
			order by kcu.table_schema,
					kcu.table_name,
					position
		`
		rowsConstraintsColumns, errConstraitsColumns := postCrawler.dbConn.Query(sqlConstraintsColumns, schemaName, tableNameCleaned, constraintName)
		if errConstraitsColumns != nil {
			log.Error().Msgf("Could not get columns of constraint %s.%s.%s", schemaName, tableName, constraintName)
			return constraints, errConstraitsColumns
		}

		defer rowsConstraintsColumns.Close()

		for rowsConstraintsColumns.Next() {
			var sourceConstraintCol databasemodels.Column
			if err := rowsConstraintsColumns.Scan(&sourceConstraintCol.Position, &sourceConstraintCol.Name, &constraintType); err != nil {
				continue
			}

			sourceConstraintCol.Table = generateInternalName(postCrawler.Host, postCrawler.DBName, schemaName, tableName)

			switch constraintType {
			case "PRIMARY KEY":
				constraint.Type = post_model.POSTGRES_CONSTRAINT_PK

			case "FOREIGN KEY":
				constraint.Type = post_model.POSTGRES_CONSTRAINT_FK

			case "UNIQUE":
				constraint.Type = post_model.POSTGRES_CONSTRAINT_UNIQUE

			default:
				constraint.Type = constraintType
			}

			constraint.Sources = append(constraint.Sources, sourceConstraintCol)
		}

		if constraintType != post_model.POSTGRES_CONSTRAINT_FK {
			constraints = append(constraints, constraint)
			continue
		}

		// Get all table relations for each constraints
		sqlFKConstraints := `
			select
				ctu.table_schema || '.' || ctu.table_name || '.' || c.column_name as foreign_table,
				c.ordinal_position
			from
				information_schema.columns c,
					information_schema.constraint_table_usage ctu,
					information_schema.constraint_column_usage ccu
			where
					ctu.constraint_name = $1 and
					ctu.table_schema = $2 and
					ccu.constraint_name = ctu.constraint_name and
					ccu.table_schema = ctu.table_schema and
					c.table_schema = ctu.table_schema and
					c.table_name = ccu.table_name and
					ccu.column_name = c.column_name
		`
		rowsFKConstraint, errFKConstrains := postCrawler.dbConn.Query(sqlFKConstraints, constraintName, schemaName)
		if errFKConstrains != nil {
			continue
		}

		defer rowsFKConstraint.Close()

		for rowsFKConstraint.Next() {
			var fkColumn databasemodels.Column

			if err := rowsFKConstraint.Scan(&fkColumn.Name, &fkColumn.Position); err != nil {
				continue
			}

			fkColumn.Table = generateInternalName(postCrawler.Host, postCrawler.DBName, schemaName, tableName)

			constraint.Destinations = append(constraint.Destinations, fkColumn)
			break

		}

		constraints = append(constraints, constraint)
	}

	return constraints, nil
}

func (postCrawler *postgresCrawler) getTableColumns(schemaName, tableName string) ([]databasemodels.Column, error) {
	columns := []databasemodels.Column{}
	tableNameCleaned := cleanupSchemaName(tableName)
	sqlStatement := `select column_name, data_type, ordinal_position from information_schema.columns where table_schema = $1 and table_name = $2`
	rows, err := postCrawler.dbConn.Query(sqlStatement, schemaName, tableNameCleaned)
	if err != nil {
		return columns, err
	}

	defer rows.Close()

	for rows.Next() {
		var column databasemodels.Column
		if err := rows.Scan(&column.Name, &column.Type, &column.Position); err != nil {
			return columns, err
		}

		column.Table = generateInternalName(postCrawler.Host, postCrawler.DBName, schemaName, tableName)

		columns = append(columns, column)
	}

	return columns, nil
}

func (postCrawler *postgresCrawler) getTableIndexes(schemaName, tableName string) ([]databasemodels.Index, error) {
	indexes := []databasemodels.Index{}
	tableNameCleaned := cleanupSchemaName(tableName)
	sqlStatement := `select indexname from pg_indexes where schemaname = $1 AND tablename = $2`
	rows, err := postCrawler.dbConn.Query(sqlStatement, schemaName, tableNameCleaned)
	if err != nil {
		return indexes, err
	}

	defer rows.Close()

	for rows.Next() {
		index := databasemodels.Index{
			Table:  generateInternalName(postCrawler.Host, postCrawler.DBName, schemaName, tableName),
			Schema: generateInternalName(postCrawler.Host, postCrawler.DBName, schemaName, ""),
		}
		var indexName string
		if err := rows.Scan(&indexName); err != nil {
			return indexes, err
		}

		index.Name = indexName

		// Get index columns
		indexColsSqlStatement := `
			SELECT
				coalesce(att.attname,
							(('{' || pg_get_expr(
										idx.indexprs,
										idx.indrelid
									)
								|| '}')::text[]
							)[k.i]
						) AS index_column,
				k.i AS index_order
			FROM pg_index idx
			CROSS JOIN LATERAL unnest(idx.indkey) WITH ORDINALITY AS k(attnum, i)
			LEFT JOIN pg_attribute AS att
				ON idx.indrelid = att.attrelid AND k.attnum = att.attnum
			WHERE idx.indexrelid::regclass = '"%s"'::regclass`
		rowsIndexCols, errIndexCols := postCrawler.dbConn.Query(fmt.Sprintf(indexColsSqlStatement, indexName))
		if errIndexCols != nil {
			return indexes, errIndexCols
		}

		defer rowsIndexCols.Close()

		for rowsIndexCols.Next() {
			var indexCol databasemodels.Column
			if err := rowsIndexCols.Scan(&indexCol.Name, &indexCol.Position); err != nil {
				log.Debug().Msgf("Could not read column from the index %s because %s", indexName, err.Error())
				continue
			}
			index.Columns = append(index.Columns, indexCol)
		}

		indexes = append(indexes, index)
	}

	return indexes, nil
}

func (postCrawler *postgresCrawler) getSchemaMaterializedViewNames(schemaName string) ([]string, error) {
	viewNames := []string{}
	sqlStatement := `
		select matviewname as view_name
		from pg_matviews
		where schemaname = $1
		order by schemaname,
				view_name`
	rows, err := postCrawler.dbConn.Query(sqlStatement, schemaName)
	if err != nil {
		return viewNames, err
	}

	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return viewNames, err
		}

		viewNames = append(viewNames, name)
	}

	return viewNames, nil
}

func (postCrawler *postgresCrawler) getMaterializedView(schemaName, viewName string) (databasemodels.MaterializedView, error) {
	view := databasemodels.MaterializedView{
		Name:   viewName,
		Schema: schemaName,
	}
	sqlStatement := `
		select
			attr.attnum,
			attr.attname as column_name,
			tp.typname as datatype
		from pg_catalog.pg_attribute as attr
		join pg_catalog.pg_class as cls on cls.oid = attr.attrelid
		join pg_catalog.pg_namespace as ns on ns.oid = cls.relnamespace
		join pg_catalog.pg_type as tp on tp.oid = attr.atttypid
		where
			ns.nspname = $1
			and cls.relname = $2
			and attr.attnum >= 1
		order by
			attr.attnum
	`
	rows, err := postCrawler.dbConn.Query(sqlStatement, schemaName, viewName)
	if err != nil {
		return view, err
	}

	defer rows.Close()

	for rows.Next() {
		var column databasemodels.Column
		if err := rows.Scan(&column.Position, &column.Name, &column.Type); err != nil {
			return view, nil
		}

		view.Columns = append(view.Columns, column)
	}

	return view, nil
}
