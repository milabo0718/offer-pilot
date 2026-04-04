package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/milabo0718/offer-pilot/backend/config"

	"github.com/go-redis/redis/v8"
)

// redis客户端初始化函数
func NewRedisClient(conf *config.RedisConfig) (*redis.Client, error) {
	host := conf.RedisHost
	port := conf.RedisPort
	password := conf.RedisPassword
	db := conf.RedisDb
	addr := host + ":" + strconv.Itoa(port)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return client, nil
}

type RedisStore struct {
	client        *redis.Client
	captchaPrefix string
}

func NewRedisStore(client *redis.Client, captchaPrefix string) *RedisStore {
	if captchaPrefix == "" {
		captchaPrefix = "captcha:%s"
	}
	return &RedisStore{
		client:        client,
		captchaPrefix: captchaPrefix,
	}
}

func (r *RedisStore) generateCaptcha(email string) string {
	return fmt.Sprintf(r.captchaPrefix, email)
}

func (r *RedisStore) SetCaptchaForEmail(ctx context.Context, email, captcha string) error {
	key := r.generateCaptcha(email)
	expire := 2 * time.Minute
	return r.client.Set(ctx, key, captcha, expire).Err()
}

func (r *RedisStore) CheckCaptchaForEmail(ctx context.Context, email, userInput string) (bool, error) {
	key := r.generateCaptcha(email)

	storedCaptcha, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	if strings.EqualFold(storedCaptcha, userInput) {
		// 验证成功后删除 key，同样使用传进来的 ctx
		if err := r.client.Del(ctx, key).Err(); err != nil {
			// 可以记录日志
		}
		return true, nil
	}

	return false, nil
}
