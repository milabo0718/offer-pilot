package aihelper

import (
	"context"

	"github.com/cloudwego/eino/schema"
)


func ParseTextWithModel(ctx context.Context, modelType string, prompt string, config map[string]interface{}) (string, error) {
	if modelType == "" {
		modelType = "1"
	}
	if config == nil {
		config = map[string]interface{}{}
	}

	factory := NewAIModelFactory()
	aiModel, err := factory.CreateAIModel(ctx, modelType, config)
	if err != nil {
		return "", err
	}

	resp, err := aiModel.GenerateResponse(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: prompt,
		},
	})
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}
