package message

import (
	"context"

	"github.com/milabo0718/offer-pilot/backend/model"
	"gorm.io/gorm"
)

type MessageDao struct {
	DB *gorm.DB
}

func NewMessageDao(db *gorm.DB) *MessageDao {
	return &MessageDao{DB: db}
}

func (d *MessageDao) GetMessagesBySessionID(ctx context.Context, sessionID string) ([]model.Message, error) {
	var msgs []model.Message
	err := d.DB.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at asc").Find(&msgs).Error
	return msgs, err
}

func (d *MessageDao) GetMessagesBySessionIDs(ctx context.Context, sessionIDs []string) ([]model.Message, error) {
	var msgs []model.Message
	if len(sessionIDs) == 0 {
		return msgs, nil
	}
	err := d.DB.WithContext(ctx).Where("session_id IN ?", sessionIDs).Order("created_at asc").Find(&msgs).Error
	return msgs, err
}

func (d *MessageDao) CreateMessage(ctx context.Context, message *model.Message) (*model.Message, error) {
	err := d.DB.WithContext(ctx).Create(message).Error
	return message, err
}

func (d *MessageDao) GetAllMessages(ctx context.Context) ([]model.Message, error) {
	var msgs []model.Message
	err := d.DB.WithContext(ctx).Order("created_at asc").Find(&msgs).Error
	return msgs, err
}
