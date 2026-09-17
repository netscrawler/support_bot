-- name: ListActiveCrons :many
select cron, name, event_type from crons where is_active = true;

-- name: ListEventsForActiveReports :many
select c.name as cron_name, r.name as report_name
from report_crons rc join crons c on c.id = rc.cron_id
join reports r on r.id = rc.report_id where r.active = true;

-- name: ListEventsForActiveReportsByCronName :many
select c.name as cron_name, r.name as report_name
from report_crons rc join crons c on c.id = rc.cron_id
join reports r on r.id = rc.report_id
where c.name = $1 and r.active = true;
