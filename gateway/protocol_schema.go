package gateway

import "encoding/json"

func ProtocolJSONSchema() map[string]interface{} {
	return map[string]interface{}{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id":     "https://tinyclaw.local/schemas/gateway-protocol-v1.json",
		"title":   "TinyClaw Typed Gateway Protocol v1",
		"type":    "object",
		"oneOf": []interface{}{
			frameSchema("connect", connectSchema()),
			frameSchema("req", requestSchema()),
			frameSchema("res", responseSchema()),
			frameSchema("event", eventSchema()),
		},
	}
}

func ProtocolJSONSchemaBytes() ([]byte, error) {
	return json.MarshalIndent(ProtocolJSONSchema(), "", "  ")
}

func frameSchema(frameType string, props map[string]interface{}) map[string]interface{} {
	props["type"] = map[string]interface{}{"const": frameType}
	return map[string]interface{}{
		"type":                 "object",
		"additionalProperties": true,
		"properties":           props,
		"required":             []interface{}{"type"},
	}
}

func connectSchema() map[string]interface{} {
	return map[string]interface{}{
		"protocol_version": map[string]interface{}{"const": ProtocolVersionV1},
		"role":             map[string]interface{}{"enum": []interface{}{"control", "node"}},
		"workspace_id":     map[string]interface{}{"type": "string"},
		"auth": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"type":        map[string]interface{}{"type": "string"},
				"token":       map[string]interface{}{"type": "string"},
				"device_id":   map[string]interface{}{"type": "string"},
				"nonce":       map[string]interface{}{"type": "string"},
				"signature":   map[string]interface{}{"type": "string"},
				"public_key":  map[string]interface{}{"type": "string"},
				"actor_token": map[string]interface{}{"type": "string"},
			},
			"required": []interface{}{"type"},
		},
		"timestamp": map[string]interface{}{"type": "integer"},
	}
}

func requestSchema() map[string]interface{} {
	return map[string]interface{}{
		"id":              map[string]interface{}{"type": "string"},
		"method":          map[string]interface{}{"type": "string"},
		"params":          map[string]interface{}{},
		"idempotency_key": map[string]interface{}{"type": "string"},
		"workspace_id":    map[string]interface{}{"type": "string"},
		"timestamp":       map[string]interface{}{"type": "integer"},
	}
}

func responseSchema() map[string]interface{} {
	return map[string]interface{}{
		"id":     map[string]interface{}{"type": "string"},
		"ok":     map[string]interface{}{"type": "boolean"},
		"result": map[string]interface{}{},
		"error": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"code":    map[string]interface{}{"type": "string"},
				"message": map[string]interface{}{"type": "string"},
			},
			"required": []interface{}{"code", "message"},
		},
		"timestamp": map[string]interface{}{"type": "integer"},
	}
}

func eventSchema() map[string]interface{} {
	return map[string]interface{}{
		"event":         map[string]interface{}{"type": "string"},
		"payload":       map[string]interface{}{},
		"seq":           map[string]interface{}{"type": "integer"},
		"state_version": map[string]interface{}{"type": "integer"},
		"timestamp":     map[string]interface{}{"type": "integer"},
	}
}
