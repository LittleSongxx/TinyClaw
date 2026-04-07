package agentruntime

import "testing"

func TestExtractJSONObject(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "plain_json",
			input:  `{"plan":[{"name":"search","description":"look up docs"}]}`,
			expect: `{"plan":[{"name":"search","description":"look up docs"}]}`,
		},
		{
			name:   "json_with_prefix_suffix",
			input:  "Here is the result:\n```json\n{\"agent\":\"browser\"}\n```\nThanks",
			expect: `{"agent":"browser"}`,
		},
		{
			name:   "json_with_braces_in_string",
			input:  `before {"plan":[{"name":"writer","description":"summarize {nested} content"}]} after`,
			expect: `{"plan":[{"name":"writer","description":"summarize {nested} content"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractJSONObject(tt.input)
			if err != nil {
				t.Fatalf("ExtractJSONObject failed: %v", err)
			}
			if got != tt.expect {
				t.Fatalf("expected %s, got %s", tt.expect, got)
			}
		})
	}
}

func TestExtractJSONObject_NoJSON(t *testing.T) {
	if _, err := ExtractJSONObject("no json here"); err == nil {
		t.Fatal("expected error when no JSON object exists")
	}
}
