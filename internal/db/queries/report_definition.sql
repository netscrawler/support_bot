-- report_definition.sql
-- Queries for report_definitions (models.ReportDefinition) — the export
-- pipeline's own persisted render records. Unrelated to report.sql, which
-- serves the DSL report-configuration domain (models.Report) against the
-- differently-named "reports" table.

-- name: CreateReportDefinition :exec
insert into report_definitions
    (id, name, description, layout, format, status, file_path, error_msg, created_at, updated_at, expires_at)
values
    ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: GetReportDefinitionByID :one
select id, name, description, layout, format, status, file_path, error_msg, created_at, updated_at, expires_at
from report_definitions
where id = $1;

-- name: UpdateReportDefinition :exec
update report_definitions
set name = $2, description = $3, layout = $4, format = $5, status = $6,
    file_path = $7, error_msg = $8, updated_at = $9, expires_at = $10
where id = $1;

-- name: ListReportDefinitionsByStatus :many
select id, name, description, layout, format, status, file_path, error_msg, created_at, updated_at, expires_at
from report_definitions
where status = $1
order by created_at asc
limit $2;

-- name: DeleteReportDefinition :exec
delete from report_definitions where id = $1;

-- name: CleanupExpiredReportDefinitions :execrows
delete from report_definitions
where expires_at is not null and expires_at < now();
