-- report.sql
-- Consolidated report/card/recipient/export/pipeline/cron queries.
-- Replaces internal/repository/report.go, internal/orchestrator/orchestrator_repository.go,
-- and internal/tg_bot/repository/report_repository.go's independent copies of this SQL.

-- name: ReportExistsByName :one
select exists(select 1 from reports where name = $1);

-- name: GetReportRowByID :one
select r.id, r.name, r.title, r.active, r.access_from_lk, r.pipeline_id, e.expr as evaluation
from reports r
left join evaluate e on e.id = r.eval_id
where r.id = $1::bigint
limit 1;

-- name: ListReportRows :many
select r.id, r.name, r.title, r.active, r.access_from_lk, r.pipeline_id, e.expr as evaluation
from reports r
left join evaluate e on e.id = r.eval_id;

-- name: ListActiveReportRows :many
select r.id, r.name, r.title, r.active, r.access_from_lk, r.pipeline_id, e.expr as evaluation
from reports r
left join evaluate e on e.id = r.eval_id
where r.active = true;

-- name: GetActiveReportRowByName :one
select r.id, r.name, r.title, r.active, r.access_from_lk, r.pipeline_id, e.expr as evaluation
from reports r
left join evaluate e on e.id = r.eval_id
where r.name = $1 and r.active = true
limit 1;

-- name: GetAnyReportRowByName :one
select r.id, r.name, r.title, r.active, r.access_from_lk, r.pipeline_id, e.expr as evaluation
from reports r
left join evaluate e on e.id = r.eval_id
where r.name = $1
limit 1;

-- name: GetReportIDByPublicID :one
select report_id from public_reports where public_id = $1 limit 1;

-- name: ListCardsByReportID :many
select q.card_uuid, q.title, q.q_type, q.params
from report_queries rq
join queries q on q.id = rq.query_id
where rq.report_id = $1::bigint;

-- name: ListRecipientsByReportID :many
select
    rc.name,
    rc.config,
    rc.remote_path,
    rc.thread_id,
    rc.email_id,
    rc.type,
    rc.need_delete_after_end_of_day,
    e.dest,
    e.copy,
    e.subject,
    e.body,
    c.chat_id,
    c.title as chat_title,
    c.type as chat_type,
    c.description as chat_description,
    c.is_active as chat_is_active,
    c.ch_type as chat_ch_type
from reports_recipients rr
join recipients rc on rc.id = rr.recipient_id
left join chats c on c.id = rc.chat_id
left join email_templates e on e.id = rc.email_id
where rr.report_id = $1::bigint;

-- name: ListExportsByReportID :many
select ef.format, re.file_name, t.id as template_id, t.title as template_title,
       t.type as template_type, t.template_text, re.sort_order
from reports_export re
join export_formats ef on ef.id = re.format_id
left join report_templates rt on rt.report_id = re.report_id
left join templates t on t.id = rt.template_id
where re.report_id = $1::bigint;

-- name: ListCronsByReportID :many
select c.cron, c.name, c.description, c.is_active, c.event_type
from report_crons rc
join crons c on c.id = rc.cron_id
where rc.report_id = $1::bigint;

-- name: GetPipelineByID :one
select pipeline from pipelines where id = $1::bigint;

-- name: ListReportsForLK :many
select id, name, title from reports where access_from_lk = true order by id limit $1 offset $2;

-- name: CountReportsForLK :one
select count(*) from reports where access_from_lk = true;

-- name: ListReportLinkedCrons :many
select c.cron as cron, c.name as name
from report_crons rc
left join crons c on c.id = rc.cron_id
left join reports r on rc.report_id = r.id
where c.is_active = true and r.name = $1;

-- Write-side queries for ReportStore.Create's transaction (Task 7).

-- name: FindEvaluationIDByExpr :one
select id from evaluate where expr = $1;

-- name: CreateEvaluation :one
insert into evaluate(expr) values ($1) returning id;

-- name: CreatePipeline :one
insert into pipelines(pipeline) values ($1) returning id;

-- name: FindQueryIDByUUIDAndTitle :one
select id from queries where card_uuid = $1 and title = $2;

-- name: CreateQuery :one
insert into queries(card_uuid, title, q_type, params) values ($1, $2, $3, $4) returning id;

-- name: FindRecipientIDByName :one
select id from recipients where name = $1;

-- name: CreateRecipient :one
insert into recipients(name, config, remote_path, chat_id, thread_id, email_id, type, need_delete_after_end_of_day)
values ($1, '{}', $2, $3, $4, $5, $6, $7)
returning id;

-- name: CreateEmailTemplate :one
insert into email_templates(dest, copy, subject, body) values ($1, $2, $3, $4) returning id;

-- name: CreateReport :one
insert into reports(name, title, eval_id, pipeline_id, access_from_lk, active)
values ($1, $2, $3, $4, $5, $6)
returning id;

-- name: LinkQueryToReport :exec
insert into report_queries(report_id, query_id) values ($1, $2);

-- name: LinkRecipientToReport :exec
insert into reports_recipients(report_id, recipient_id) values ($1, $2);

-- name: FindExportFormatIDByFormat :one
select id from export_formats where format = $1;

-- name: CreateExportFormat :one
insert into export_formats(format) values ($1) returning id;

-- name: LinkExportToReport :exec
insert into reports_export(report_id, format_id, file_name, sort_order) values ($1, $2, $3, $4);

-- name: LinkTemplateToReport :exec
insert into report_templates(report_id, template_id) values ($1, $2);

-- name: CreateTemplate :one
insert into templates(template_text, title, type) values ($1, $2, $3) returning id;

-- name: FindTemplateIDByTitleAndType :one
select id from templates where title = $1 and type = $2;

-- name: FindCronIDByNameAndExpr :one
select id from crons where name = $1 and cron = $2;

-- name: CreateCron :one
insert into crons(cron, name, description, is_active, event_type) values ($1, $2, $3, $4, $5) returning id;

-- name: LinkCronToReport :exec
insert into report_crons(report_id, cron_id) values ($1, $2);
