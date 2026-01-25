// Package text предоставляет набор функций для использования в Go шаблонах.
//
// Все функции доступны через FuncMap и могут использоваться в text/template и html/template.
// Помимо функций из этого пакета, также доступны все функции из библиотеки Sprig.
package text

/*
ПРИМЕРЫ ИСПОЛЬЗОВАНИЯ
===========================================

Пример 1: Форматирование денежных сумм
Данные: {"send_amount": 1234567.89, "send_count": 42}
Шаблон:
  Сумма: {{ formatMoney .send_amount }}
  Количество: {{ formatNumber .send_count }}

Результат:
  Сумма: 1 234 567.89
  Количество: 42

Пример 2: Работа с датами
Данные: текущая дата = 2026-01-25
Шаблон:
  {{ $lastMonth := lastMonth now }}
  Отчет за {{ formatRuMonthYear $lastMonth }}

Результат:
  Отчет за декабрь 2025

Пример 3: Упрощенный доступ к map
Данные: {"SomeMap": [{"amount": 1000}]}
Старый способ:
  {{ range index . "SomeMap" }}
    {{ .amount }}
  {{ end }}

Новый способ (все еще используем index, но с упрощениями):
  {{ $stats := index . "SomeMap" }}
  {{ if not (isEmpty $stats) }}
    {{ range $stats }}
      {{ formatMoney .amount }}
    {{ end }}
  {{ end }}

Пример 4: Работа с JSON
Данные: {"some_json": '{"some_key_one": 5, "some_key_two": 10}'}
Старый способ:
  {{- if and . (ne . "null") -}}
    {{- $some_map := fromJson . -}}
    {{- range $k, $v := $some_map -}}
      {{ $k }}: {{ $v }}
    {{- end -}}
  {{- end -}}

Новый способ:
  {{ $some_map := parseJsonMap .some_json }}
  {{ if notEmpty .some_json }}
    {{ range $k, $v := $some_map }}
      {{ $k }}: {{ $v }}
    {{ end }}
  {{ end }}

Пример 5: Условное форматирование
Данные: {"bank": "BANK Bank", "amount": 1000}
Шаблон:
  {{ if contains .bank "BANK" }}
    Банк BANK: {{ formatMoney .amount }}
  {{ else }}
    Другой банк: {{ formatMoney .amount }}
  {{ end }}

Результат:
  Банк BANK: 1 000.00

Пример 6: Значения по умолчанию
Данные: {"description": ""}
Шаблон:
  Описание: {{ default "Нет описания" .description }}

Результат:
  Описание: Нет описания

Пример 7: Проверка на пустоту и форматирование
Данные: {"items": [1, 2, 3], "total": 0}
Шаблон:
  {{ if not (isEmpty .items) }}
    Количество элементов: {{ size .items }}
  {{ end }}
  {{ if isZero .total }}
    Итого: данные отсутствуют
  {{ else }}
    Итого: {{ formatMoney .total }}
  {{ end }}

Результат:
  Количество элементов: 3
  Итого: данные отсутствуют
*/
