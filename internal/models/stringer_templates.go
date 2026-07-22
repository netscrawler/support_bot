package models

const recipientStringTemplate = `
<b>{{.Name}}</b>

{{if .Chat}}
<p><b>Чат</b></p>
<ul>
    <li><b>Тип:</b> <code>{{.Chat.ChType}}</code></li>
    {{if .Chat.Title}}<li><b>Название:</b> {{.Chat.Title}}</li>{{end}}
    <li><b>ID:</b> <code>{{.Chat.ChatID}}</code></li>
    {{if .ThreadID}}<li><b>Thread ID:</b> <code>{{.ThreadID}}</code></li>{{end}}
</ul>
{{end}}

<p><b>Настройки</b></p>
<ul>
    <li><b>Удалить после конца дня:</b> {{if .NeedDeleteAfterEndOfDay}}Да{{else}}Нет{{end}}</li>
</ul>

{{if .Email}}
<p><b>Email</b></p>

<p><b>Получатели</b></p>
<ul>
{{range .Email.Dest}}
    <li><code>{{.}}</code></li>
{{end}}
</ul>

{{if .Email.Copy}}
<p><b>Копия</b></p>
<ul>
{{range .Email.Copy}}
    <li><code>{{.}}</code></li>
{{end}}
</ul>
{{end}}

<p><b>Тема:</b> {{.Email.Subject}}</p>
{{end}}

{{if .RemotePath}}
<p><b>Путь на диске:</b></p>
<blockquote><code>{{.RemotePath}}</code></blockquote>
{{end}}
`

const exportStringTemplate = `
<ul>
    <li><b>Формат:</b> <code>{{.Format}}</code></li>

    {{if .FileName}}
    <li><b>Имя файла:</b> <code>{{.FileName}}</code></li>
    {{end}}

    {{if .Template}}
    <li><b>Тип шаблона:</b> <code>{{.Template.Type}}</code></li>
    <li><b>Название:</b> {{.Template.Title}}</li>
    {{end}}
</ul>
`
const reportInfoStringTemplate = `
<h2>{{.Name}}</h2>

{{.Title}}

<h3>Запросы</h3>
{{if .Queries}}
{{range .Queries}}
{{.}}<br>
{{end}}
{{else}}
Пусто
{{end}}

<h3>Получатели</h3>
{{if .Recipients}}
{{range .Recipients}}
{{.}}<br>
{{end}}
{{else}}
Пусто
{{end}}

<h3>Форматы</h3>
{{if .Exports}}
{{range .Exports}}
{{.}}<br>
{{end}}
{{else}}
Пусто
{{end}}

<h3>Условие отправки</h3>
{{if .Evaluation}}
<blockquote><code>{{.Evaluation}}</code></blockquote>
{{else}}
Пусто
{{end}}

<h3>Расписание отправки</h3>
{{if .LinkedCron}}
{{range .LinkedCron}}
• <code>{{.}}</code><br>
{{end}}
{{else}}
Пусто
{{end}}

<h3>Следующая отправка</h3>
{{if .NextCron}}
<blockquote><code>{{.NextCron}}</code></blockquote>
{{else}}
Пусто
{{end}}
`
