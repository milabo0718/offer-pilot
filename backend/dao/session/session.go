package session

import (
	"context"

	"github.com/milabo0718/offer-pilot/backend/model"
	"gorm.io/gorm"
)

type SessionDao struct {
	db *gorm.DB
}

func NewSessionDao(db *gorm.DB) *SessionDao {
	return &SessionDao{db: db}
}

func (d *SessionDao) GetSessionsByUserName(ctx context.Context, UserName int64) ([]model.Session, error) {
	var sessions []model.Session
	err := d.db.WithContext(ctx).Where("user_name = ?", UserName).Find(&sessions).Error
	return sessions, err
}

func (d *SessionDao) CreateSession(ctx context.Context, session *model.Session) (*model.Session, error) {
	err := d.db.WithContext(ctx).Create(session).Error
	return session, err
}

func (d *SessionDao) GetSessionByID(ctx context.Context, sessionID string) (*model.Session, error) {
	var session model.Session
	err := d.db.WithContext(ctx).Where("id = ?", sessionID).First(&session).Error
	return &session, err
}
