package aihelper

import (
	"context"
	"sync"

	"github.com/milabo0718/offer-pilot/backend/common/rabbitmq"
	"github.com/milabo0718/offer-pilot/backend/model"
)

type AIHelperManager struct {
	helpers       map[string]map[string]*AIHelper // SessionID -> modelType -> AIHelper
	mu            sync.RWMutex
	factory       *AIModelFactory
	rabbitPublish *rabbitmq.RabbitMQ
}

func NewAIHelperManager(factory *AIModelFactory, rabbitPublish *rabbitmq.RabbitMQ) *AIHelperManager {
	return &AIHelperManager{
		helpers:       make(map[string]map[string]*AIHelper),
		factory:       factory,
		rabbitPublish: rabbitPublish,
	}
}

func (m *AIHelperManager) GetOrCreateAIHelper(ctx context.Context, userName string, sessionID string, modelType string, config map[string]interface{}) (*AIHelper, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 获取用户的会话映射
	userHelpers, exists := m.helpers[userName]
	if !exists {
		userHelpers = make(map[string]*AIHelper)
		m.helpers[userName] = userHelpers
	}

	// 检查会话是否已存在
	helper, exists := userHelpers[sessionID]
	if exists {
		return helper, nil
	}

	model_, err := m.factory.CreateAIModel(ctx, modelType, config)
	if err != nil {
		return nil, err
	}

	saveFunc := func(msg *model.Message) error {
		data := rabbitmq.GenerateMessageMQParam(msg.SessionID, msg.Content, msg.UserName, msg.IsUser)
		return m.rabbitPublish.Publish(data) // 使用注入的 MQ 发送
	}

	userHelpers[sessionID] = NewAIHelper(model_, sessionID, saveFunc)
	return userHelpers[sessionID], nil
}

// 获取指定用户的指定会话的AIHelper
func (m *AIHelperManager) GetAIHelper(userName string, sessionID string) (*AIHelper, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userHelpers, exists := m.helpers[userName]
	if !exists {
		return nil, false
	}

	helper, exists := userHelpers[sessionID]
	return helper, exists
}

// 移除指定用户的指定会话的AIHelper
func (m *AIHelperManager) RemoveAIHelper(userName string, sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	userHelpers, exists := m.helpers[userName]
	if !exists {
		return
	}

	delete(userHelpers, sessionID)

	// 如果用户没有会话了，清理用户映射
	if len(userHelpers) == 0 {
		delete(m.helpers, userName)
	}
}

// 获取指定用户的所有会话ID
func (m *AIHelperManager) GetUserSessions(userName string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userHelpers, exists := m.helpers[userName]
	if !exists {
		return []string{}
	}

	sessionIDs := make([]string, 0, len(userHelpers))
	//取出所有的key
	for sessionID := range userHelpers {
		sessionIDs = append(sessionIDs, sessionID)
	}

	return sessionIDs
}
