package postgres

import (
	"fmt"

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

func (postCrawler *postgresCrawler) getTableData(schemaName, tableName string) (post_model.Table, error) {
	table := post_model.Table{
		Name:        tableName,
		Columns:     []post_model.Column{},
		Constraints: []post_model.Constraint{},
		Schema:      schemaName,
		Indexes:     []string{},
	}

	columns, errColumns := postCrawler.getTableColumns(schemaName, tableName)
	if errColumns != nil {
		log.Warn().Msgf("Something happened while trying to get the columns of %s.%s due to %w", schemaName, tableName, errColumns)
	}
	table.Columns = columns

	constraints, errConstraints := postCrawler.getTableConstraints(schemaName, tableName)
	if errConstraints != nil {
		log.Warn().Msgf("Something happened while trying to get the constraints of %s.%s due to %w", schemaName, tableName, errConstraints)
	}
	table.Constraints = constraints

	return table, nil
}

func (postCrawler *postgresCrawler) getTableConstraints(schemaName, tableName string) ([]post_model.Constraint, error) {
	constraints := []post_model.Constraint{}

	// Get all constraint names of table
	sqlTableConstraints := `select constraint_name from information_schema.key_column_usage where table_schema = $1 and table_name = $2`
	resTableConstraints, errTableConstraints := postCrawler.dbConn.Query(sqlTableConstraints, schemaName, tableName)
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

		constraint := post_model.Constraint{
			Name:         constraintName,
			Type:         "",
			Sources:      []post_model.Column{},
			Destinations: []post_model.Column{},
		}

		// Get all columns of the constraint
		sqlConstraintsColumns := `
			select
				kcu.ordinal_position as position,
				kcu.column_name as key_column,
				tco.constraint_type
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
		rowsConstraintsColumns, errConstraitsColumns := postCrawler.dbConn.Query(sqlConstraintsColumns, schemaName, tableName, constraintName)
		if errConstraitsColumns != nil {
			log.Error().Msgf("Could not get columns of constraint %s.%s.%s", schemaName, tableName, constraintName)
			return constraints, errConstraitsColumns
		}

		defer rowsConstraintsColumns.Close()

		for rowsConstraintsColumns.Next() {
			var sourceConstraintCol post_model.Column
			if err := rowsConstraintsColumns.Scan(&sourceConstraintCol.Position, &sourceConstraintCol.Name, &constraintType); err != nil {
				continue
			}

			constraint.Type = constraintType
			constraint.Sources = append(constraint.Sources, sourceConstraintCol)
		}

		if constraintType != post_model.POSTGRES_CONSTRAINT_FK {
			constraints = append(constraints, constraint)
			continue
		}

		// Get all table relations for each constraints
		sqlFKConstraints := `
			select kcu.table_schema || '.' || kcu.table_name || '.' || kcu.column_name as foreign_table,
				kcu.ordinal_position
			from information_schema.table_constraints tco
			join information_schema.key_column_usage kcu
					on tco.constraint_schema = kcu.constraint_schema
					and tco.constraint_name = kcu.constraint_name
			join information_schema.referential_constraints rco
					on tco.constraint_schema = rco.constraint_schema
					and tco.constraint_name = rco.constraint_name
			join information_schema.table_constraints rel_tco
					on rco.unique_constraint_schema = rel_tco.constraint_schema
					and rco.unique_constraint_name = rel_tco.constraint_name
			where tco.constraint_type = 'FOREIGN KEY' and kcu.constraint_name = $1
			group by kcu.table_schema,
					kcu.table_name,
					rel_tco.table_name,
					rel_tco.table_schema,
					kcu.constraint_name,
					kcu.column_name,
					kcu.ordinal_position
			order by kcu.table_schema,
					kcu.table_name
		`
		rowsFKConstraint, errFKConstrains := postCrawler.dbConn.Query(sqlFKConstraints, constraintName)
		if errFKConstrains != nil {
			continue
		}

		defer rowsFKConstraint.Close()

		for rowsFKConstraint.Next() {
			var fkColumn post_model.Column

			if err := rowsFKConstraint.Scan(&fkColumn.Name, &fkColumn.Position); err != nil {
				continue
			}

			constraint.Destinations = append(constraint.Destinations, fkColumn)
			break

		}

		constraints = append(constraints, constraint)
	}

	return constraints, nil
}

func (postCrawler *postgresCrawler) getTableColumns(schemaName, tableName string) ([]post_model.Column, error) {
	columns := []post_model.Column{}
	sqlStatement := `select column_name, data_type, ordinal_position from information_schema.columns where table_schema = $1 and table_name = $2`
	rows, err := postCrawler.dbConn.Query(sqlStatement, schemaName, tableName)
	if err != nil {
		return columns, err
	}

	defer rows.Close()

	for rows.Next() {
		var column post_model.Column
		if err := rows.Scan(&column.Name, &column.Type, &column.Position); err != nil {
			return columns, err
		}
		columns = append(columns, column)
	}

	return columns, nil
}

func (postCrawler *postgresCrawler) getTableIndexes(schemaName, tableName string) ([]post_model.Index, error) {
	indexes := []post_model.Index{}
	sqlStatement := `select indexname from pg_indexes where schemaname = $1 AND tablename = $2`
	rows, err := postCrawler.dbConn.Query(sqlStatement, schemaName, tableName)
	if err != nil {
		return indexes, err
	}

	defer rows.Close()

	for rows.Next() {
		index := post_model.Index{
			Table:  tableName,
			Schema: schemaName,
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
			var indexCol post_model.Column
			if err := rowsIndexCols.Scan(&indexCol.Name, &indexCol.Position); err != nil {
				log.Debug().Msgf("Could not read column from the index %s because %w", indexName, err)
				continue
			}
			index.Columns = append(index.Columns, indexCol)
		}

		indexes = append(indexes, index)
	}

	return indexes, nil
}

func (postCrawler *postgresCrawler) getSchemaViewNames(schemaName string) ([]string, error) {
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

func (postCrawler *postgresCrawler) getView(schemaName, viewName string) (post_model.View, error) {
	view := post_model.View{
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
		var column post_model.Column
		if err := rows.Scan(&column.Position, &column.Name, &column.Type); err != nil {
			return view, nil
		}

		view.Columns = append(view.Columns, column)
	}

	return view, nil
}
