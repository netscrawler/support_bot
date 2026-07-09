package models

const recipientStringTemplate = `<b>Название:</b> {{.Name}}
{{if .Chat}}<b>Чат:</b> <code>{{.Chat.ChType}}</code>
{{if .Chat.Title}}<b>Название:</b> {{.Chat.Title}}
{{end}}<b>ID:</b> <code>{{.Chat.ChatID}}</code> {{if .ThreadID}}<b>Thread ID:</b> <code>{{.ThreadID}}</code>{{end}}{{end}}
<b>Удалить после конца дня:</b> {{if .NeedDeleteAfterEndOfDay}}Да{{else}}Нет{{end}}
{{if .Email}}<b>Email</b>
<b>Получатели:</b>
{{range .Email.Dest}}• <code>{{.}}</code>
{{end}}{{if .Email.Copy}}
<b>Копия:</b>
{{range .Email.Copy}}• <code>{{.}}</code>
{{end}}{{end}}<b>Тема:</b> {{.Email.Subject}}{{end}}
{{if .RemotePath}}<b>Путь на диске:</b> <code>{{.RemotePath}}</code>{{end}}
`

const exportStringTemplate = `<code>{{.Format}}</code>
{{if .FileName}}<b>Имя файла:</b> <code>{{.FileName}}</code>{{end}}
{{if .Template}}<b>Шаблон:</b> <code>{{.Template.Type}}</code>
<b>Название:</b> {{.Template.Title}}{{end}}`

const reportInfoStringTemplate = `<b>Отчет:  {{.Name}}</b> {{.Title}}
<b>Запросы:</b>
{{if .Queries}}{{range .Queries}}• {{ .}}
{{end}}{{else}}пусто{{end}}
<b>Получатели:</b>
{{if .Recipients}}{{range .Recipients}}• {{ .}}{{end}}{{else}}пусто{{end}}
<b>Формат:</b>
{{if .Exports}}{{range .Exports}}• {{.}}{{end}}{{else}}пусто{{end}}
<b>Условие отправки:</b>{{if .Evaluation}}<code>{{ .Evaluation}}</code>{{else}}пусто{{end}}
<b>Расписание отправки:</b>
{{if .LinkedCron}}{{range .LinkedCron}}• <code>{{ .}}</code>
{{end}}{{else}}пусто
{{end}}
<b>Следующая отправка:</b>
{{if .NextCron}}<code>{{ .NextCron}}</code>{{else}}пусто{{end}}`
