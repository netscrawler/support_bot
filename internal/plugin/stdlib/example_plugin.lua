-- Пример Lua плагина с использованием stdlib

plugin = {
	name = "example_with_stdlib",
	version = "1.0.0",
	description = "Пример плагина с использованием стандартной библиотеки",
	author = "Your Name",

	-- Инициализация плагина
	init = function(config)
		-- Проверяем конфигурацию
		if not config.api_key then
			return false, "api_key is required"
		end

		-- Сохраняем конфигурацию
		plugin.config = config

		return true, nil
	end,

	-- Валидация параметров
	validate = function(params)
		if not params.card_ids then
			return false, "card_ids parameter is required"
		end

		return true, nil
	end,

	-- Основная логика выполнения
	execute = function(params)
		-- 1. Подготавливаем карточки для сбора данных
		local cards = {}
		for i, id in ipairs(params.card_ids) do
			table.insert(cards, {
				id = id,
				name = "Card " .. id,
			})
		end

		-- 2. Собираем данные через stdlib
		local data, err = stdlib.Collect(cards)
		if err then
			return nil, "Failed to collect data: " .. err
		end

		-- 3. Проверяем условие через stdlib.Evaluate
		local expression = params.expression or "len(data.card_1) > 0"
		local condition_met, eval_err = stdlib.Evaluate(data, expression)
		if eval_err then
			return nil, "Failed to evaluate: " .. eval_err
		end

		if not condition_met then
			-- Условие не выполнено - не отправляем отчёт
			return {
				status = "skipped",
				reason = "Condition not met: " .. expression,
			}, nil
		end

		-- 4. Экспортируем данные в нужный формат
		local export_config = {
			format = params.format or "xlsx",
			template = params.template or "default.tmpl",
		}

		local report, export_err = stdlib.Export(data, export_config)
		if export_err then
			return nil, "Failed to export: " .. export_err
		end

		-- 5. Отправляем отчёт
		if params.telegram_chat then
			-- Отправка через Telegram
			local chat = {
				chat_id = params.telegram_chat.chat_id,
				topic_id = params.telegram_chat.topic_id or 0,
			}

			local send_err = stdlib.SendTelegram(chat, report)
			if send_err then
				return nil, "Failed to send telegram: " .. send_err
			end
		elseif params.file_server then
			-- Загрузка на файловый сервер
			local remote_path = params.file_server.path or "/reports/"

			local upload_err = stdlib.SendFileServer(remote_path, report)
			if upload_err then
				return nil, "Failed to upload: " .. upload_err
			end
		elseif params.email then
			-- Отправка по email
			local mail = {
				from = params.email.from,
				to = params.email.to,
				subject = params.email.subject or "Report",
				body = params.email.body or "Please find the report attached.",
				attachments = { report },
			}

			local email_err = stdlib.SendEmail(mail)
			if email_err then
				return nil, "Failed to send email: " .. email_err
			end
		end

		-- 6. Возвращаем результат
		return {
			status = "success",
			cards_processed = #cards,
			report_generated = true,
			format = export_config.format,
		},
			nil
	end,

	-- Очистка ресурсов
	cleanup = function()
		plugin.config = nil
	end,
}
