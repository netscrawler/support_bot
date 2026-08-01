package service

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReportManager_LoadDir_Validation(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	val := &ReportValidation{}
	m := NewReportManager(nil, val, log)

	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "report_test")
	assert.NoError(t, err)

	defer os.RemoveAll(tmpDir)

	// 1. Invalid JSON - should log and continue (no error returned from LoadDir)
	err = os.WriteFile(tmpDir+"/invalid.json", []byte("{ invalid }"), 0o644)
	assert.NoError(t, err)

	err = m.LoadDir(ctx, tmpDir)
	assert.NoError(t, err)

	// 2. Valid JSON but validation fails (empty name)
	err = os.WriteFile(tmpDir+"/noname.json", []byte(`{"title": "No Name"}`), 0o644)
	assert.NoError(t, err)

	err = m.LoadDir(ctx, tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "report name is empty")

	// 3. Duplicate query title
	err = os.WriteFile(tmpDir+"/dup_query.json", []byte(`{
		"name": "DUP_QUERY",
		"title": "Dup Query",
		"evaluation": "expr",
		"queries": [{"title": "Q1"}, {"title": "Q1"}]
	}`), 0o644)
	assert.NoError(t, err)

	err = m.LoadDir(ctx, tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate query title \"Q1\"")
}
