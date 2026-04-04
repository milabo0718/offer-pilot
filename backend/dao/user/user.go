package user

import (
	"context"

	"github.com/milabo0718/offer-pilot/backend/model"

	"gorm.io/gorm"
)

type UserDao struct {
	db *gorm.DB
}

func NewUserDao(db *gorm.DB) *UserDao {
	return &UserDao{db: db}
}

func (dao *UserDao) IsExistUser(ctx context.Context, username string) (bool, *model.User) {
	var user model.User

	err := dao.db.WithContext(ctx).Where("username = ?", username).First(&user).Error

	if err != nil {
		return false, nil
	}

	// 没报错说明查到了
	return true, &user
}

func (dao *UserDao) Register(ctx context.Context, username, email, password string) (*model.User, bool) {

	user := &model.User{
		Email:    email,
		Name:     username,
		Username: username,
		Password: password,
	}

	if err := dao.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, false
	}

	return user, true
}
