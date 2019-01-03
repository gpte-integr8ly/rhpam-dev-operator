package rhpamdev

const (
	PostgresqlJbpmSchema        = "postgresql-jbpm-schema.sql"
	PostgresqlJbpmLoTriggerClob = "postgresql-jbpm-lo-trigger-clob.sql"
	QuartzTablesPostgresql      = "quartz_tables_postgres.sql"
	ScriptCreateDatabase        = "create_rhpam_database.sh"
	ScriptWaitForPostgresql     = "wait_for_postgresql.sh"
)

func initConfigmapResources() []string {
	resources := []string{PostgresqlJbpmSchema, PostgresqlJbpmLoTriggerClob, QuartzTablesPostgresql, ScriptCreateDatabase, ScriptWaitForPostgresql}
	return resources
}
