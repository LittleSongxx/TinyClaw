import test from "node:test";
import assert from "node:assert/strict";
import { getSkillDescription, getSkillSection } from "./skillDescriptions.js";

test("returns localized zh description for mapped local skill", () => {
    const skill = {
        source: "local",
        manifest: {
            id: "general_research",
            description: "Research a topic with evidence from web, academic, time, and memory tools before answering.",
        },
        sections: {
            when_to_use: "Use this skill for research-heavy questions.",
        },
    };

    assert.equal(
        getSkillDescription(skill, "zh"),
        "在输出答案前，使用网页、学术、时间和记忆工具对主题进行证据化研究。"
    );
    assert.match(getSkillSection(skill, "when_to_use", "zh"), /研究型问题/);
});

test("returns zh description for legacy proxy by reusing mcp descriptions", () => {
    const skill = {
        source: "legacy",
        path: "legacy://amap",
        manifest: {
            id: "legacy_amap_proxy",
            description: "Gets geolocation and map data from AMap. Requires AMAP_MAPS_API_KEY in the app environment.",
        },
        sections: {
            instructions: "Act as a compatibility proxy.",
        },
    };

    assert.equal(
        getSkillDescription(skill, "zh"),
        "从高德地图服务获取地理信息。"
    );
    assert.match(getSkillSection(skill, "instructions", "zh"), /amap MCP 服务/);
});

test("falls back to english when no zh mapping exists", () => {
    const skill = {
        source: "local",
        manifest: {
            id: "future_skill",
            description: "A future skill without zh mapping.",
        },
        sections: {
            output_contract: "Return the result.",
        },
    };

    assert.equal(getSkillDescription(skill, "zh"), "A future skill without zh mapping.");
    assert.equal(getSkillSection(skill, "output_contract", "zh"), "Return the result.");
});

test("supports special legacy_all_tools_proxy copy", () => {
    const skill = {
        source: "legacy",
        path: "legacy://all_tools",
        manifest: {
            id: "legacy_all_tools_proxy",
            description: "Catch-all compatibility skill that exposes every currently registered MCP tool.",
        },
        sections: {
            when_to_use: "Use this skill only as a fallback.",
        },
    };

    assert.equal(getSkillDescription(skill, "zh"), "兜底兼容 skill，暴露当前已注册的全部 MCP 工具。");
    assert.match(getSkillSection(skill, "when_to_use", "zh"), /兜底 skill/);
});

