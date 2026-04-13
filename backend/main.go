package main

import (
	"fmt"
	"log"

	"github.com/milabo0718/offer-pilot/backend/common/aihelper"
	"github.com/milabo0718/offer-pilot/backend/common/email"
	"github.com/milabo0718/offer-pilot/backend/common/mysql"
	"github.com/milabo0718/offer-pilot/backend/common/rabbitmq"
	"github.com/milabo0718/offer-pilot/backend/common/redis"
	"github.com/milabo0718/offer-pilot/backend/config"
	sessioncontroller "github.com/milabo0718/offer-pilot/backend/controller/session"
	usercontroller "github.com/milabo0718/offer-pilot/backend/controller/user"
	messagedao "github.com/milabo0718/offer-pilot/backend/dao/message"
	sessiondao "github.com/milabo0718/offer-pilot/backend/dao/session"
	userdao "github.com/milabo0718/offer-pilot/backend/dao/user"
	"github.com/milabo0718/offer-pilot/backend/router"
	sessionservice "github.com/milabo0718/offer-pilot/backend/service/session"
	userservice "github.com/milabo0718/offer-pilot/backend/service/user"
	"github.com/milabo0718/offer-pilot/backend/utils/myjwt"

	"gorm.io/gorm"
)

type App struct {
	DB          *gorm.DB
	RedisStore  *redis.RedisStore
	JWTManager  *myjwt.JWTManager
	EmailSender *email.EmailSender
	AiManager   *aihelper.AIHelperManager
}

func startServer(host string, port int, app *App) error {
	// 用户相关的DAO、Service、Controller初始化
	userDao := userdao.NewUserDao(app.DB)

	userService := userservice.NewUserService(userDao, app.RedisStore, app.JWTManager, app.EmailSender)

	userController := usercontroller.NewUserController(userService)

	// Session相关的DAO、Service、Controller初始化
	sessionDao := sessiondao.NewSessionDao(app.DB)
	messageDao := messagedao.NewMessageDao(app.DB)

	sessionService := sessionservice.NewSessionService(sessionDao, messageDao, app.AiManager)

	sessionController := sessioncontroller.NewSessionController(sessionService)
	// 初始化路由
	r := router.InitRouter(userController, sessionController, app.JWTManager)

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

	// 初始化RabbitMQ连接
	rabbitMQConn, err := rabbitmq.NewRabbitMQConnection(&conf.RabbitmqConfig)
	if err != nil {
		log.Fatalf("RabbitMQ 连接失败: %v", err)
	}
	defer rabbitMQConn.Close()

	messagePublisher, err := rabbitmq.NewRabbitMQ(rabbitMQConn, "Message")
	if err != nil {
		log.Fatalf("RabbitMQ 初始化失败: %v", err)
	}

	messageConsumerWorker, err := rabbitmq.NewRabbitMQ(rabbitMQConn, "Message")
	if err != nil {
		log.Fatalf("创建 MQ Consumer 失败: %v", err)
	}

	messageDao := messagedao.NewMessageDao(db)

	msgConsumer := rabbitmq.NewMessageConsumer(messageDao, messageConsumerWorker)

	go msgConsumer.Start()
	// 初始化AI助手管理器
	factory := aihelper.NewAIModelFactory()

	aiManager := aihelper.NewAIHelperManager(factory, messagePublisher)
	app := &App{
		DB:          db,
		RedisStore:  redisStore,
		JWTManager:  jwtManager,
		EmailSender: emailSender,
		AiManager:   aiManager,
	}

	host := conf.MainConfig.Host
	port := conf.MainConfig.Port

	err = startServer(host, port, app)
	if err != nil {
		panic(err)
	}
}
