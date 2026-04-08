import { getMcpDescription } from "./mcpDescriptions.js";

const SKILL_DESCRIPTION_ZH = {
    general_research: {
        description: "在输出答案前，使用网页、学术、时间和记忆工具对主题进行证据化研究。",
        when_to_use: "适用于研究型问题、对比分析、事实收集和需要证据支撑的总结任务。",
        when_not_to_use: "不适用于浏览器自动化、本地文件操作或仅限 GitHub 的工作流。",
        instructions: "只收集回答当前问题所需的证据，尽量优先使用直接来源，并清楚区分事实与推断。",
        output_contract: "优先给出最强证据，再补充不确定性或缺失信息，保持回答简洁。",
        failure_handling: "如果检索不完整，明确说明哪些内容无法验证，并给出当前最可靠的部分答案。",
    },
    browser_operator: {
        description: "通过浏览器相关 MCP 工具检查网页并执行交互操作。",
        when_to_use: "适用于页面检查、网页交互流程、浏览任务以及从实时网站收集证据。",
        when_not_to_use: "不适用于本地文件系统工作、纯仓库问题或单纯知识总结任务。",
        instructions: "优先采用可重复的浏览器操作，在汇报前确认关键页面状态，并保留简短的观察记录。",
        output_contract: "返回页面结果或交互结果时，提供足够上下文，让人能判断发生了什么。",
        failure_handling: "如果页面无法访问或自动化失败，明确说明阻塞原因以及最后确认的页面状态。",
    },
    workspace_operator: {
        description: "通过文件系统相关 MCP 工具检查工作区内的文件和目录。",
        when_to_use: "适用于本地文件检查、工作区分析，以及依赖文件系统证据的任务。",
        when_not_to_use: "不适用于纯浏览器流程或 GitHub API 任务。",
        instructions: "先检查再下结论，只在可访问的工作区路径内操作，并说明结论对应的具体文件或目录。",
        output_contract: "返回结果时带上清晰的文件路径引用或明确的文件系统事实。",
        failure_handling: "如果路径不存在或操作失败，明确指出失败的路径或动作。",
    },
    github_operator: {
        description: "使用 GitHub 相关工具检查仓库、Pull Request、提交和 Issue。",
        when_to_use: "适用于 GitHub 仓库、PR、Issue 和提交记录相关的工作流。",
        when_not_to_use: "不适用于通用网页浏览、本地工作区操作或无关研究任务。",
        instructions: "优先使用 GitHub 原生工具，保持结果聚焦在仓库上下文，并简洁总结当前状态。",
        output_contract: "返回相关 GitHub 对象及其当前状态，聚焦用户真正需要的信息。",
        failure_handling: "如果 GitHub 无法访问，明确说明是认证、连接还是资源不存在的问题。",
    },
};

const LEGACY_ALL_TOOLS_PROXY_ZH = {
    description: "兜底兼容 skill，暴露当前已注册的全部 MCP 工具。",
    when_to_use: "只有在没有更具体的 skill 能匹配当前任务时，才使用这个兜底 skill。",
    when_not_to_use: "如果专用 skill 或单服务 legacy skill 已经能更准确地完成任务，就不要使用它。",
    instructions: "保持聚焦，只调用完成当前任务所需的最少工具。",
    output_contract: "直接返回最终答案或结果，不添加无关流程。",
    failure_handling: "如果现有工具仍无法完成任务，明确说明缺少哪些能力或信息。",
};

const LEGACY_SECTION_TEMPLATE_ZH = {
    when_to_use: (serverLabel) => `当任务明显对应到 ${serverLabel}，且没有更强的专用 skill 更适合时，使用这个兼容代理。`,
    when_not_to_use: () => "如果已经有更专门、非 legacy 的 skill 可以处理同类任务，就不要使用这个兼容代理。",
    instructions: (serverLabel) => `以兼容代理的方式工作，只调用 ${serverLabel} 暴露的工具，并让结果严格对齐用户请求。`,
    output_contract: () => "直接返回用户需要的实际结果，不添加无关流程。",
    failure_handling: (serverLabel) => `如果底层 ${serverLabel} 无法完成请求，直接说明阻塞原因。`,
};

function getSkillId(skill) {
    return skill?.manifest?.id || "";
}

function getSkillEnglishDescription(skill) {
    return skill?.manifest?.description || "-";
}

function getSkillEnglishSection(skill, sectionKey) {
    return skill?.sections?.[sectionKey] || "-";
}

function getLegacyServerName(skill) {
    const path = skill?.path || "";
    if (path.startsWith("legacy://")) {
        return path.slice("legacy://".length);
    }

    const id = getSkillId(skill);
    const match = id.match(/^legacy_(.+)_proxy$/);
    return match ? match[1] : "";
}

function getLegacyServiceLabel(skill) {
    const serverName = getLegacyServerName(skill);
    if (!serverName) {
        return "该 MCP 服务";
    }
    return `${serverName} MCP 服务`;
}

function getLocalizedSkillRecord(skill) {
    return SKILL_DESCRIPTION_ZH[getSkillId(skill)] || null;
}

function getLegacyAllToolsDescription(sectionKey) {
    return LEGACY_ALL_TOOLS_PROXY_ZH[sectionKey] || "";
}

function getLegacySkillDescriptionZh(skill) {
    const skillId = getSkillId(skill);
    if (skillId === "legacy_all_tools_proxy") {
        return LEGACY_ALL_TOOLS_PROXY_ZH.description;
    }

    const serverName = getLegacyServerName(skill);
    if (!serverName) {
        return "";
    }

    return getMcpDescription(serverName, getSkillEnglishDescription(skill), "zh");
}

function getLegacySkillSectionZh(skill, sectionKey) {
    const skillId = getSkillId(skill);
    if (skillId === "legacy_all_tools_proxy") {
        return getLegacyAllToolsDescription(sectionKey);
    }

    const sectionTemplate = LEGACY_SECTION_TEMPLATE_ZH[sectionKey];
    if (!sectionTemplate) {
        return "";
    }

    return sectionTemplate(getLegacyServiceLabel(skill));
}

export function getSkillDescription(skill, language = "zh") {
    const englishDescription = getSkillEnglishDescription(skill);
    if (language === "en") {
        return englishDescription;
    }

    const localizedRecord = getLocalizedSkillRecord(skill);
    if (localizedRecord?.description) {
        return localizedRecord.description;
    }

    if (skill?.source === "legacy") {
        return getLegacySkillDescriptionZh(skill) || englishDescription;
    }

    return englishDescription;
}

export function getSkillSection(skill, sectionKey, language = "zh") {
    const englishSection = getSkillEnglishSection(skill, sectionKey);
    if (language === "en") {
        return englishSection;
    }

    const localizedRecord = getLocalizedSkillRecord(skill);
    if (localizedRecord?.[sectionKey]) {
        return localizedRecord[sectionKey];
    }

    if (skill?.source === "legacy") {
        return getLegacySkillSectionZh(skill, sectionKey) || englishSection;
    }

    return englishSection;
}
