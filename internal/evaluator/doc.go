// Package evaluator предоставляет функциональность для оценки необходимости отправки отчета
// на основе CEL (Common Expression Language) выражений.
//
// Основная задача модуля - анализ структурированных данных отчета и принятие решения
// о необходимости его отправки на основе заданных правил.
//
// Входные данные:
//
// На вход получает структуру map[string][]map[string]any, где:
//   - Ключ первого уровня - название листа(запроса в бд queries поле - title)
//   - Значение - массив записей (каждая запись это map[string]any)
//
// Пример:
//
//	{
//	  "sheet1": {},
//	  "sheet2": {
//	    {"total": 0, "count": 5},
//	    {"total": 10, "count": 0},
//	  },
//	}
//
// CEL окружение:
//
// Работает через [go-cel](https://github.com/google/cel-go) библиотеку с подключенными модулями:
//   - cel.StdLib() - стандартная библиотека CEL
//   - ext.Lists() - расширенные операции со списками
//   - ext.Sets() - операции с множествами
//   - ext.TwoVarComprehensions() - двухпараметровые comprehensions
//   - cel.OptionalTypes() - поддержка опциональных типов
//   - cel.Macros(cel.StandardMacros...) - стандартные макросы
//
// Доступная переменная в выражениях:
//   - report - map[string][]map[string]any с данными отчета
//
// Специальные выражения:
//   - "[*]" (AlwaysTrueExpr) - всегда возвращает true
//   - "[!*]" (AlwaysFalseExpr) - всегда возвращает false
//
// Примеры CEL выражений:
//
//	1)
//
//	report := map[string][]map[string]any{
//		"sheet1": {
//			{"total": 0, "count": 5},
//			{"total": 10, "count": 0},
//		},
//	}
//
//	// Проверка что все записи в sheet1 имеют total != 0
//	report["sheet1"].all(r, r["total"] != 0)
//
//	2)
//
//	report := map[string][]map[string]any{
//		"sheet1": {},
//	}
//
//	// Проверка количества записей в конкретном листе
//	size(report["sheet1"]) > 1 -> Вернет false
//
//	3)
//
//	report := map[string][]map[string]any{
//		"sheet1": {},
//		"sheet2": {
//			{"total": 0, "count": 5},
//			{"total": 10, "count": 0},
//		},
//	}
//
// Подсчет общего количества записей во всех листах
//
//	report.map(k, report[k]).flatten().size() > 1 -> Вернет true
//
// Кеширование:
//
// # Модуль использует LRU кеш (размер 15) для хранения скомпилированных CEL программ
//
// Использование:
//
//	log := slog.Default()
//	eval, err := evaluator.NewEvaluator(log)
//	if err != nil {
//	    // обработка ошибки
//	}
//
//	report := map[string][]map[string]any{
//	    "sheet1": {
//	        {"total": 10, "count": 5},
//	    },
//	}
//
//	expr := `size(report["sheet1"]) > 0`
//	shouldSend, err := eval.Evaluate(ctx, report, expr)
//	if err != nil {}
package evaluator
