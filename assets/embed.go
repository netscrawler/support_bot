package assets

import "embed"

//go:embed fonts/*.ttf
var FontFS embed.FS

//go:embed js/chart.umd.min.js
var ChartFS embed.FS
