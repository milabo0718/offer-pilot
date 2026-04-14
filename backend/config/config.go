package config

import (
	"log"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type MainConfig struct {
	AppName string `mapstructure:"appName"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
}

type EmailConfig struct {
	AuthCode string `mapstructure:"authCode"`
	Email    string `mapstructure:"email"`
	DevMode  bool   `mapstructure:"devMode"`
}

type RedisConfig struct {
	RedisPort     int    `mapstructure:"port"`
	RedisDb       int    `mapstructure:"db"`
	RedisHost     string `mapstructure:"host"`
	RedisPassword string `mapstructure:"password"`
	CaptchaPrefix string `mapstructure:"captchaPrefix"`
}

type MysqlConfig struct {
	MysqlPort         int    `mapstructure:"port"`
	MysqlHost         string `mapstructure:"host"`
	MysqlUser         string `mapstructure:"user"`
	MysqlPassword     string `mapstructure:"password"`
	MysqlDatabaseName string `mapstructure:"databaseName"`
	MysqlCharset      string `mapstructure:"charset"`
}

type JwtConfig struct {
	ExpireDuration int    `mapstructure:"expire_duration"`
	Issuer         string `mapstructure:"issuer"`
	Subject        string `mapstructure:"subject"`
	Key            string `mapstructure:"key"`
}

type RabbitmqConfig struct {
	RabbitmqPort     int    `mapstructure:"port"`
	RabbitmqHost     string `mapstructure:"host"`
	RabbitmqUsername string `mapstructure:"username"`
	RabbitmqPassword string `mapstructure:"password"`
	RabbitmqVhost    string `mapstructure:"vhost"`
}

type RagConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	ChatAugmentEnabled bool   `mapstructure:"chatAugmentEnabled"`
	ChatAugmentTopK    int    `mapstructure:"chatAugmentTopK"`
	RedisAddr          string `mapstructure:"redisAddr"`
	RedisPassword      string `mapstructure:"redisPassword"`
	RedisDB            int    `mapstructure:"redisDB"`
	IndexName          string `mapstructure:"indexName"`
	KeyPrefix          string `mapstructure:"keyPrefix"`
	VectorField        string `mapstructure:"vectorField"`
	VectorDim          int    `mapstructure:"vectorDim"`
	DistanceMetric     string `mapstructure:"distanceMetric"`
	DefaultTopK        int    `mapstructure:"defaultTopK"`
	MaxTopK            int    `mapstructure:"maxTopK"`
	DefaultIngestDir   string `mapstructure:"defaultIngestDir"`
	BatchSize          int    `mapstructure:"batchSize"`
	EmbeddingProvider  string `mapstructure:"embeddingProvider"`
	EmbeddingAPIKey    string `mapstructure:"embeddingAPIKey"`
	EmbeddingBaseURL   string `mapstructure:"embeddingBaseURL"`
	EmbeddingModelName string `mapstructure:"embeddingModelName"`
	UseMockEmbedding   bool   `mapstructure:"useMockEmbedding"`
}

type Config struct {
	MainConfig     `mapstructure:"mainConfig"`
	EmailConfig    `mapstructure:"emailConfig"`
	RedisConfig    `mapstructure:"redisConfig"`
	MysqlConfig    `mapstructure:"mysqlConfig"`
	JwtConfig      `mapstructure:"jwtConfig"`
	RabbitmqConfig `mapstructure:"rabbitmqConfig"`
	RagConfig      `mapstructure:"ragConfig"`
}

// RedisHostPort 解析 ragConfig.redisAddr，若为空则回退到默认本地地址。
func (r RagConfig) RedisHostPort() (string, int) {
	addr := strings.TrimSpace(r.RedisAddr)
	if addr == "" {
		return "127.0.0.1", 6379
	}

	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "127.0.0.1", 6379
	}

	host := strings.TrimSpace(parts[0])
	port, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || host == "" {
		return "127.0.0.1", 6379
	}

	return host, port
}

func InitConfig(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("读取配置文件失败: %v\n", err)
		return nil, err
	}

	var conf Config
	if err := viper.Unmarshal(&conf); err != nil {
		log.Printf("解析配置文件失败: %v\n", err)
		return nil, err
	}

	log.Println("配置加载成功!")
	return &conf, nil
}
