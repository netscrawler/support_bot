-- name: ReportExistsByName :one
select exists(select 1 from reports where name = $1);
