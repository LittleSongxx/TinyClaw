import React, { useEffect, useState } from "react";
import BotSelector from "../components/BotSelector";
import Modal from "../components/Modal";
import Pagination from "../components/Pagination";
import Toast from "../components/Toast.jsx";
import { useTranslation } from "react-i18next";

function formatTime(ts) {
    if (!ts) {
        return "-";
    }
    return new Date(ts * 1000).toLocaleString();
}

function trimText(text, limit = 80) {
    if (!text) {
        return "-";
    }
    if (text.length <= limit) {
        return text;
    }
    return `${text.slice(0, limit)}...`;
}

function JsonBlock({ title, value }) {
    return (
        <div className="rounded-lg border border-gray-200 bg-white p-4">
            <div className="mb-2 text-sm font-semibold text-gray-700">{title}</div>
            <pre className="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded bg-slate-900 p-3 text-xs text-slate-100">
                {value || "-"}
            </pre>
        </div>
    );
}

export default function Runs() {
    const { t } = useTranslation();
    const [botId, setBotId] = useState(null);
    const [runs, setRuns] = useState([]);
    const [page, setPage] = useState(1);
    const [pageSize] = useState(10);
    const [total, setTotal] = useState(0);
    const [mode, setMode] = useState("");
    const [status, setStatus] = useState("");
    const [userId, setUserId] = useState("");
    const [selectedRun, setSelectedRun] = useState(null);
    const [detailVisible, setDetailVisible] = useState(false);
    const [toast, setToast] = useState({ show: false, message: "", type: "error" });
    const [loading, setLoading] = useState(false);
    const [replayLoading, setReplayLoading] = useState(false);

    const showToast = (message, type = "error") => {
        setToast({ show: true, message, type });
    };

    useEffect(() => {
        if (botId !== null) {
            fetchRuns();
        }
    }, [botId, page, mode, status, userId]);

    const fetchRuns = async () => {
        try {
            setLoading(true);
            const params = new URLSearchParams({
                id: botId,
                page,
                pageSize,
                mode,
                status,
                userId,
            });
            const res = await fetch(`/bot/run/list?${params.toString()}`);
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to fetch runs");
                return;
            }
            setRuns(data.data.list || []);
            setTotal(data.data.total || 0);
        } catch (err) {
            showToast(`Failed to fetch runs: ${err.message}`);
        } finally {
            setLoading(false);
        }
    };

    const openRunDetail = async (runId) => {
        try {
            const params = new URLSearchParams({
                id: botId,
                run_id: runId,
            });
            const res = await fetch(`/bot/run/get?${params.toString()}`);
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to fetch run detail");
                return;
            }
            setSelectedRun(data.data);
            setDetailVisible(true);
        } catch (err) {
            showToast(`Failed to fetch run detail: ${err.message}`);
        }
    };

    const replayRun = async (runId) => {
        try {
            setReplayLoading(true);
            const params = new URLSearchParams({
                id: botId,
                run_id: runId,
            });
            const res = await fetch(`/bot/run/replay?${params.toString()}`, {
                method: "POST",
            });
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Replay failed");
                return;
            }
            setSelectedRun(data.data);
            setDetailVisible(true);
            showToast(t("replay_success"), "success");
            await fetchRuns();
        } catch (err) {
            showToast(`Replay failed: ${err.message}`);
        } finally {
            setReplayLoading(false);
        }
    };

    const plannerOutputs = (selectedRun?.steps || []).filter((step) => step.kind === "planner" || step.kind === "judge");
    const toolSteps = (selectedRun?.steps || []).filter((step) => (step.observations || []).length > 0);

    return (
        <div className="min-h-screen bg-gray-100 p-6">
            {toast.show && (
                <Toast
                    message={toast.message}
                    type={toast.type}
                    onClose={() => setToast({ ...toast, show: false })}
                />
            )}

            <div className="mb-6 flex items-center justify-between">
                <div>
                    <h2 className="text-2xl font-bold text-gray-800">{t("run_manage")}</h2>
                    <p className="mt-1 text-sm text-gray-500">{t("run_manage_desc")}</p>
                </div>
            </div>

            <div className="mb-6 grid gap-4 rounded-lg bg-white p-4 shadow md:grid-cols-4">
                <BotSelector
                    value={botId}
                    onChange={(bot) => {
                        setBotId(bot.id);
                        setPage(1);
                    }}
                />

                <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">{t("mode")}</label>
                    <select
                        value={mode}
                        onChange={(event) => {
                            setMode(event.target.value);
                            setPage(1);
                        }}
                        className="w-full rounded border border-gray-300 px-3 py-2"
                    >
                        <option value="">{t("all_modes")}</option>
                        <option value="task">task</option>
                        <option value="mcp">mcp</option>
                    </select>
                </div>

                <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">{t("status")}</label>
                    <select
                        value={status}
                        onChange={(event) => {
                            setStatus(event.target.value);
                            setPage(1);
                        }}
                        className="w-full rounded border border-gray-300 px-3 py-2"
                    >
                        <option value="">{t("all_status")}</option>
                        <option value="running">running</option>
                        <option value="succeeded">succeeded</option>
                        <option value="failed">failed</option>
                    </select>
                </div>

                <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">{t("user_id")}</label>
                    <input
                        type="text"
                        value={userId}
                        onChange={(event) => {
                            setUserId(event.target.value);
                            setPage(1);
                        }}
                        placeholder={t("user_id_placeholder")}
                        className="w-full rounded border border-gray-300 px-3 py-2"
                    />
                </div>
            </div>

            <div className="overflow-x-auto rounded-lg bg-white shadow">
                <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                    <tr>
                        {[
                            "ID",
                            t("mode"),
                            t("status"),
                            t("user_id"),
                            t("token"),
                            t("total_steps"),
                            t("replay_of"),
                            t("update_time"),
                            t("final_output"),
                            t("action"),
                        ].map((title) => (
                            <th
                                key={title}
                                className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
                            >
                                {title}
                            </th>
                        ))}
                    </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                    {runs.length > 0 ? (
                        runs.map((run) => (
                            <tr key={run.id} className="hover:bg-gray-50">
                                <td className="px-4 py-4 text-sm text-gray-800">{run.id}</td>
                                <td className="px-4 py-4 text-sm text-gray-800">{run.mode}</td>
                                <td className="px-4 py-4 text-sm text-gray-800">{run.status}</td>
                                <td className="px-4 py-4 text-sm text-gray-800">{run.user_id}</td>
                                <td className="px-4 py-4 text-sm text-gray-800">{run.token_total || 0}</td>
                                <td className="px-4 py-4 text-sm text-gray-800">{run.step_count || 0}</td>
                                <td className="px-4 py-4 text-sm text-gray-800">{run.replay_of || "-"}</td>
                                <td className="px-4 py-4 text-sm text-gray-800">{formatTime(run.update_time)}</td>
                                <td className="px-4 py-4 text-sm text-gray-800">{trimText(run.final_output)}</td>
                                <td className="px-4 py-4 text-sm text-gray-800 space-x-3">
                                    <button
                                        onClick={() => openRunDetail(run.id)}
                                        className="text-blue-600 hover:underline"
                                    >
                                        {t("view")}
                                    </button>
                                    <button
                                        onClick={() => replayRun(run.id)}
                                        className="text-emerald-600 hover:underline"
                                    >
                                        {replayLoading ? t("replaying") : t("replay")}
                                    </button>
                                </td>
                            </tr>
                        ))
                    ) : (
                        <tr>
                            <td colSpan={10} className="py-8 text-center text-gray-500">
                                {loading ? t("loading") : t("no_data")}
                            </td>
                        </tr>
                    )}
                    </tbody>
                </table>
            </div>

            <Pagination page={page} pageSize={pageSize} total={total} onPageChange={setPage} />

            <Modal
                visible={detailVisible}
                title={t("run_detail")}
                onClose={() => setDetailVisible(false)}
            >
                <div className="max-h-[80vh] space-y-4 overflow-y-auto pr-2">
                    <div className="rounded-lg border border-gray-200 bg-slate-50 p-4">
                        <div className="mb-3 flex items-center justify-between">
                            <h3 className="text-lg font-semibold text-gray-800">{t("run_summary")}</h3>
                            {selectedRun?.run?.id && (
                                <button
                                    onClick={() => replayRun(selectedRun.run.id)}
                                    className="rounded bg-emerald-600 px-3 py-1 text-sm text-white hover:bg-emerald-700"
                                >
                                    {replayLoading ? t("replaying") : t("replay")}
                                </button>
                            )}
                        </div>
                        <div className="grid gap-3 text-sm text-gray-700 md:grid-cols-3">
                            <div><span className="font-semibold">ID:</span> {selectedRun?.run?.id || "-"}</div>
                            <div><span className="font-semibold">{t("mode")}:</span> {selectedRun?.run?.mode || "-"}</div>
                            <div><span className="font-semibold">{t("status")}:</span> {selectedRun?.run?.status || "-"}</div>
                            <div><span className="font-semibold">{t("user_id")}:</span> {selectedRun?.run?.user_id || "-"}</div>
                            <div><span className="font-semibold">{t("token")}:</span> {selectedRun?.run?.token_total || 0}</div>
                            <div><span className="font-semibold">{t("total_steps")}:</span> {selectedRun?.run?.step_count || 0}</div>
                            <div className="md:col-span-3"><span className="font-semibold">{t("input")}:</span> {selectedRun?.run?.input || "-"}</div>
                        </div>
                    </div>

                    <div className="grid gap-4 md:grid-cols-2">
                        <JsonBlock
                            title={t("planner_judge_output")}
                            value={plannerOutputs.map((step) => `${step.kind}#${step.step_index} ${step.name}\n${step.raw_output || "-"}`).join("\n\n")}
                        />
                        <JsonBlock
                            title={t("final_output")}
                            value={selectedRun?.run?.final_output || selectedRun?.run?.error || "-"}
                        />
                    </div>

                    <div className="rounded-lg border border-gray-200 bg-white p-4">
                        <h3 className="mb-3 text-lg font-semibold text-gray-800">{t("step_trace")}</h3>
                        <div className="space-y-3">
                            {(selectedRun?.steps || []).map((step) => (
                                <div key={step.id} className="rounded border border-gray-200 bg-slate-50 p-3">
                                    <div className="mb-2 flex flex-wrap items-center gap-3 text-sm text-gray-700">
                                        <span className="rounded bg-slate-900 px-2 py-1 text-xs text-white">
                                            #{step.step_index} {step.kind}
                                        </span>
                                        <span className="font-semibold">{step.name}</span>
                                        <span>{step.status}</span>
                                        <span>{step.provider || "-"}/{step.model || "-"}</span>
                                        <span>{t("token")}: {step.token || 0}</span>
                                        {step.tool_name && <span>{t("tool_name")}: {step.tool_name}</span>}
                                    </div>
                                    <div className="grid gap-3 md:grid-cols-2">
                                        <JsonBlock title={t("input")} value={step.input} />
                                        <JsonBlock title={t("raw_output")} value={step.raw_output} />
                                    </div>
                                    {step.error && (
                                        <div className="mt-3 rounded bg-red-50 p-3 text-sm text-red-700">
                                            {step.error}
                                        </div>
                                    )}
                                </div>
                            ))}
                        </div>
                    </div>

                    <div className="rounded-lg border border-gray-200 bg-white p-4">
                        <h3 className="mb-3 text-lg font-semibold text-gray-800">{t("tool_calls")}</h3>
                        <div className="space-y-3">
                            {toolSteps.length > 0 ? toolSteps.map((step) => (
                                <div key={`obs-${step.id}`} className="rounded border border-gray-200 bg-slate-50 p-3">
                                    <div className="mb-2 text-sm font-semibold text-gray-700">
                                        #{step.step_index} {step.tool_name || step.name}
                                    </div>
                                    <pre className="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded bg-slate-900 p-3 text-xs text-slate-100">
                                        {JSON.stringify(step.observations || [], null, 2)}
                                    </pre>
                                </div>
                            )) : (
                                <div className="rounded bg-slate-50 p-4 text-sm text-gray-500">{t("no_data")}</div>
                            )}
                        </div>
                    </div>

                    <div className="rounded-lg border border-gray-200 bg-white p-4">
                        <h3 className="mb-3 text-lg font-semibold text-gray-800">{t("final_output_error")}</h3>
                        <JsonBlock title={t("final_output")} value={selectedRun?.run?.final_output} />
                        {selectedRun?.run?.error && (
                            <div className="mt-3 rounded bg-red-50 p-3 text-sm text-red-700">
                                {selectedRun.run.error}
                            </div>
                        )}
                    </div>
                </div>
            </Modal>
        </div>
    );
}
