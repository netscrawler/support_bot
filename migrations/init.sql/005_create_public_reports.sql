create table public_reports(
    id serial primary key,
    public_id uuid unique not null,
    report_id bigint not null,
    constraint fk_reports foreign key (report_id) references reports (id) on delete cascade
);