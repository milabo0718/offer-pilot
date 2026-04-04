package config

import (
	"log"

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

type Config struct {
	MainConfig     `mapstructure:"mainConfig"`
	EmailConfig    `mapstructure:"emailConfig"`
	RedisConfig    `mapstructure:"redisConfig"`
	MysqlConfig    `mapstructure:"mysqlConfig"`
	JwtConfig      `mapstructure:"jwtConfig"`
	RabbitmqConfig `mapstructure:"rabbitmqConfig"`
}

func InitConfig(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
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
