package html

import (
	"fmt"
	"html/template"
	"strings"

	"support_bot/internal/models"
)

type pageStyleConfig struct {
	Format      string
	Orientation string
	WidthMM     int
	HeightMM    int
	PaddingMM   int
}

func styles(args ...any) template.HTML {
	cfg := defaultPageStyleConfig()

	if len(args) > 0 {
		cfg = pageStyleConfigFromAny(args[0], cfg)
	}

	return template.HTML(fmt.Sprintf(`<style>
    @page {
        size: %s %s;
        margin: 0;
    }

    * {
        box-sizing: border-box;
    }

    html,
    body {
        margin: 0;
        padding: 0;
        background: #eef1f5;
        font-family: Inter,
        -apple-system,
        BlinkMacSystemFont,
        "Segoe UI",
        Arial,
        sans-serif;
        color: #172033;
    }

    body {
        padding: 20px;
    }

    .page {
        width: %[4]dmm;
        min-height: %[5]dmm;
        margin: 0 auto;
        padding: %[6]dmm %[6]dmm;

        background: #ffffff;

        box-shadow: 0 8px 30px rgba(15, 23, 42, 0.10);

        position: relative;
    }

    .header {
        display: flex;
        align-items: flex-start;
        justify-content: space-between;

        padding-bottom: 10mm;
        margin-bottom: 8mm;

        border-bottom: 1px solid #e5e9f0;
    }

    .header-content {
        min-width: 0;
    }

    .eyebrow {
        margin: 0 0 4px;

        font-size: 10px;
        font-weight: 600;
        letter-spacing: 0.08em;
        text-transform: uppercase;

        color: #7a8499;
    }

    h1 {
        margin: 0;

        font-size: 25px;
        line-height: 1.2;
        font-weight: 700;
        letter-spacing: -0.02em;

        color: #111827;
    }

    .subtitle {
        margin: 6px 0 0;

        font-size: 12px;
        line-height: 1.5;

        color: #6b7280;
    }

    .report-date {
        padding: 7px 10px;

        border-radius: 8px;
        background: #f5f7fa;

        font-size: 11px;
        color: #667085;

        white-space: nowrap;
    }

    .section {
        margin-bottom: 7mm;
    }

    .section-title {
        margin: 0 0 4mm;

        font-size: 15px;
        font-weight: 650;

        color: #1f2937;
    }

    .report-grid {
        display: grid;
        grid-template-columns: repeat(12, minmax(0, 1fr));
        grid-template-rows: repeat(12, minmax(0, 1fr));
        gap: 5mm;

        width: 100%%;
        height: 230mm;
    }

    .report-page {
        width: %[4]dmm;
        height: %[5]dmm;
        box-sizing: border-box;
        position: relative;
        overflow: hidden;

        page-break-after: always;
        break-after: page;
    }

    .report-page:last-child {
        page-break-after: auto;
        break-after: auto;
    }

    .report-grid-item {
        min-width: 0;
        min-height: 0;
        overflow: hidden;

        break-inside: avoid;
        page-break-inside: avoid;
    }

    .chart-card {
        width: 100%%;
        min-width: 0;

        padding: 5mm;

        border: 1px solid #e5e9f0;
        border-radius: 10px;

        background: #ffffff;

        box-shadow: 0 2px 8px rgba(15, 23, 42, 0.04);
    }

    .chart-card-title {
        margin: 0 0 3mm;

        font-size: 12px;
        font-weight: 600;

        color: #344054;
    }

    .report-chart {
        position: relative;
        width: 100%%;
    }

    .footer {
        position: absolute;
        left: %[7]dmm;
        right: %[7]dmm;
        bottom: 9mm;

        display: flex;
        justify-content: space-between;

        padding-top: 4mm;

        border-top: 1px solid #e5e9f0;

        font-size: 9px;
        color: #98a2b3;
    }

    /*
     * Печать
     */
    @media print {
        html,
        body {
            background: #ffffff;
        }

        body {
            padding: 0;
        }

        .page {
            margin: 0;
            box-shadow: none;
        }

        .report-page {
            break-after: page;
            page-break-after: always;
        }

        .report-page:last-child {
            break-after: auto;
            page-break-after: auto;
        }
    }

    /*
     * Если экран уже A4 по высоте — не уменьшаем страницу.
     */
    @media screen {
        .page {
            min-height: %[5]dmm;
        }
    }
</style>`,
		cfg.Format,
		cfg.Orientation,
		cfg.PaddingMM,
		cfg.WidthMM,
		cfg.HeightMM,
		cfg.PaddingMM,
		cfg.PaddingMM,
	))
}

func defaultPageStyleConfig() pageStyleConfig {
	return pageStyleConfig{
		Format:      "A4",
		Orientation: "portrait",
		WidthMM:     210,
		HeightMM:    297,
		PaddingMM:   16,
	}
}

func pageStyleConfigFromAny(value any, fallback pageStyleConfig) pageStyleConfig {
	switch v := value.(type) {
	case models.PageConfig:
		return pageStyleConfigFromPageConfig(v, fallback)
	case *models.PageConfig:
		if v == nil {
			return fallback
		}
		return pageStyleConfigFromPageConfig(*v, fallback)
	case models.Layout:
		return pageStyleConfigFromPageConfig(v.Page, fallback)
	case *models.Layout:
		if v == nil {
			return fallback
		}
		return pageStyleConfigFromPageConfig(v.Page, fallback)
	default:
		return fallback
	}
}

func pageStyleConfigFromPageConfig(page models.PageConfig, fallback pageStyleConfig) pageStyleConfig {
	cfg := fallback

	if strings.TrimSpace(page.Format) != "" {
		cfg.Format = page.Format
	}
	if strings.TrimSpace(page.Orientation) != "" {
		cfg.Orientation = page.Orientation
	}
	if page.PaddingMM > 0 {
		cfg.PaddingMM = page.PaddingMM
	}

	switch strings.ToLower(cfg.Format) {
	case "a4":
		cfg.WidthMM = 210
		cfg.HeightMM = 297
	default:
		cfg.WidthMM = 210
		cfg.HeightMM = 297
	}

	if strings.EqualFold(cfg.Orientation, "landscape") {
		cfg.WidthMM, cfg.HeightMM = cfg.HeightMM, cfg.WidthMM
	}

	return cfg
}
