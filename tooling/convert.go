package tooling

import (
	"github.com/cohesion-org/deepseek-go"
	"github.com/sashabaranov/go-openai"
)

func ToOpenAITools(specs []ToolSpec) []openai.Tool {
	if len(specs) == 0 {
		return nil
	}

	tools := make([]openai.Tool, 0, len(specs))
	for _, spec := range specs {
		tool := openai.Tool{
			Type: "function",
			Function: &openai.FunctionDefinition{
				Name:        spec.Name,
				Description: spec.Description,
			},
		}

		if schema, ok := spec.InputSchema.(map[string]interface{}); ok && len(schema) > 0 {
			tool.Function.Parameters = schema
		}
		tools = append(tools, tool)
	}
	return tools
}

func ToDeepseekTools(specs []ToolSpec) []deepseek.Tool {
	if len(specs) == 0 {
		return nil
	}

	tools := make([]deepseek.Tool, 0, len(specs))
	for _, spec := range specs {
		tool := deepseek.Tool{
			Type: "function",
			Function: deepseek.Function{
				Name:        spec.Name,
				Description: spec.Description,
			},
		}

		if schema, ok := spec.InputSchema.(map[string]interface{}); ok && len(schema) > 0 {
			parameters := &deepseek.FunctionParameters{}
			if schemaType, ok := schema["type"].(string); ok {
				parameters.Type = schemaType
			}
			if properties, ok := schema["properties"].(map[string]interface{}); ok {
				parameters.Properties = properties
			}
			switch required := schema["required"].(type) {
			case []string:
				parameters.Required = required
			case []interface{}:
				items := make([]string, 0, len(required))
				for _, item := range required {
					if value, ok := item.(string); ok {
						items = append(items, value)
					}
				}
				parameters.Required = items
			}
			tool.Function.Parameters = parameters
		}
		tools = append(tools, tool)
	}
	return tools
}
