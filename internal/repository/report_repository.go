package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"support_bot/internal/models"
)

// ReportRepository implements models.ReportRepository using SQLite/PostgreSQL
type ReportRepository struct {
	db *sql.DB
}

// NewReportRepository creates a new report repository instance
func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

// Create inserts a new report record
func (r *ReportRepository) Create(report *models.ReportDefinition) error {
	layoutJSON, err := json.Marshal(report.Layout)
	if err != nil {
		return fmt.Errorf("failed to marshal layout: %w", err)
	}

	query := `
		INSERT INTO reports (id, name, description, layout_json, format, status, file_path, error_msg, created_at, updated_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var expiresAt sql.NullTime
	if report.ExpiresAt != nil {
		expiresAt = sql.NullTime{Time: *report.ExpiresAt, Valid: true}
	}

	_, err = r.db.Exec(query,
		report.ID,
		report.Name,
		report.Description,
		string(layoutJSON),
		report.Format,
		report.Status,
		report.FilePath,
		report.ErrorMsg,
		report.CreatedAt,
		report.UpdatedAt,
		expiresAt,
	)

	return err
}

// GetByID retrieves a report by its ID
func (r *ReportRepository) GetByID(id string) (*models.ReportDefinition, error) {
	query := `
		SELECT id, name, description, layout_json, format, status, file_path, error_msg, created_at, updated_at, expires_at
		FROM reports
		WHERE id = ?
	`

	row := r.db.QueryRow(query, id)

	report := &models.ReportDefinition{}
	var layoutJSON string
	var expiresAt sql.NullTime

	err := row.Scan(
		&report.ID,
		&report.Name,
		&report.Description,
		&layoutJSON,
		&report.Format,
		&report.Status,
		&report.FilePath,
		&report.ErrorMsg,
		&report.CreatedAt,
		&report.UpdatedAt,
		&expiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		report.ExpiresAt = &expiresAt.Time
	}

	if layoutJSON != "" {
		report.Layout = &models.Layout{}
		if err := json.Unmarshal([]byte(layoutJSON), report.Layout); err != nil {
			return nil, fmt.Errorf("failed to unmarshal layout: %w", err)
		}
	}

	return report, nil
}

// Update updates an existing report record
func (r *ReportRepository) Update(report *models.ReportDefinition) error {
	layoutJSON, err := json.Marshal(report.Layout)
	if err != nil {
		return fmt.Errorf("failed to marshal layout: %w", err)
	}

	query := `
		UPDATE reports
		SET name = ?, description = ?, layout_json = ?, format = ?, status = ?, 
		    file_path = ?, error_msg = ?, updated_at = ?, expires_at = ?
		WHERE id = ?
	`

	var expiresAt sql.NullTime
	if report.ExpiresAt != nil {
		expiresAt = sql.NullTime{Time: *report.ExpiresAt, Valid: true}
	}

	_, err = r.db.Exec(query,
		report.Name,
		report.Description,
		string(layoutJSON),
		report.Format,
		report.Status,
		report.FilePath,
		report.ErrorMsg,
		report.UpdatedAt,
		expiresAt,
		report.ID,
	)

	return err
}

// ListByStatus returns reports filtered by status
func (r *ReportRepository) ListByStatus(status models.ReportStatus, limit int) ([]*models.ReportDefinition, error) {
	query := `
		SELECT id, name, description, layout_json, format, status, file_path, error_msg, created_at, updated_at, expires_at
		FROM reports
		WHERE status = ?
		ORDER BY created_at ASC
		LIMIT ?
	`

	rows, err := r.db.Query(query, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []*models.ReportDefinition

	for rows.Next() {
		report := &models.ReportDefinition{}
		var layoutJSON string
		var expiresAt sql.NullTime

		err := rows.Scan(
			&report.ID,
			&report.Name,
			&report.Description,
			&layoutJSON,
			&report.Format,
			&report.Status,
			&report.FilePath,
			&report.ErrorMsg,
			&report.CreatedAt,
			&report.UpdatedAt,
			&expiresAt,
		)

		if err != nil {
			return nil, err
		}

		if expiresAt.Valid {
			report.ExpiresAt = &expiresAt.Time
		}

		if layoutJSON != "" {
			report.Layout = &models.Layout{}
			if err := json.Unmarshal([]byte(layoutJSON), report.Layout); err != nil {
				return nil, fmt.Errorf("failed to unmarshal layout: %w", err)
			}
		}

		reports = append(reports, report)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return reports, nil
}

// Delete removes a report record
func (r *ReportRepository) Delete(id string) error {
	query := `DELETE FROM reports WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// CleanupExpired removes reports past their expiration time
func (r *ReportRepository) CleanupExpired() (int64, error) {
	query := `DELETE FROM reports WHERE expires_at IS NOT NULL AND expires_at < ?`
	result, err := r.db.Exec(query, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
