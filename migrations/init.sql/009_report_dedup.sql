alter table reports add column dedup boolean not null default false;
alter table report_runs add column data_hash text;
create index report_runs_last_success on report_runs (report_name, started_at desc)
    where status = 'success' and kind in ('cron', 'manual');
