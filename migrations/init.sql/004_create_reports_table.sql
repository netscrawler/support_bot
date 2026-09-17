-- Migration: report_definitions — storage for models.ReportDefinition, the
-- new canonical export-pipeline report record (layout + render status).
--
-- Deliberately NOT named "reports": migrations/init.sql/002_create_report_tables.sql
-- already creates a table called "reports" for the unrelated DSL report-
-- configuration domain (models.Report — queries/recipients/exports/crons).
-- The original SQLite-dialect version of this file also used "reports" with
-- CREATE TABLE IF NOT EXISTS, which would silently no-op against Postgres
-- (002 runs first, lexicographically) instead of failing loudly. Kept
-- distinct on purpose; do not rename back to "reports".

create table report_definitions
(
    id          text primary key,
    name        text not null,
    description text not null default '',
    layout      jsonb,
    format      text not null,
    status      text not null default 'pending'
                check (status in ('pending', 'processing', 'completed', 'failed')),
    file_path   text not null default '',
    error_msg   text not null default '',
    created_at  timestamptz not null default now(),
    updated_at  timestamptz not null default now(),
    expires_at  timestamptz
);

create index idx_report_definitions_status_active
    on report_definitions (status)
    where status in ('pending', 'processing');

create index idx_report_definitions_expires_at
    on report_definitions (expires_at)
    where expires_at is not null;
