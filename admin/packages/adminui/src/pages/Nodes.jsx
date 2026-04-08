import React, { useEffect, useMemo, useState } from "react";
import BotSelector from "../components/BotSelector";
import Toast from "../components/Toast";
import { useTranslation } from "react-i18next";

function Nodes() {
    const { t } = useTranslation();
    const [bot, setBot] = useState(null);
    const [nodes, setNodes] = useState([]);
    const [sessions, setSessions] = useState([]);
    const [approvals, setApprovals] = useState([]);
    const [loading, setLoading] = useState(false);
    const [executing, setExecuting] = useState(false);
    const [selectedNodeId, setSelectedNodeId] = useState("");
    const [capability, setCapability] = useState("screen.snapshot");
    const [argumentsText, setArgumentsText] = useState("{\n  \"scope\": \"virtual_desktop\"\n}");
    const [requireApproval, setRequireApproval] = useState(false);
    const [result, setResult] = useState(null);
    const [toast, setToast] = useState(null);
    const [windowInfo, setWindowInfo] = useState(null);
    const [treeDepth, setTreeDepth] = useState(4);
    const [uiaTree, setUiaTree] = useState([]);
    const [foundElements, setFoundElements] = useState([]);
    const [selectedElement, setSelectedElement] = useState(null);
    const [elementText, setElementText] = useState("Hello from TinyClaw");
    const [findForm, setFindForm] = useState({
        name: "",
        automation_id: "",
        role: "",
        class_name: "",
        exact: false,
        max_results: 8,
    });

    useEffect(() => {
        if (!bot?.id) {
            setNodes([]);
            setSessions([]);
            setApprovals([]);
            setSelectedNodeId("");
            setResult(null);
            setWindowInfo(null);
            setUiaTree([]);
            setFoundElements([]);
            setSelectedElement(null);
            return;
        }
        fetchGatewayState(bot.id);
    }, [bot?.id]);

    const selectedNode = useMemo(
        () => nodes.find((item) => item.id === selectedNodeId) || null,
        [nodes, selectedNodeId]
    );

    const screenshotPreview = useMemo(() => {
        const mimeType = result?.data?.mime_type;
        const base64 = result?.data?.base64;
        if (!mimeType || !base64) {
            return "";
        }
        return `data:${mimeType};base64,${base64}`;
    }, [result]);

    const templates = useMemo(() => {
        const baseTemplates = [
            {
                label: "截取全桌面",
                capability: "screen.snapshot",
                requireApproval: false,
                arguments: { scope: "virtual_desktop" },
            },
            {
                label: "截取当前窗口",
                capability: "screen.snapshot",
                requireApproval: false,
                arguments: { scope: "active_window" },
            },
            {
                label: "列桌面窗口",
                capability: "window.list",
                requireApproval: false,
                arguments: {},
            },
            {
                label: "加载活动窗口 UIA 树",
                capability: "ui.inspect",
                requireApproval: false,
                arguments: { mode: "window_tree", depth: treeDepth },
            },
        ];

        if (selectedNode?.platform === "windows") {
            return [
                ...baseTemplates,
                {
                    label: "打开记事本",
                    capability: "app.launch",
                    requireApproval: false,
                    arguments: { command: "notepad.exe" },
                },
                {
                    label: "聚焦记事本",
                    capability: "window.focus",
                    requireApproval: false,
                    arguments: { title: "记事本" },
                },
                {
                    label: "输入测试文字",
                    capability: "input.keyboard.type",
                    requireApproval: true,
                    arguments: { text: "Hello from TinyClaw" },
                },
                {
                    label: "发送 Ctrl+S",
                    capability: "input.keyboard.hotkey",
                    requireApproval: true,
                    arguments: { keys: ["CTRL", "S"] },
                },
                {
                    label: "单击屏幕中心",
                    capability: "input.mouse.click",
                    requireApproval: true,
                    arguments: { x: 960, y: 540, button: "left", clicks: 1 },
                },
                {
                    label: "查找保存按钮",
                    capability: "ui.find",
                    requireApproval: false,
                    arguments: { name: "保存", role: "button", max_results: 5 },
                },
            ];
        }

        return [
            ...baseTemplates,
            {
                label: "列当前目录",
                capability: "fs.list",
                requireApproval: false,
                arguments: { path: "." },
            },
        ];
    }, [selectedNode?.platform, treeDepth]);

    const fetchGatewayState = async (botId) => {
        setLoading(true);
        try {
            const [nodesRes, sessionsRes, approvalsRes] = await Promise.all([
                fetch(`/bot/nodes/list?id=${botId}`),
                fetch(`/bot/sessions/list?id=${botId}`),
                fetch(`/bot/approvals/list?id=${botId}`),
            ]);

            const [nodesData, sessionsData, approvalsData] = await Promise.all([
                nodesRes.json(),
                sessionsRes.json(),
                approvalsRes.json(),
            ]);

            if (nodesData.code !== 0) {
                throw new Error(nodesData.message || "获取节点列表失败");
            }
            if (sessionsData.code !== 0) {
                throw new Error(sessionsData.message || "获取会话列表失败");
            }
            if (approvalsData.code !== 0) {
                throw new Error(approvalsData.message || "获取审批列表失败");
            }

            const nextNodes = nodesData.data || [];
            setNodes(nextNodes);
            setSessions(sessionsData.data || []);
            setApprovals(approvalsData.data || []);

            if (nextNodes.length === 0) {
                setSelectedNodeId("");
            } else if (!nextNodes.some((item) => item.id === selectedNodeId)) {
                setSelectedNodeId(nextNodes[0].id);
            }
        } catch (err) {
            setToast({ message: err.message || "加载节点信息失败", type: "error" });
        } finally {
            setLoading(false);
        }
    };

    const parseArguments = () => {
        const trimmed = argumentsText.trim();
        if (!trimmed) {
            return {};
        }
        return JSON.parse(trimmed);
    };

    const applyCommandState = (activeCapability, commandResult) => {
        if (!commandResult?.data) {
            return;
        }
        const payload = commandResult.data;
        if (payload.window) {
            setWindowInfo(payload.window);
        }
        if (activeCapability === "window.focus") {
            setWindowInfo(payload);
        }
        if (activeCapability === "ui.inspect") {
            setWindowInfo(payload.window || null);
            setUiaTree(payload.nodes || []);
            setFoundElements([]);
            if (payload.element) {
                setSelectedElement(withWindowLocator(payload.element, payload.window));
            } else if (payload.nodes?.length > 0) {
                setSelectedElement(withWindowLocator(payload.nodes[0], payload.window));
            }
        }
        if (activeCapability === "ui.find") {
            const matches = (payload.matches || []).map((item) => withWindowLocator(item, payload.window));
            setFoundElements(matches);
            if (matches.length > 0) {
                setSelectedElement(matches[0]);
            }
            if (payload.window) {
                setWindowInfo(payload.window);
            }
        }
        if ((activeCapability.startsWith("input.") || activeCapability === "ui.focus") && payload.element) {
            setSelectedElement(withWindowLocator(payload.element, windowInfo || payload.window || null));
        }
    };

    const executeCommand = async (override = null) => {
        if (!bot?.id) {
            setToast({ message: "请先选择机器人", type: "error" });
            return;
        }
        const activeNodeId = override?.nodeId || selectedNodeId;
        const activeCapability = override?.capability || capability;
        const activeRequireApproval = override?.requireApproval ?? requireApproval;

        if (!activeNodeId) {
            setToast({ message: "请先选择一个在线节点", type: "error" });
            return;
        }
        if (!activeCapability) {
            setToast({ message: "请选择 capability", type: "error" });
            return;
        }

        setExecuting(true);
        try {
            const activeArguments = override?.arguments ?? parseArguments();
            const response = await fetch(`/bot/node/command?id=${bot.id}`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({
                    node_id: activeNodeId,
                    capability: activeCapability,
                    arguments: activeArguments,
                    require_approval: activeRequireApproval,
                }),
            });
            const data = await response.json();
            if (data.code !== 0) {
                throw new Error(data.message || "执行节点命令失败");
            }
            setResult(data.data);
            setCapability(activeCapability);
            setArgumentsText(JSON.stringify(activeArguments, null, 2));
            setRequireApproval(Boolean(activeRequireApproval));
            applyCommandState(activeCapability, data.data);
            setToast({ message: t("command_executed"), type: "success" });
            await fetchGatewayState(bot.id);
        } catch (err) {
            setToast({ message: err.message || "执行节点命令失败", type: "error" });
        } finally {
            setExecuting(false);
        }
    };

    const decideApproval = async (commandId, approved) => {
        if (!bot?.id) {
            return;
        }
        try {
            const response = await fetch(`/bot/approvals/decide?id=${bot.id}`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    command_id: commandId,
                    approved: approved,
                }),
            });
            const data = await response.json();
            if (data.code !== 0) {
                throw new Error(data.message || "审批失败");
            }
            setToast({ message: t("approval_decided"), type: "success" });
            if (data.data?.result) {
                setResult(data.data.result);
                applyCommandState(data.data.result.capability || "", data.data.result);
            }
            await fetchGatewayState(bot.id);
        } catch (err) {
            setToast({ message: err.message || "审批失败", type: "error" });
        }
    };

    const fillTemplate = (template) => {
        setCapability(template.capability);
        setArgumentsText(JSON.stringify(template.arguments, null, 2));
        setRequireApproval(Boolean(template.requireApproval));
    };

    const copyResult = async () => {
        if (!result) {
            return;
        }
        try {
            await navigator.clipboard.writeText(JSON.stringify(result, null, 2));
            setToast({ message: t("result_copied"), type: "success" });
        } catch (err) {
            setToast({ message: "复制失败", type: "error" });
        }
    };

    const captureActiveWindow = () => executeCommand({
        capability: "screen.snapshot",
        requireApproval: false,
        arguments: { scope: "active_window" },
    });

    const loadWindowTree = () => executeCommand({
        capability: "ui.inspect",
        requireApproval: false,
        arguments: { mode: "window_tree", depth: treeDepth },
    });

    const runElementSearch = () => {
        const payload = {
            depth: treeDepth,
            max_results: Number(findForm.max_results) || 8,
            exact: findForm.exact,
        };
        if (findForm.name.trim()) payload.name = findForm.name.trim();
        if (findForm.automation_id.trim()) payload.automation_id = findForm.automation_id.trim();
        if (findForm.role.trim()) payload.role = findForm.role.trim();
        if (findForm.class_name.trim()) payload.class_name = findForm.class_name.trim();
        if (windowInfo?.handle) payload.window_handle = windowInfo.handle;
        executeCommand({
            capability: "ui.find",
            requireApproval: false,
            arguments: payload,
        });
    };

    const executeElementAction = (action, extra = {}, needsApproval = true) => {
        if (!selectedElement) {
            setToast({ message: "请先从 UIA 树或查找结果中选择一个元素", type: "error" });
            return;
        }
        executeCommand({
            capability: action,
            requireApproval: needsApproval,
            arguments: {
                ...extra,
                element: toElementLocator(selectedElement),
            },
        });
    };

    return (
        <div className="space-y-6">
            {toast && (
                <Toast
                    message={toast.message}
                    type={toast.type}
                    onClose={() => setToast(null)}
                />
            )}

            <section className="rounded-xl bg-white p-6 shadow space-y-4">
                <div className="flex flex-col gap-3 xl:flex-row xl:items-start xl:justify-between">
                    <div>
                        <h1 className="text-2xl font-semibold text-gray-900">{t("node_console")}</h1>
                        <p className="mt-1 text-sm text-gray-600">{t("node_console_desc")}</p>
                    </div>
                    <div className="w-full xl:w-96">
                        <BotSelector value={bot} onChange={setBot} />
                    </div>
                </div>

                <div className="grid gap-3 md:grid-cols-4">
                    <SummaryCard label={t("online_nodes")} value={nodes.length} />
                    <SummaryCard label={t("recent_sessions")} value={sessions.length} />
                    <SummaryCard label={t("pending_approvals")} value={approvals.length} />
                    <SummaryCard label={t("target_node")} value={selectedNode?.name || selectedNode?.id || "-"} compact />
                </div>
            </section>

            <section className="grid gap-6 xl:grid-cols-[1.1fr,0.9fr]">
                <div className="rounded-xl bg-white p-6 shadow space-y-4">
                    <div className="flex items-center justify-between">
                        <h2 className="text-lg font-semibold text-gray-900">{t("online_nodes")}</h2>
                        <button
                            type="button"
                            onClick={() => bot?.id && fetchGatewayState(bot.id)}
                            className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white hover:bg-slate-800 disabled:opacity-50"
                            disabled={!bot?.id || loading}
                        >
                            {loading ? t("loading") : t("refresh")}
                        </button>
                    </div>

                    <div className="space-y-3">
                        {nodes.length === 0 && (
                            <div className="rounded-lg border border-dashed border-slate-300 p-5 text-sm text-slate-500">
                                {t("no_online_nodes")}
                            </div>
                        )}

                        {nodes.map((item) => {
                            const isActive = item.id === selectedNodeId;
                            return (
                                <button
                                    type="button"
                                    key={item.id}
                                    onClick={() => setSelectedNodeId(item.id)}
                                    className={`w-full rounded-xl border p-4 text-left transition ${
                                        isActive
                                            ? "border-blue-500 bg-blue-50 shadow-sm"
                                            : "border-slate-200 hover:border-slate-300 hover:bg-slate-50"
                                    }`}
                                >
                                    <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
                                        <div>
                                            <div className="text-base font-semibold text-slate-900">{item.name || item.id}</div>
                                            <div className="mt-1 text-xs text-slate-500 break-all">node_id: {item.id}</div>
                                        </div>
                                        <div className="text-xs text-slate-600">
                                            <div>platform: {item.platform || "-"}</div>
                                            <div>hostname: {item.hostname || "-"}</div>
                                            <div>last_seen: {formatUnix(item.last_seen_at)}</div>
                                        </div>
                                    </div>

                                    <div className="mt-3 flex flex-wrap gap-2">
                                        {(item.capabilities || []).map((cap) => (
                                            <span key={`${item.id}-${cap.name}`} className="rounded-full bg-slate-800 px-2.5 py-1 text-xs font-medium text-white">
                                                {cap.name}
                                            </span>
                                        ))}
                                    </div>
                                </button>
                            );
                        })}
                    </div>
                </div>

                <div className="rounded-xl bg-white p-6 shadow space-y-4">
                    <h2 className="text-lg font-semibold text-gray-900">{t("pending_approvals")}</h2>
                    <div className="space-y-3 max-h-[34rem] overflow-auto">
                        {approvals.length === 0 && (
                            <div className="rounded-lg border border-dashed border-slate-300 p-4 text-sm text-slate-500">
                                {t("no_pending_approvals")}
                            </div>
                        )}

                        {approvals.map((item) => (
                            <div key={item.id} className="rounded-xl border border-amber-200 bg-amber-50 p-4">
                                <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
                                    <div>
                                        <div className="font-semibold text-slate-900">{item.capability}</div>
                                        <div className="mt-1 text-xs text-slate-600 break-all">{item.id}</div>
                                    </div>
                                    <div className="text-xs text-slate-600">{formatUnix(item.created_at)}</div>
                                </div>
                                <div className="mt-3 text-sm text-slate-700">
                                    <div className="font-medium">{t("approval_summary")}</div>
                                    <div className="mt-1">{item.summary || "-"}</div>
                                </div>
                                <pre className="mt-3 overflow-auto rounded-lg bg-white/80 p-3 text-xs text-slate-700">
                                    {JSON.stringify(item.arguments || {}, null, 2)}
                                </pre>
                                <div className="mt-3 flex gap-3">
                                    <button
                                        type="button"
                                        onClick={() => decideApproval(item.id, true)}
                                        className="rounded-lg bg-emerald-600 px-3 py-2 text-sm font-semibold text-white hover:bg-emerald-700"
                                    >
                                        {t("approve_action")}
                                    </button>
                                    <button
                                        type="button"
                                        onClick={() => decideApproval(item.id, false)}
                                        className="rounded-lg bg-red-600 px-3 py-2 text-sm font-semibold text-white hover:bg-red-700"
                                    >
                                        {t("reject_action")}
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            </section>

            <section className="grid gap-6 xl:grid-cols-[0.9fr,1.1fr]">
                <div className="rounded-xl bg-white p-6 shadow space-y-4">
                    <h2 className="text-lg font-semibold text-gray-900">{t("quick_actions")}</h2>
                    <p className="text-sm text-gray-600">这里保留了节点可视化入口，同时把活动窗口截图、UIA 树和元素调试也接进来了。</p>
                    <div className="grid gap-3 md:grid-cols-2">
                        {templates.map((template) => (
                            <div key={template.label} className="rounded-xl border border-slate-200 p-4">
                                <div className="font-semibold text-slate-900">{template.label}</div>
                                <div className="mt-1 text-xs text-slate-500">{template.capability}</div>
                                <div className="mt-3 flex gap-2">
                                    <button
                                        type="button"
                                        onClick={() => fillTemplate(template)}
                                        className="rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
                                    >
                                        填入
                                    </button>
                                    <button
                                        type="button"
                                        onClick={() => executeCommand(template)}
                                        className="rounded-lg bg-claw-600 px-3 py-2 text-sm font-semibold text-white hover:bg-claw-700"
                                    >
                                        立即发送
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>

                <div className="rounded-xl bg-white p-6 shadow space-y-4">
                    <h2 className="text-lg font-semibold text-gray-900">{t("manual_node_command")}</h2>
                    <div className="grid gap-4 md:grid-cols-2">
                        <div>
                            <label className="mb-1 block text-sm font-medium text-gray-700">Node</label>
                            <select
                                value={selectedNodeId}
                                onChange={(event) => setSelectedNodeId(event.target.value)}
                                className="w-full rounded-lg border border-gray-300 px-4 py-2 shadow-sm focus:border-claw-400 focus:outline-none focus:ring"
                            >
                                <option value="">请选择节点</option>
                                {nodes.map((item) => (
                                    <option key={item.id} value={item.id}>
                                        {item.name || item.id}
                                    </option>
                                ))}
                            </select>
                        </div>
                        <div>
                            <label className="mb-1 block text-sm font-medium text-gray-700">Capability</label>
                            <input
                                type="text"
                                value={capability}
                                onChange={(event) => setCapability(event.target.value)}
                                className="w-full rounded-lg border border-gray-300 px-4 py-2 shadow-sm focus:border-claw-400 focus:outline-none focus:ring"
                            />
                        </div>
                    </div>

                    <label className="flex items-center gap-2 text-sm text-slate-700">
                        <input
                            type="checkbox"
                            checked={requireApproval}
                            onChange={(event) => setRequireApproval(event.target.checked)}
                            className="h-4 w-4 rounded border-gray-300 text-claw-600 focus:ring-claw-500"
                        />
                        {t("require_approval")}
                    </label>

                    <div>
                        <label className="mb-1 block text-sm font-medium text-gray-700">Arguments JSON</label>
                        <textarea
                            value={argumentsText}
                            onChange={(event) => setArgumentsText(event.target.value)}
                            rows={12}
                            className="w-full rounded-lg border border-gray-300 px-4 py-3 font-mono text-sm shadow-sm focus:border-claw-400 focus:outline-none focus:ring"
                        />
                    </div>

                    <div className="flex gap-3">
                        <button
                            type="button"
                            onClick={() => executeCommand()}
                            disabled={executing}
                            className="rounded-lg bg-slate-900 px-4 py-2 font-semibold text-white hover:bg-slate-800 disabled:opacity-50"
                        >
                            {executing ? t("loading") : t("command")}
                        </button>
                        <button
                            type="button"
                            onClick={copyResult}
                            className="rounded-lg border border-slate-300 px-4 py-2 font-semibold text-slate-700 hover:bg-slate-50"
                        >
                            复制结果
                        </button>
                    </div>
                </div>
            </section>

            <section className="grid gap-6 xl:grid-cols-[1.1fr,0.9fr]">
                <div className="rounded-xl bg-white p-6 shadow space-y-4">
                    <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                        <div>
                            <h2 className="text-lg font-semibold text-gray-900">活动窗口调试</h2>
                            <p className="mt-1 text-sm text-slate-600">抓取当前前台窗口、查看窗口元数据，并加载该窗口的 UI Automation 控件树。</p>
                        </div>
                        <div className="flex flex-wrap gap-2">
                            <button
                                type="button"
                                onClick={captureActiveWindow}
                                className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-semibold text-white hover:bg-slate-800"
                            >
                                截取当前窗口
                            </button>
                            <button
                                type="button"
                                onClick={loadWindowTree}
                                className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50"
                            >
                                加载 UIA 树
                            </button>
                        </div>
                    </div>

                    <div className="flex items-center gap-3">
                        <label className="text-sm font-medium text-slate-700">树深度</label>
                        <input
                            type="number"
                            min="1"
                            max="8"
                            value={treeDepth}
                            onChange={(event) => setTreeDepth(Number(event.target.value) || 4)}
                            className="w-28 rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-claw-400 focus:outline-none focus:ring"
                        />
                    </div>

                    {windowInfo ? (
                        <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-700">
                            <div className="text-base font-semibold text-slate-900">{windowInfo.title || "当前窗口"}</div>
                            <div className="mt-2 grid gap-1 md:grid-cols-2">
                                <InfoLine label="handle" value={windowInfo.handle} />
                                <InfoLine label="process" value={windowInfo.process_name} />
                                <InfoLine label="foreground" value={String(Boolean(windowInfo.is_foreground))} />
                                <InfoLine label="bounds" value={formatBounds(windowInfo.bounds)} />
                            </div>
                        </div>
                    ) : (
                        <div className="rounded-lg border border-dashed border-slate-300 p-4 text-sm text-slate-500">
                            先执行“截取当前窗口”或“加载 UIA 树”，这里会显示当前活动窗口信息。
                        </div>
                    )}

                    <div className="grid gap-4 md:grid-cols-2">
                        <div>
                            <label className="mb-1 block text-sm font-medium text-slate-700">名称</label>
                            <input
                                value={findForm.name}
                                onChange={(event) => setFindForm((prev) => ({ ...prev, name: event.target.value }))}
                                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-claw-400 focus:outline-none focus:ring"
                                placeholder="例如：保存、确定、用户名"
                            />
                        </div>
                        <div>
                            <label className="mb-1 block text-sm font-medium text-slate-700">AutomationId</label>
                            <input
                                value={findForm.automation_id}
                                onChange={(event) => setFindForm((prev) => ({ ...prev, automation_id: event.target.value }))}
                                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-claw-400 focus:outline-none focus:ring"
                            />
                        </div>
                        <div>
                            <label className="mb-1 block text-sm font-medium text-slate-700">角色 / ControlType</label>
                            <input
                                value={findForm.role}
                                onChange={(event) => setFindForm((prev) => ({ ...prev, role: event.target.value }))}
                                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-claw-400 focus:outline-none focus:ring"
                                placeholder="例如：button、edit、checkbox"
                            />
                        </div>
                        <div>
                            <label className="mb-1 block text-sm font-medium text-slate-700">ClassName</label>
                            <input
                                value={findForm.class_name}
                                onChange={(event) => setFindForm((prev) => ({ ...prev, class_name: event.target.value }))}
                                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-claw-400 focus:outline-none focus:ring"
                            />
                        </div>
                    </div>

                    <div className="flex flex-wrap items-center gap-3">
                        <label className="flex items-center gap-2 text-sm text-slate-700">
                            <input
                                type="checkbox"
                                checked={findForm.exact}
                                onChange={(event) => setFindForm((prev) => ({ ...prev, exact: event.target.checked }))}
                                className="h-4 w-4 rounded border-gray-300 text-claw-600 focus:ring-claw-500"
                            />
                            精确匹配
                        </label>
                        <label className="text-sm text-slate-700">
                            最多返回
                            <input
                                type="number"
                                min="1"
                                max="20"
                                value={findForm.max_results}
                                onChange={(event) => setFindForm((prev) => ({ ...prev, max_results: Number(event.target.value) || 8 }))}
                                className="ml-2 w-20 rounded-lg border border-gray-300 px-2 py-1 text-sm shadow-sm focus:border-claw-400 focus:outline-none focus:ring"
                            />
                        </label>
                        <button
                            type="button"
                            onClick={runElementSearch}
                            className="rounded-lg bg-claw-600 px-3 py-2 text-sm font-semibold text-white hover:bg-claw-700"
                        >
                            查找元素
                        </button>
                    </div>
                </div>

                <div className="rounded-xl bg-white p-6 shadow space-y-4">
                    <h2 className="text-lg font-semibold text-gray-900">元素调试</h2>
                    <div className="max-h-52 overflow-auto rounded-xl border border-slate-200 p-3">
                        {foundElements.length === 0 ? (
                            <div className="text-sm text-slate-500">查找结果会显示在这里，点击任一元素即可进入下方操作卡片。</div>
                        ) : (
                            <div className="space-y-2">
                                {foundElements.map((item, index) => (
                                    <button
                                        type="button"
                                        key={`${item.path || item.automation_id || item.name}-${index}`}
                                        onClick={() => setSelectedElement(item)}
                                        className={`w-full rounded-lg border p-3 text-left ${
                                            item.path === selectedElement?.path
                                                ? "border-blue-500 bg-blue-50"
                                                : "border-slate-200 hover:bg-slate-50"
                                        }`}
                                    >
                                        <div className="font-medium text-slate-900">{item.name || item.automation_id || item.path || "未命名元素"}</div>
                                        <div className="mt-1 text-xs text-slate-600">
                                            {item.role || item.control_type || "-"} · path {item.path || "-"}
                                        </div>
                                    </button>
                                ))}
                            </div>
                        )}
                    </div>

                    <div className="rounded-xl border border-slate-200 bg-slate-50 p-4">
                        <div className="text-sm font-semibold text-slate-900">选中元素</div>
                        {selectedElement ? (
                            <div className="mt-3 space-y-2 text-sm text-slate-700">
                                <InfoLine label="name" value={selectedElement.name || "-"} />
                                <InfoLine label="automation_id" value={selectedElement.automation_id || "-"} />
                                <InfoLine label="role" value={selectedElement.role || selectedElement.control_type || "-"} />
                                <InfoLine label="class_name" value={selectedElement.class_name || "-"} />
                                <InfoLine label="path" value={selectedElement.path || "-"} />
                                <InfoLine label="bounds" value={formatBounds(selectedElement.bounds)} />
                                <InfoLine label="window" value={selectedElement.window_title || windowInfo?.title || "-"} />
                                <div>
                                    <label className="mb-1 block text-sm font-medium text-slate-700">输入文本</label>
                                    <textarea
                                        value={elementText}
                                        onChange={(event) => setElementText(event.target.value)}
                                        rows={3}
                                        className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-claw-400 focus:outline-none focus:ring"
                                    />
                                </div>
                                <div className="flex flex-wrap gap-2 pt-2">
                                    <button
                                        type="button"
                                        onClick={() => executeElementAction("ui.focus", {}, false)}
                                        className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50"
                                    >
                                        聚焦元素
                                    </button>
                                    <button
                                        type="button"
                                        onClick={() => executeElementAction("input.mouse.click")}
                                        className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-semibold text-white hover:bg-slate-800"
                                    >
                                        点击元素
                                    </button>
                                    <button
                                        type="button"
                                        onClick={() => executeElementAction("input.keyboard.type", { text: elementText })}
                                        className="rounded-lg bg-claw-600 px-3 py-2 text-sm font-semibold text-white hover:bg-claw-700"
                                    >
                                        向元素输入
                                    </button>
                                    <button
                                        type="button"
                                        onClick={() => executeElementAction("input.keyboard.key", { key: "ENTER" })}
                                        className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50"
                                    >
                                        向元素发送 Enter
                                    </button>
                                </div>
                            </div>
                        ) : (
                            <div className="mt-2 text-sm text-slate-500">从 UIA 树或查找结果中选择一个元素后，这里会显示定位信息和调试动作。</div>
                        )}
                    </div>
                </div>
            </section>

            <section className="grid gap-6 xl:grid-cols-[1fr,1fr]">
                <div className="rounded-xl bg-white p-6 shadow space-y-4">
                    <h2 className="text-lg font-semibold text-gray-900">{t("command_result")}</h2>
                    {result ? (
                        <pre className="max-h-[32rem] overflow-auto rounded-lg bg-slate-950 p-4 text-xs text-slate-100">
                            {JSON.stringify(result, null, 2)}
                        </pre>
                    ) : (
                        <div className="rounded-lg border border-dashed border-slate-300 p-5 text-sm text-slate-500">
                            {t("result_placeholder")}
                        </div>
                    )}
                </div>

                <div className="rounded-xl bg-white p-6 shadow space-y-4">
                    <h2 className="text-lg font-semibold text-gray-900">{t("screenshot_preview")}</h2>
                    {screenshotPreview ? (
                        <img src={screenshotPreview} alt="node screenshot" className="w-full rounded-lg border border-slate-200" />
                    ) : (
                        <div className="rounded-lg border border-dashed border-slate-300 p-5 text-sm text-slate-500">
                            暂无截图预览
                        </div>
                    )}
                </div>
            </section>

            <section className="rounded-xl bg-white p-6 shadow space-y-4">
                <div className="flex items-center justify-between">
                    <h2 className="text-lg font-semibold text-gray-900">活动窗口 UIA 树</h2>
                    <span className="text-xs text-slate-500">{uiaTree.length > 0 ? `${uiaTree.length} 个根节点` : "暂无树数据"}</span>
                </div>
                <div className="max-h-[28rem] overflow-auto rounded-xl border border-slate-200 p-3">
                    {uiaTree.length === 0 ? (
                        <div className="text-sm text-slate-500">先加载 UIA 树，这里会显示活动窗口的控件层级。</div>
                    ) : (
                        uiaTree.map((node) => (
                            <UIATreeNode
                                key={node.path || node.name}
                                node={node}
                                depth={0}
                                selectedPath={selectedElement?.path}
                                onSelect={(nodeItem) => setSelectedElement(withWindowLocator(nodeItem, windowInfo))}
                            />
                        ))
                    )}
                </div>
            </section>

            <section className="rounded-xl bg-white p-6 shadow space-y-4">
                <h2 className="text-lg font-semibold text-gray-900">{t("recent_sessions")}</h2>
                <div className="grid gap-4 xl:grid-cols-2">
                    {sessions.length === 0 && (
                        <div className="rounded-lg border border-dashed border-slate-300 p-4 text-sm text-slate-500">
                            {t("no_recent_sessions")}
                        </div>
                    )}
                    {sessions.slice(0, 12).map((item) => (
                        <div key={item.session_id} className="rounded-xl border border-slate-200 p-4 text-sm text-slate-700">
                            <div className="font-medium text-slate-900 break-all">{item.session_id}</div>
                            <div className="mt-2 grid gap-1">
                                <div>channel: {item.channel || "-"}</div>
                                <div>kind: {item.kind || "-"}</div>
                                <div>peer/group: {item.peer_id || item.group_id || "-"}</div>
                                <div>messages: {item.message_count ?? 0}</div>
                                <div>updated: {formatUnix(item.update_time)}</div>
                            </div>
                        </div>
                    ))}
                </div>
            </section>
        </div>
    );
}

function SummaryCard({ label, value, compact = false }) {
    return (
        <div className="rounded-xl border border-slate-200 bg-slate-50 p-4">
            <div className="text-sm text-slate-500">{label}</div>
            <div className={`mt-2 font-semibold text-slate-900 ${compact ? "text-lg break-all" : "text-3xl"}`}>
                {value}
            </div>
        </div>
    );
}

function formatUnix(value) {
    if (!value) {
        return "-";
    }
    return new Date(value * 1000).toLocaleString();
}

function InfoLine({ label, value }) {
    return (
        <div className="flex items-start gap-2">
            <span className="min-w-24 text-xs font-medium uppercase tracking-wide text-slate-500">{label}</span>
            <span className="break-all text-sm text-slate-800">{value || "-"}</span>
        </div>
    );
}

function UIATreeNode({ node, depth, selectedPath, onSelect }) {
    const isSelected = node?.path && node.path === selectedPath;
    return (
        <div>
            <button
                type="button"
                onClick={() => onSelect(node)}
                className={`flex w-full items-start rounded-lg border px-3 py-2 text-left ${
                    isSelected
                        ? "border-blue-500 bg-blue-50"
                        : "border-transparent hover:border-slate-200 hover:bg-slate-50"
                }`}
                style={{ marginLeft: `${depth * 16}px` }}
            >
                <div className="min-w-0">
                    <div className="truncate text-sm font-medium text-slate-900">
                        {node.name || node.automation_id || node.path || "未命名元素"}
                    </div>
                    <div className="mt-1 text-xs text-slate-500">
                        {node.role || node.control_type || "-"} · path {node.path || "-"}
                    </div>
                </div>
            </button>
            {Array.isArray(node.children) &&
                node.children.map((child) => (
                    <UIATreeNode
                        key={child.path || `${node.path}-child`}
                        node={child}
                        depth={depth + 1}
                        selectedPath={selectedPath}
                        onSelect={onSelect}
                    />
                ))}
        </div>
    );
}

function withWindowLocator(node, windowInfo) {
    if (!node) {
        return null;
    }
    const next = {
        ...node,
    };
    if (windowInfo?.handle && !next.window_handle) {
        next.window_handle = windowInfo.handle;
    }
    if (windowInfo?.title && !next.window_title) {
        next.window_title = windowInfo.title;
    }
    if (windowInfo?.process_name && !next.process_name) {
        next.process_name = windowInfo.process_name;
    }
    return next;
}

function formatBounds(bounds) {
    if (!bounds) {
        return "-";
    }
    const { x, y, width, height } = bounds;
    if ([x, y, width, height].every((value) => typeof value === "number")) {
        return `${x}, ${y}, ${width} x ${height}`;
    }
    return JSON.stringify(bounds);
}

function toElementLocator(node) {
    if (!node) {
        return null;
    }
    return {
        path: node.path,
        automation_id: node.automation_id,
        name: node.name,
        role: node.role,
        class_name: node.class_name,
        window_handle: node.window_handle,
        window_title: node.window_title,
        process_name: node.process_name,
        index: node.index,
    };
}

export default Nodes;
