package rabbitmq

import (
	"context"
	"encoding/json"
	"log"

	"github.com/milabo0718/offer-pilot/backend/dao/message"
	"github.com/milabo0718/offer-pilot/backend/model"

	"github.com/streadway/amqp"
)

type MessageMQParam struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	UserName  string `json:"user_name"`
	IsUser    bool   `json:"is_user"`
}

func GenerateMessageMQParam(sessionID string, content string, userName string, IsUser bool) []byte {
	param := MessageMQParam{
		SessionID: sessionID,
		Content:   content,
		UserName:  userName,
		IsUser:    IsUser,
	}
	data, _ := json.Marshal(param)
	return data
}

type MessageConsumer struct {
	messageDao *message.MessageDao
	worker     *RabbitMQ
}

func NewMessageConsumer(messageDao *message.MessageDao, worker *RabbitMQ) *MessageConsumer {
	return &MessageConsumer{
		messageDao: messageDao,
		worker:     worker,
	}
}

func (c *MessageConsumer) HandleMessage(msg *amqp.Delivery) error {
	var param MessageMQParam
	err := json.Unmarshal(msg.Body, &param)
	if err != nil {
		return err
	}
	newMsg := &model.Message{
		SessionID: param.SessionID,
		Content:   param.Content,
		UserName:  param.UserName,
		IsUser:    param.IsUser,
	}
	//消费者异步插入到数据库中
	ctx := context.Background()
	_, err = c.messageDao.CreateMessage(ctx, newMsg)
	if err != nil {
		log.Printf("消费者保存消息到数据库失败: %v", err)
		return err
	}
	return nil
}

func (c *MessageConsumer) Start() {
	c.worker.Consume(c.HandleMessage)
}
