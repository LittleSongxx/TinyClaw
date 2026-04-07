export const MCP_AVAILABILITY_ORDER = ["ready", "secret", "runtime", "setup"];

const FALLBACK_NOTES = {
    zh: {
        ready: "当前环境检查通过，可直接添加或启用这个 MCP 服务。",
        secret: "当前环境里仍缺少这个 MCP 所需的密钥或令牌。",
        runtime: "当前环境里缺少这个 MCP 所需的运行时或服务端点。",
        setup: "这个 MCP 还保留了模板配置，接入前需要按实际环境调整。",
        "secret+runtime": "当前同时缺少密钥和运行环境。",
        "secret+setup": "当前同时缺少密钥，且模板配置还没有改成真实值。",
        "runtime+setup": "当前同时缺少运行环境，且模板配置还没有改成真实值。",
        "secret+runtime+setup": "当前同时缺少密钥、运行环境，并且模板配置还没有改成真实值。",
    },
    en: {
        ready: "The current environment checks passed, so this MCP service can be added or enabled directly.",
        secret: "The current environment is still missing the required secret or token for this MCP service.",
        runtime: "The current environment is still missing the required runtime or endpoint for this MCP service.",
        setup: "This MCP service still contains template configuration that should be adjusted before use.",
        "secret+runtime": "The current environment is missing both the required secret and runtime support.",
        "secret+setup": "The current environment is missing the required secret, and the template configuration still needs adjustment.",
        "runtime+setup": "The current environment is missing the required runtime, and the template configuration still needs adjustment.",
        "secret+runtime+setup": "The current environment is missing the required secret, runtime support, and real configuration values.",
    },
};

function normalizeStatuses(statuses) {
    const uniqueStatuses = Array.isArray(statuses) ? [...new Set(statuses)] : [];
    const orderedStatuses = MCP_AVAILABILITY_ORDER.filter((status) => uniqueStatuses.includes(status));

    if (orderedStatuses.length > 0) {
        return orderedStatuses;
    }

    return ["setup"];
}

export function getMcpAvailability(service, language = "zh") {
    const statuses = normalizeStatuses(service?.availability?.statuses);
    const noteKey = statuses.join("+");
    const note =
        service?.availability?.notes?.[language] ||
        service?.availability?.notes?.zh ||
        service?.availability?.notes?.en ||
        FALLBACK_NOTES[language]?.[noteKey] ||
        FALLBACK_NOTES[language]?.setup;

    return {
        statuses,
        note,
        registered: Boolean(service?.availability?.registered),
    };
}

export function getMcpAvailabilityCounts(services) {
    const counts = {
        ready: 0,
        secret: 0,
        runtime: 0,
        setup: 0,
    };

    for (const service of services) {
        const { statuses } = getMcpAvailability(service);
        for (const status of statuses) {
            counts[status] += 1;
        }
    }

    return counts;
}

export function serviceMatchesAvailabilityFilter(service, activeStatus) {
    if (!activeStatus) {
        return true;
    }

    return getMcpAvailability(service).statuses.includes(activeStatus);
}
