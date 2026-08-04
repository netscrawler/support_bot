package models

var ReportExample = Report{
	Name:  "Example Report",
	Title: "Example Report Title",
	Queries: []Card{
		{
			CardUUID: "uuid",
			Title:    "example_mb",
			Type:     "mb",
		},
		{
			CardUUID: "appmetrica_url",
			Title:    "example appmetrica query",
			Params: map[string]string{
				"date1": "=yesterday",
				"date2": "=today",
			},
			Type: "appmetrica",
		},
		{
			CardUUID: "project=SS",
			Title:    "jira query",
			Params: map[string]string{
				"limit": "1",
			},
			Type: "jira",
		},
	},
	Recipients: []Recipient{
		{
			Name: "TG",
			Chat: &Chat{
				ChatID: -2184218821847818,
				Title:  ptr("Example Chat"),
				Type:   "group",
				ChType: "tg",
			},
			Email:                   nil,
			Type:                    "tg",
			NeedDeleteAfterEndOfDay: true,
		},
		{
			Name:       "Email",
			RemotePath: nil,
			Email: &EmailTemplate{
				Dest:    []string{"example@example.com"},
				Copy:    []string{"example2@example.com"},
				Subject: "EXAMPLE",
				Body:    ptr("EXAMPLE TEMPLATE"),
			},
			Type: "email",
		},
		{
			Name:       "SMB",
			RemotePath: ptr("/some/disk/path"),
			Type:       "smb",
		},
	},
	Exports: []Export{
		{
			Format: "text",
			Template: &Template{
				Title:        "EXAMPLE_TEMPLATE",
				Type:         "rich_text",
				TemplateText: "SOME_RICH_TEXT_TEMPLATE",
			},
		},
		{
			Format:   "xlsx",
			FileName: ptr("report.xlsx"),
			Order: map[string][]string{
				"sheet1": {"example_mb", "example appmetrica query"},
			},
		},
	},
	Pipeline: &Pipeline{
		Name: "ExamplePipe",
		Steps: []Step{
			{
				ID:         "example_step1",
				Type:       "lua",
				ScriptName: "example.lua",
				Params:     nil,
			},
			{
				ID:         "example_step2",
				Type:       "lua",
				ScriptName: "example2.lua",
				Params:     nil,
			},
		},
	},
	Evaluation:   "[*]",
	Active:       false,
	AccessFromLK: false,
	Crons: []Cron{
		{
			Name:     "example cron",
			Cron:     "* * * * *",
			IsActive: true,
		},
	},
}

func ptr[T any](t T) *T {
	return &t
}
