package mysql

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/milabo0718/offer-pilot/backend/config"
	"github.com/milabo0718/offer-pilot/backend/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB(conf *config.MysqlConfig) (*gorm.DB, error) {
	host := conf.MysqlHost
	port := conf.MysqlPort
	dbname := conf.MysqlDatabaseName
	username := conf.MysqlUser
	password := conf.MysqlPassword
	charset := conf.MysqlCharset

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&loc=Local", username, password, host, port, dbname, charset)

	var log logger.Interface
	if gin.Mode() == "debug" {
		log = logger.Default.LogMode(logger.Info)
	} else {
		log = logger.Default
	}

	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       dsn,
		DefaultStringSize:         256,
		DisableDatetimePrecision:  true,
		DontSupportRenameIndex:    true,
		DontSupportRenameColumn:   true,
		SkipInitializeWithVersion: false,
	}), &gorm.Config{
		Logger: log,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := migration(db); err != nil {
		return nil, err
	}

	return db, nil
}

func migration(db *gorm.DB) error {
	return db.AutoMigrate(
		new(model.User),
		new(model.Session),
		new(model.Message),
	)
}
