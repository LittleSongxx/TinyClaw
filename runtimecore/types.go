package runtimecore

import (
	"context"

	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/tooling"
)

type Mode string

const (
	ModeChat     Mode = "chat"
	ModeTask     Mode = "task"
	ModeMCP      Mode = "mcp"
	ModeSkill    Mode = "skill"
	ModeWorkflow Mode = "workflow"
)

type RunRequest struct {
	Ctx              context.Context     `json:"-"`
	Mode             Mode                `json:"mode"`
	Input            string              `json:"input"`
	UserID           string              `json:"user_id"`
	ChatID           string              `json:"chat_id,omitempty"`
	MsgID            string              `json:"msg_id,omitempty"`
	ReplayOf         int64               `json:"replay_of,omitempty"`
	SkillID          string              `json:"skill_id,omitempty"`
	PerMsgLen        int                 `json:"per_msg_len,omitempty"`
	Cs               *param.ContextState `json:"-"`
	MessageChan      chan *param.MsgInfo `json:"-"`
	HTTPMsgChan      chan string         `json:"-"`
	Images           [][]byte            `json:"-"`
	ContentParameter map[string]string   `json:"content_parameter,omitempty"`
	UseRecall        *bool               `json:"use_recall,omitempty"`
	ToolBroker       *tooling.Broker     `json:"-"`
}

type RunResult struct {
	Run        *db.AgentRun `json:"run,omitempty"`
	Output     string       `json:"output,omitempty"`
	Mode       Mode         `json:"mode"`
	UsedRecall bool         `json:"used_recall,omitempty"`
}

type RunStreamEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

type ToolInventory struct {
	Count        int                `json:"count"`
	RuntimeCount int                `json:"runtime_count"`
	LegacyCount  int                `json:"legacy_count"`
	Tools        []tooling.ToolSpec `json:"tools"`
	RuntimeTools []tooling.ToolSpec `json:"runtime_tools,omitempty"`
	LegacyTools  []tooling.ToolSpec `json:"legacy_tools,omitempty"`
}

type SkillDescriptor struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Modes        []string `json:"modes,omitempty"`
	Triggers     []string `json:"triggers,omitempty"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
	Memory       string   `json:"memory,omitempty"`
	Priority     int      `json:"priority,omitempty"`
	Legacy       bool     `json:"legacy"`
	Path         string   `json:"path"`
}

type SkillsStatus struct {
	Count    int                    `json:"count"`
	Warnings []string               `json:"warnings,omitempty"`
	Servers  interface{}            `json:"servers,omitempty"`
	Skills   []SkillDescriptor      `json:"skills"`
	ByMode   map[string]int         `json:"by_mode,omitempty"`
	Extra    map[string]interface{} `json:"extra,omitempty"`
}
