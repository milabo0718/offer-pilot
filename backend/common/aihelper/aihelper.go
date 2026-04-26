package aihelper

import (
	"context"
	"sync"

	"github.com/cloudwego/eino/schema"
	"github.com/milabo0718/offer-pilot/backend/model"
	"github.com/milabo0718/offer-pilot/backend/utils"
)

// 这里一个用户的一个会话对应一个AIHelper，AIHelper里包含了这个会话的历史消息和使用的模型等信息
type AIHelper struct {
	model        AIModel
	messages     []*model.Message
	mu           sync.RWMutex
	SessionID    string
	saveFunc     func(*model.Message) error
	systemPrompt string
}

func NewAIHelper(model_ AIModel, sessionID string, saveFunc func(*model.Message) error) *AIHelper {
	return &AIHelper{
		model:     model_,
		messages:  make([]*model.Message, 0),
		saveFunc:  saveFunc,
		SessionID: sessionID,
	}
}

func (a *AIHelper) AddMessage(Content string, UserName string, IsUser bool, Save bool) {
	userMsg := model.Message{
		SessionID: a.SessionID,
		Content:   Content,
		UserName:  UserName,
		IsUser:    IsUser,
	}
	a.messages = append(a.messages, &userMsg)
	if Save {
		a.saveFunc(&userMsg)
	}
}

// SaveMessage 保存消息到数据库（通过回调函数避免循环依赖）
func (a *AIHelper) SetSaveFunc(saveFunc func(*model.Message) error) {
	a.saveFunc = saveFunc
}

// SetSystemPrompt 设置当前会话的系统提示词（面试官人格 + 岗位画像）。
// 不会进入消息历史/数据库，仅在每次调用模型时作为首条 system 消息注入。
func (a *AIHelper) SetSystemPrompt(prompt string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.systemPrompt = prompt
}

// buildSchemaMessages 构造发送给模型的消息列表，若设置了 systemPrompt 则前置为 system 消息。
func (a *AIHelper) buildSchemaMessages() []*schema.Message {
	msgs := utils.ConvertToSchemaMessages(a.messages)
	if a.systemPrompt == "" {
		return msgs
	}
	out := make([]*schema.Message, 0, len(msgs)+1)
	out = append(out, &schema.Message{Role: schema.System, Content: a.systemPrompt})
	out = append(out, msgs...)
	return out
}

// GetMessages 获取所有消息历史
func (a *AIHelper) GetMessages() []*model.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]*model.Message, len(a.messages))
	copy(out, a.messages)
	return out
}

// 同步生成
func (a *AIHelper) GenerateResponse(userName string, ctx context.Context, userQuestion string) (*model.Message, error) {
	//调用存储函数
	a.AddMessage(userQuestion, userName, true, true)

	a.mu.RLock()
	//将model.Message转化成schema.Message，并在首位注入 system 提示词
	messages := a.buildSchemaMessages()
	a.mu.RUnlock()

	//调用模型生成回复
	schemaMsg, err := a.model.GenerateResponse(ctx, messages)
	if err != nil {
		return nil, err
	}

	//将schema.Message转化成model.Message
	modelMsg := utils.ConvertToModelMessage(a.SessionID, userName, schemaMsg)

	//调用存储函数
	a.AddMessage(modelMsg.Content, userName, false, true)

	return modelMsg, nil
}

// 流式生成
func (a *AIHelper) StreamResponse(userName string, ctx context.Context, cb StreamCallback, userQuestion string) (*model.Message, error) {

	//调用存储函数
	a.AddMessage(userQuestion, userName, true, true)

	a.mu.RLock()
	messages := a.buildSchemaMessages()
	a.mu.RUnlock()

	content, err := a.model.StreamResponse(ctx, messages, cb)
	if err != nil {
		return nil, err
	}
	//转化成model.Message
	modelMsg := &model.Message{
		SessionID: a.SessionID,
		UserName:  userName,
		Content:   content,
		IsUser:    false,
	}

	//调用存储函数
	a.AddMessage(modelMsg.Content, userName, false, true)

	return modelMsg, nil
}

// GetModelType 获取模型类型
func (a *AIHelper) GetModelType() string {
	return a.model.GetModelType()
}
