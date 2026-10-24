alter table public_reports
    add column created_at timestamptz not null default now(),
    add column expires_at timestamptz;
