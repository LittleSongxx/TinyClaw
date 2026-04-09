package runtimecore

import "testing"

func TestShouldPreferRuntimeToolsForNodeInventory(t *testing.T) {
	cases := []string{
		"请列出当前可用的电脑节点",
		"列一下在线节点",
		"show me the connected devices",
		"list nodes",
	}

	for _, input := range cases {
		if !shouldPreferRuntimeTools(input) {
			t.Fatalf("expected runtime tools for %q", input)
		}
	}
}

func TestShouldPreferRuntimeToolsForDesktopActions(t *testing.T) {
	cases := []string{
		"请直接截取当前 Windows 桌面，并把截图发给我",
		"打开记事本并输入 hello",
		"focus the browser window",
		"read the current file from the desktop",
		"/MCP list",
		"查询哈尔滨工业大学坐标",
	}

	for _, input := range cases {
		if !shouldPreferRuntimeTools(input) {
			t.Fatalf("expected runtime tools for %q", input)
		}
	}
}

func TestShouldPreferRuntimeToolsIgnoresKnowledgeQuestion(t *testing.T) {
	cases := []string{
		"请总结一下 A2A 协议的核心特性",
		"knowledge base 里有哪些关于 agent discovery 的内容",
	}

	for _, input := range cases {
		if shouldPreferRuntimeTools(input) {
			t.Fatalf("did not expect runtime tools for %q", input)
		}
	}
}

func TestDetermineRecallUsageHonorsExplicitFlag(t *testing.T) {
	value := true
	if !determineRecallUsage(&value, "请列出当前可用的电脑节点") {
		t.Fatalf("expected explicit true to force recall")
	}

	value = false
	if determineRecallUsage(&value, "请总结一下 A2A 协议") {
		t.Fatalf("expected explicit false to disable recall")
	}
}
