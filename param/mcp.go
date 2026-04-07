package param

import mcpParam "github.com/LittleSongxx/mcp-client-go/clients/param"

type MCPAvailability struct {
	Statuses   []string          `json:"statuses"`
	Notes      map[string]string `json:"notes"`
	Registered bool              `json:"registered,omitempty"`
}

type MCPInspectData struct {
	McpServers   map[string]*mcpParam.MCPConfig `json:"mcpServers"`
	Availability map[string]*MCPAvailability    `json:"availability"`
}
