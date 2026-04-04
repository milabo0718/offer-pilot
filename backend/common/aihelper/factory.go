package aihelper

import (
	"context"
	"fmt"
)

// 这是一个简单的工厂模式，用于创造不同类型的AI模型实例。
type ModelCreator func(ctx context.Context, config map[string]any) (AIModel, error)

type AIModelFactory struct {
	creators map[string]ModelCreator
}

func NewAIModelFactory() *AIModelFactory {
	factory := &AIModelFactory{
		creators: make(map[string]ModelCreator),
	}
	factory.registerCreators()
	return factory
}

func (f *AIModelFactory) registerCreators() {
	// openai模型的创建函数
	f.creators["1"] = func(ctx context.Context, config map[string]any) (AIModel, error) {
		return NewOpenAIModel(ctx)
	}

	f.creators["2"] = func(ctx context.Context, config map[string]any) (AIModel, error) {
		baseURL, _ := config["baseURL"].(string)
		modelName, ok := config["modelName"].(string)
		if !ok {
			return nil, fmt.Errorf("Ollama model requires modelName")
		}
		return NewOllamaModel(ctx, baseURL, modelName)
	}
}

func (f *AIModelFactory) CreateAIModel(ctx context.Context, modelType string, config map[string]interface{}) (AIModel, error) {
	creator, ok := f.creators[modelType]
	if !ok {
		return nil, fmt.Errorf("unsupported model type: %s", modelType)
	}
	return creator(ctx, config)
}

// RegisterModel 可扩展注册
func (f *AIModelFactory) RegisterModel(modelType string, creator ModelCreator) {
	f.creators[modelType] = creator
}
