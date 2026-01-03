package exporter

import "bytes"

type Exporter interface {
	Export() (*bytes.Buffer, error)
}
