-- Добавление первого админа (замените значения на свои)
INSERT INTO tg_users (telegram_id, username, first_name, role)
VALUES (476788912, 'netscrawler', 'Данил', 'primary');

insert into users(id,login, email,password,role,tg_profile,active)
values('08964796-783b-45a9-946f-3d9cb4c6b1d9','login@gmail.com','test@gmail.com', '$argon2id$v=19$m=65536,t=3,p=2$h6NPtaB3nWqn6sG11B3VXQ$+QSRRx5DNMl73EBeB1SBiU5nL88B0nrBT4cX+QMRPXQ','admin',1,true);
