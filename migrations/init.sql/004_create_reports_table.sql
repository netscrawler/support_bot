-- Migration: Create reports table
-- Stores generated report metadata and layout configuration

CREATE TABLE IF NOT EXISTS reports (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT,
    layout_json TEXT NOT NULL,  -- JSON representation of models.Layout
    format      TEXT NOT NULL,  -- Target format: html, pdf, xlsx, csv, png, text
    status      TEXT NOT NULL DEFAULT 'pending', -- pending, processing, completed, failed
    file_path   TEXT,           -- Path to generated file on disk/S3
    error_msg   TEXT,           -- Error message if status is 'failed'
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at  DATETIME        -- Optional TTL for auto-cleanup
);

CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status);
CREATE INDEX IF NOT EXISTS idx_reports_created_at ON reports(created_at);
