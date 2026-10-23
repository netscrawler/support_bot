create table report_runs (
    id          bigserial primary key,
    report_name text        not null,
    kind        text        not null, -- cron | manual | lk | test
    status      text        not null, -- success | skipped | failed | partial
    reason      text,                 -- для skipped: evaluate (позже duplicate)
    error       text,
    chat_id     bigint,               -- для lk/test: чат запросившего
    started_at  timestamptz not null,
    finished_at timestamptz not null
);
create index report_runs_name_started on report_runs (report_name, started_at desc);
create index report_runs_started on report_runs (started_at);
