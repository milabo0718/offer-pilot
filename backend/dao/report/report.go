package report

import (
	"context"

	"github.com/milabo0718/offer-pilot/backend/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ReportDao struct {
	db *gorm.DB
}

func NewReportDao(db *gorm.DB) *ReportDao {
	return &ReportDao{db: db}
}

func (d *ReportDao) GetBySessionID(ctx context.Context, sessionID string) (*model.InterviewReport, error) {
	var report model.InterviewReport
	err := d.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&report).Error
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (d *ReportDao) Upsert(ctx context.Context, report *model.InterviewReport) error {
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "session_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"user_name",
			"model_type",
			"jd_profile",
			"report_json",
			"updated_at",
		}),
	}).Create(report).Error
}
