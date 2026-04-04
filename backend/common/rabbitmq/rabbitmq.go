package rabbitmq

import (
	"fmt"
	"log"

	"github.com/milabo0718/offer-pilot/backend/config"
	"github.com/streadway/amqp"
)

type RabbitMQ struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	Exchange string
	Key      string
}

func NewRabbitMQConnection(conf *config.RabbitmqConfig) (*amqp.Connection, error) {
	mqUrl := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/%s",
		conf.RabbitmqUsername, conf.RabbitmqPassword, conf.RabbitmqHost, conf.RabbitmqPort, conf.RabbitmqVhost,
	)
	log.Println("mqUrl is  " + mqUrl)
	var err error
	conn, err := amqp.Dial(mqUrl)
	if err != nil {
		log.Fatalf("RabbitMQ connection failed: %v", err)
	}
	return conn, nil
}

func NewRabbitMQ(conn *amqp.Connection, key string) (*RabbitMQ, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	return &RabbitMQ{
		conn:     conn,
		channel:  channel,
		Exchange: "",
		Key:      key,
	}, nil
}

func (r *RabbitMQ) Destory() {
	if r.channel != nil {
		r.channel.Close()
	}
}

func (r *RabbitMQ) Publish(message []byte) error {
	// 创建队列（不存在时）
	// 使用默认交换机的情况下，queue即为key
	_, err := r.channel.QueueDeclare(r.Key, false, false, false, false, nil)
	if err != nil {
		return err
	}

	// 调用 channel 发送消息到队列
	return r.channel.Publish(r.Exchange, r.Key, false, false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        message,
		},
	)
}

// Consume 消费者
// handle: 消息的消费业务函数，用于消费消息
func (r *RabbitMQ) Consume(handle func(msg *amqp.Delivery) error) {
	// 创建队列
	q, err := r.channel.QueueDeclare(r.Key, false, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	// 接收消息
	msgs, err := r.channel.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	// 处理消息
	for msg := range msgs {
		if err := handle(&msg); err != nil {
			fmt.Println(err.Error())
		}
	}
}
