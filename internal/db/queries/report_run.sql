-- report_run.sql
-- Журнал запусков отчётов (таблица report_runs).

-- name: InsertReportRun :exec
insert into report_runs(report_name, kind, status, reason, error, chat_id, started_at, finished_at, data_hash)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: LastReportRunStatus :one
select status from report_runs
where report_name = $1 and kind in ('cron', 'manual')
  and (status <> 'skipped' or reason is distinct from 'duplicate')
order by started_at desc
limit 1;

-- name: LastReportRunSuccessHash :one
select data_hash from report_runs
where report_name = $1 and status = 'success' and kind in ('cron', 'manual')
order by started_at desc
limit 1;

-- name: ListReportRuns :many
select id, report_name, kind, status, reason, error, chat_id, started_at, finished_at
from report_runs
where report_name = $1
order by started_at desc
limit $2;

-- name: DeleteReportRunsBefore :execrows
delete from report_runs where started_at < $1;
