package main

import (
	"fmt"
	"log"

	"github.com/milabo0718/offer-pilot/backend/common/email"
	"github.com/milabo0718/offer-pilot/backend/common/mysql"
	"github.com/milabo0718/offer-pilot/backend/common/redis"
	"github.com/milabo0718/offer-pilot/backend/config"
	usercontroller "github.com/milabo0718/offer-pilot/backend/controller/user"
	userdao "github.com/milabo0718/offer-pilot/backend/dao/user"
	"github.com/milabo0718/offer-pilot/backend/router"
	userservice "github.com/milabo0718/offer-pilot/backend/service/user"
	"github.com/milabo0718/offer-pilot/backend/utils/myjwt"

	"gorm.io/gorm"
)

type App struct {
	DB          *gorm.DB
	RedisStore  *redis.RedisStore
	JWTManager  *myjwt.JWTManager
	EmailSender *email.EmailSender
}

func startServer(host string, port int, app *App) error {
	// 用户相关的DAO、Service、Controller初始化
	userDao := userdao.NewUserDao(app.DB)

	userService := userservice.NewUserService(userDao, app.RedisStore, app.JWTManager, app.EmailSender)

	userController := usercontroller.NewUserController(userService)

	// Session相关的DAO、Service、Controller初始化
	// sessionDao := session.NewSessionDao(app.DB)

	// sessionService := session.NewSessionService(sessionDao)

	// sessionController := session.NewSessionController(sessionService)
	// 初始化路由
	r := router.InitRouter(userController)

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("Server is running on %s", addr)
	return r.Run(addr)
}

func main() {
	// 加载配置文件
	conf, err := config.InitConfig("./config")
	if err != nil {
		panic("系统启动失败：配置无法加载")
	}

	// 初始化Mysql
	db, err := mysql.NewDB(&conf.MysqlConfig)
	if err != nil {
		log.Fatalf("MySQL 连接失败: %v", err)
	}

	// 初始化redis
	rdbClient, err := redis.NewRedisClient(&conf.RedisConfig)
	if err != nil {
		log.Fatalf("Redis 连接失败: %v", err)
	}

	redisStore := redis.NewRedisStore(rdbClient, conf.RedisConfig.CaptchaPrefix)

	// 初始化JWT管理器
	jwtManager := myjwt.NewJWTManager(&conf.JwtConfig)

	// 初始化Email发送器
	emailSender := email.NewEmailSender(&conf.EmailConfig)

	app := &App{
		DB:          db,
		RedisStore:  redisStore,
		JWTManager:  jwtManager,
		EmailSender: emailSender,
	}

	host := conf.MainConfig.Host
	port := conf.MainConfig.Port

	err = startServer(host, port, app)
	if err != nil {
		panic(err)
	}
}
