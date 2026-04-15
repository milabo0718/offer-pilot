package main

import (
	"context"
	"fmt"
	"log"

	"github.com/milabo0718/offer-pilot/backend/common/aihelper"
	"github.com/milabo0718/offer-pilot/backend/common/email"
	"github.com/milabo0718/offer-pilot/backend/common/mysql"
	"github.com/milabo0718/offer-pilot/backend/common/rabbitmq"
	"github.com/milabo0718/offer-pilot/backend/common/rag/embedder"
	"github.com/milabo0718/offer-pilot/backend/common/rag/store"
	"github.com/milabo0718/offer-pilot/backend/common/redis"
	commonstt "github.com/milabo0718/offer-pilot/backend/common/stt"
	commontts "github.com/milabo0718/offer-pilot/backend/common/tts"
	"github.com/milabo0718/offer-pilot/backend/config"
	ragcontroller "github.com/milabo0718/offer-pilot/backend/controller/rag"
	sessioncontroller "github.com/milabo0718/offer-pilot/backend/controller/session"
	sttcontroller "github.com/milabo0718/offer-pilot/backend/controller/stt"
	ttscontroller "github.com/milabo0718/offer-pilot/backend/controller/tts"
	usercontroller "github.com/milabo0718/offer-pilot/backend/controller/user"
	messagedao "github.com/milabo0718/offer-pilot/backend/dao/message"
	sessiondao "github.com/milabo0718/offer-pilot/backend/dao/session"
	userdao "github.com/milabo0718/offer-pilot/backend/dao/user"
	"github.com/milabo0718/offer-pilot/backend/router"
	ragservice "github.com/milabo0718/offer-pilot/backend/service/rag"
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
	RAGCtrl     *ragcontroller.RAGController
	RAGService  *ragservice.Service
	RAGEnabled  bool
	RAGTopK     int
	TTSCtrl     *ttscontroller.TTSController
	STTCtrl     *sttcontroller.STTController
}

func startServer(host string, port int, app *App) error {
	// 用户相关的DAO、Service、Controller初始化
	userDao := userdao.NewUserDao(app.DB)

	userService := userservice.NewUserService(userDao, app.RedisStore, app.JWTManager, app.EmailSender)

	userController := usercontroller.NewUserController(userService)

	// Session相关的DAO、Service、Controller初始化
	sessionDao := sessiondao.NewSessionDao(app.DB)

	sessionService := sessionservice.NewSessionService(sessionDao, app.AiManager)
	sessionService.ConfigureRAGAugment(app.RAGService, app.RAGEnabled, app.RAGTopK)

	sessionController := sessioncontroller.NewSessionController(sessionService)
	// 初始化路由
	r := router.InitRouter(userController, sessionController, app.RAGCtrl, app.TTSCtrl, app.STTCtrl, app.JWTManager)

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

	var ragCtrl *ragcontroller.RAGController
	var ragSvc *ragservice.Service
	if conf.RagConfig.Enabled {
		emb, embErr := embedder.NewFromConfig(conf.RagConfig)
		if embErr != nil {
			log.Printf("[RAG] Embedding 初始化失败，已降级禁用 RAG: %v", embErr)
		} else {
			ragRedisHost, ragRedisPort := conf.RagConfig.RedisHostPort()
			ragRedisClient, redisErr := redis.NewRedisClient(&config.RedisConfig{
				RedisHost:     ragRedisHost,
				RedisPort:     ragRedisPort,
				RedisPassword: conf.RagConfig.RedisPassword,
				RedisDb:       conf.RagConfig.RedisDB,
			})
			if redisErr != nil {
				log.Printf("[RAG] Redis 客户端初始化失败，已降级禁用 RAG: %v", redisErr)
			} else {
				rvStore := store.NewRedisVectorStore(ragRedisClient, conf.RagConfig)
				ingestService := ragservice.NewIngestService(emb, rvStore, conf.RagConfig.BatchSize)
				searchService := ragservice.NewSearchService(emb, rvStore, conf.RagConfig.DefaultTopK, conf.RagConfig.MaxTopK)
				ragSvc = ragservice.NewService(ingestService, searchService)
				ragCtrl = ragcontroller.NewRAGController(ingestService, searchService, conf.RagConfig.DefaultIngestDir)

				health := ingestService.Health(context.Background())
				log.Printf("[RAG] 启动健康检查: reachable=%v search_ready=%v index_exists=%v index=%s msg=%s",
					health.RedisReachable, health.RedisSearchReady, health.IndexExists, health.IndexName, health.Message)
				if !health.RedisSearchReady {
					log.Printf("[RAG] 提示: 当前 Redis 未就绪向量检索能力，请使用 Redis Stack 并加载 RediSearch 模块")
				}
			}
		}
	} else {
		log.Printf("[RAG] 已在配置中关闭")
	}

	// 初始化 TTS 客户端与控制器
	ttsClient := commontts.NewTTSClient(commontts.TTSConfig{
		APIKey:     conf.TTSConfig.APIKey,
		ModelName:  conf.TTSConfig.ModelName,
		Voice:      conf.TTSConfig.Voice,
		Format:     conf.TTSConfig.Format,
		SampleRate: conf.TTSConfig.SampleRate,
	})
	ttsCtrl := ttscontroller.NewTTSController(ttsClient)
	log.Printf("[TTS] 初始化完成，模型: %s，音色: %s", conf.TTSConfig.ModelName, conf.TTSConfig.Voice)

	// 初始化 STT 客户端与控制器
	sttClient := commonstt.NewSTTClient(commonstt.STTConfig{
		APIKey:    conf.STTConfig.APIKey,
		ModelName: conf.STTConfig.ModelName,
		Language:  conf.STTConfig.Language,
	})
	sttCtrl := sttcontroller.NewSTTController(sttClient)
	log.Printf("[STT] 初始化完成，模型: %s，语种: %s", conf.STTConfig.ModelName, conf.STTConfig.Language)

	app := &App{
		DB:          db,
		RedisStore:  redisStore,
		JWTManager:  jwtManager,
		EmailSender: emailSender,
		AiManager:   aiManager,
		RAGCtrl:     ragCtrl,
		RAGService:  ragSvc,
		RAGEnabled:  conf.RagConfig.ChatAugmentEnabled,
		RAGTopK:     conf.RagConfig.ChatAugmentTopK,
		TTSCtrl:     ttsCtrl,
		STTCtrl:     sttCtrl,
	}

	host := conf.MainConfig.Host
	port := conf.MainConfig.Port

	err = startServer(host, port, app)
	if err != nil {
		panic(err)
	}
}
