import React, { useEffect, useRef, useState } from "react";
import BotSelector from "../components/BotSelector";
import ConfirmModal from "../components/ConfirmModal.jsx";
import Modal from "../components/Modal";
import Pagination from "../components/Pagination";
import Toast from "../components/Toast.jsx";
import Editor from "@monaco-editor/react";
import { useTranslation } from "react-i18next";

function formatTime(ts) {
    if (!ts) {
        return "-";
    }
    return new Date(ts * 1000).toLocaleString();
}

function isEditableDocument(document) {
    if (!document) {
        return false;
    }
    const contentType = document.content_type || "";
    if (contentType.startsWith("text/")) {
        return true;
    }
    return /\.(txt|md|markdown|csv|html|htm|json)$/i.test(document.name || "");
}

async function fileToBase64(file) {
    const buffer = await file.arrayBuffer();
    let binary = "";
    const bytes = new Uint8Array(buffer);
    for (let i = 0; i < bytes.byteLength; i += 1) {
        binary += String.fromCharCode(bytes[i]);
    }
    return btoa(binary);
}

export default function Knowledge() {
    const { t } = useTranslation();
    const fileInputRef = useRef(null);
    const [botId, setBotId] = useState(null);
    const [activeTab, setActiveTab] = useState("documents");
    const [toast, setToast] = useState({ show: false, message: "", type: "error" });

    const [documents, setDocuments] = useState([]);
    const [docPage, setDocPage] = useState(1);
    const [docTotal, setDocTotal] = useState(0);
    const [docPageSize] = useState(10);
    const [docSearch, setDocSearch] = useState("");
    const [docModalVisible, setDocModalVisible] = useState(false);
    const [docFileName, setDocFileName] = useState("");
    const [docContent, setDocContent] = useState("");
    const [docEditing, setDocEditing] = useState(false);
    const [docDeleteVisible, setDocDeleteVisible] = useState(false);
    const [docToDelete, setDocToDelete] = useState(null);

    const [jobs, setJobs] = useState([]);
    const [jobPage, setJobPage] = useState(1);
    const [jobTotal, setJobTotal] = useState(0);
    const [jobPageSize] = useState(10);
    const [jobStatus, setJobStatus] = useState("");

    const [query, setQuery] = useState("");
    const [debugResult, setDebugResult] = useState(null);
    const [retrievalRuns, setRetrievalRuns] = useState([]);
    const [retrievalPage, setRetrievalPage] = useState(1);
    const [retrievalTotal, setRetrievalTotal] = useState(0);
    const [retrievalPageSize] = useState(5);

    const showToast = (message, type = "error") => {
        setToast({ show: true, message, type });
    };

    useEffect(() => {
        if (botId === null) {
            return;
        }
        fetchDocuments();
    }, [botId, docPage, docSearch]);

    useEffect(() => {
        if (botId === null || activeTab !== "jobs") {
            return;
        }
        fetchJobs();
    }, [botId, activeTab, jobPage, jobStatus]);

    useEffect(() => {
        if (botId === null || activeTab !== "retrieval") {
            return;
        }
        fetchRetrievalRuns();
    }, [botId, activeTab, retrievalPage]);

    const fetchDocuments = async () => {
        try {
            const params = new URLSearchParams({
                id: botId,
                page: docPage,
                pageSize: docPageSize,
                name: docSearch,
            });
            const res = await fetch(`/bot/knowledge/documents/list?${params.toString()}`);
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to fetch documents");
                return;
            }
            setDocuments(data.data.list || []);
            setDocTotal(data.data.total || 0);
        } catch (err) {
            showToast(`Failed to fetch documents: ${err.message}`);
        }
    };

    const fetchJobs = async () => {
        try {
            const params = new URLSearchParams({
                id: botId,
                page: jobPage,
                pageSize: jobPageSize,
                status: jobStatus,
            });
            const res = await fetch(`/bot/knowledge/jobs/list?${params.toString()}`);
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to fetch jobs");
                return;
            }
            setJobs(data.data.list || []);
            setJobTotal(data.data.total || 0);
        } catch (err) {
            showToast(`Failed to fetch jobs: ${err.message}`);
        }
    };

    const fetchRetrievalRuns = async () => {
        try {
            const params = new URLSearchParams({
                id: botId,
                page: retrievalPage,
                pageSize: retrievalPageSize,
            });
            const res = await fetch(`/bot/knowledge/retrieval/runs/list?${params.toString()}`);
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to fetch retrieval runs");
                return;
            }
            setRetrievalRuns(data.data.list || []);
            setRetrievalTotal(data.data.total || 0);
        } catch (err) {
            showToast(`Failed to fetch retrieval runs: ${err.message}`);
        }
    };

    const openCreateTextModal = () => {
        setDocEditing(false);
        setDocFileName("");
        setDocContent("");
        setDocModalVisible(true);
    };

    const openEditTextModal = async (name) => {
        try {
            const params = new URLSearchParams({
                id: botId,
                file_name: name,
            });
            const res = await fetch(`/bot/knowledge/documents/get?${params.toString()}`);
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to fetch document content");
                return;
            }
            setDocEditing(true);
            setDocFileName(name);
            setDocContent(data.data.content || "");
            setDocModalVisible(true);
        } catch (err) {
            showToast(`Failed to fetch document content: ${err.message}`);
        }
    };

    const saveTextDocument = async () => {
        if (!docFileName.trim()) {
            showToast(t("file_name_required"));
            return;
        }
        try {
            const res = await fetch(`/bot/knowledge/documents/create?id=${botId}`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    file_name: docFileName,
                    content: docContent,
                    source_type: "text",
                    content_type: "text/plain; charset=utf-8",
                }),
            });
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to save document");
                return;
            }
            showToast(docEditing ? t("document_updated") : t("document_created"), "success");
            setDocModalVisible(false);
            await fetchDocuments();
            if (activeTab === "jobs") {
                await fetchJobs();
            }
        } catch (err) {
            showToast(`Failed to save document: ${err.message}`);
        }
    };

    const uploadDocument = async (event) => {
        const file = event.target.files?.[0];
        if (!file) {
            return;
        }
        try {
            const dataBase64 = await fileToBase64(file);
            const res = await fetch(`/bot/knowledge/documents/create?id=${botId}`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    file_name: file.name,
                    source_type: "upload",
                    content_type: file.type,
                    data_base64: dataBase64,
                }),
            });
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to upload document");
                return;
            }
            showToast(t("document_uploaded"), "success");
            await fetchDocuments();
            if (activeTab === "jobs") {
                await fetchJobs();
            }
        } catch (err) {
            showToast(`Failed to upload document: ${err.message}`);
        } finally {
            event.target.value = "";
        }
    };

    const deleteDocument = async () => {
        if (!docToDelete) {
            return;
        }
        try {
            const params = new URLSearchParams({
                id: botId,
                file_name: docToDelete,
            });
            const res = await fetch(`/bot/knowledge/documents/delete?${params.toString()}`, {
                method: "POST",
            });
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to delete document");
                return;
            }
            showToast(t("document_deleted"), "success");
            setDocDeleteVisible(false);
            setDocToDelete(null);
            await fetchDocuments();
        } catch (err) {
            showToast(`Failed to delete document: ${err.message}`);
        }
    };

    const debugRetrieve = async () => {
        if (!query.trim()) {
            showToast(t("query_required"));
            return;
        }
        try {
            const res = await fetch(`/bot/knowledge/retrieval/debug?id=${botId}`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ query }),
            });
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to debug retrieval");
                return;
            }
            setDebugResult(data.data);
            await fetchRetrievalRuns();
        } catch (err) {
            showToast(`Failed to debug retrieval: ${err.message}`);
        }
    };

    const loadRetrievalRun = async (runId) => {
        try {
            const params = new URLSearchParams({
                id: botId,
                run_id: runId,
            });
            const res = await fetch(`/bot/knowledge/retrieval/runs/get?${params.toString()}`);
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to fetch retrieval run");
                return;
            }
            setDebugResult(data.data);
            setActiveTab("retrieval");
        } catch (err) {
            showToast(`Failed to fetch retrieval run: ${err.message}`);
        }
    };

    const tabs = [
        { key: "documents", label: t("documents") },
        { key: "jobs", label: t("ingestion_jobs") },
        { key: "retrieval", label: t("retrieval_debug") },
    ];

    return (
        <div className="h-full overflow-auto bg-gray-100 p-6">
            {toast.show && (
                <Toast
                    message={toast.message}
                    type={toast.type}
                    onClose={() => setToast({ ...toast, show: false })}
                />
            )}

            <div className="mb-6 flex items-center justify-between">
                <div>
                    <h2 className="text-2xl font-bold text-gray-800">{t("knowledge_manage")}</h2>
                    <p className="mt-1 text-sm text-gray-500">{t("knowledge_manage_desc")}</p>
                </div>
                <div className="flex gap-2">
                    <input
                        ref={fileInputRef}
                        type="file"
                        className="hidden"
                        onChange={uploadDocument}
                    />
                    <button
                        onClick={() => fileInputRef.current?.click()}
                        className="rounded bg-slate-700 px-4 py-2 text-white hover:bg-slate-800"
                    >
                        {t("upload_file")}
                    </button>
                    <button
                        onClick={openCreateTextModal}
                        className="rounded bg-claw-600 px-4 py-2 text-white hover:bg-claw-700"
                    >
                        {t("add_text_document")}
                    </button>
                </div>
            </div>

            <div className="mb-4 max-w-md">
                <BotSelector
                    value={botId}
                    onChange={(bot) => {
                        setBotId(bot.id);
                        setDocPage(1);
                        setJobPage(1);
                        setRetrievalPage(1);
                    }}
                />
            </div>

            <div className="mb-6 flex gap-2">
                {tabs.map((tab) => (
                    <button
                        key={tab.key}
                        onClick={() => setActiveTab(tab.key)}
                        className={`rounded-full px-4 py-2 text-sm font-semibold ${
                            activeTab === tab.key
                                ? "bg-claw-600 text-white"
                                : "bg-white text-gray-700 shadow"
                        }`}
                    >
                        {tab.label}
                    </button>
                ))}
            </div>

            {activeTab === "documents" && (
                <>
                    <div className="mb-4 grid gap-4 rounded-lg bg-white p-4 shadow md:grid-cols-2">
                        <div>
                            <label className="mb-1 block text-sm font-medium text-gray-700">{t("file_name")}</label>
                            <input
                                type="text"
                                value={docSearch}
                                onChange={(event) => {
                                    setDocSearch(event.target.value);
                                    setDocPage(1);
                                }}
                                className="w-full rounded border border-gray-300 px-3 py-2"
                                placeholder={t("search")}
                            />
                        </div>
                    </div>

                    <div className="overflow-x-auto rounded-lg bg-white shadow">
                        <table className="min-w-full divide-y divide-gray-200">
                            <thead className="bg-gray-50">
                            <tr>
                                {[t("file_name"), t("source_type"), t("status"), t("version"), t("chunk_count"), t("update_time"), t("action")].map((title) => (
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
                            {documents.length > 0 ? documents.map((document) => (
                                <tr key={document.id} className="hover:bg-gray-50">
                                    <td className="px-4 py-4 text-sm text-gray-800">{document.name}</td>
                                    <td className="px-4 py-4 text-sm text-gray-800">{document.source_type}</td>
                                    <td className="px-4 py-4 text-sm text-gray-800">{document.current_status || document.status}</td>
                                    <td className="px-4 py-4 text-sm text-gray-800">{document.latest_version || 0}</td>
                                    <td className="px-4 py-4 text-sm text-gray-800">{document.chunk_count || 0}</td>
                                    <td className="px-4 py-4 text-sm text-gray-800">{formatTime(document.update_time)}</td>
                                    <td className="px-4 py-4 text-sm text-gray-800 space-x-3">
                                        {isEditableDocument(document) && (
                                            <button
                                                onClick={() => openEditTextModal(document.name)}
                                                className="text-claw-600 hover:underline"
                                            >
                                                {t("edit")}
                                            </button>
                                        )}
                                        <button
                                            onClick={() => {
                                                setDocToDelete(document.name);
                                                setDocDeleteVisible(true);
                                            }}
                                            className="text-red-600 hover:underline"
                                        >
                                            {t("delete")}
                                        </button>
                                    </td>
                                </tr>
                            )) : (
                                <tr>
                                    <td colSpan={7} className="py-8 text-center text-gray-500">
                                        {t("no_data")}
                                    </td>
                                </tr>
                            )}
                            </tbody>
                        </table>
                    </div>

                    <Pagination page={docPage} pageSize={docPageSize} total={docTotal} onPageChange={setDocPage} />
                </>
            )}

            {activeTab === "jobs" && (
                <>
                    <div className="mb-4 max-w-sm rounded-lg bg-white p-4 shadow">
                        <label className="mb-1 block text-sm font-medium text-gray-700">{t("status")}</label>
                        <select
                            value={jobStatus}
                            onChange={(event) => {
                                setJobStatus(event.target.value);
                                setJobPage(1);
                            }}
                            className="w-full rounded border border-gray-300 px-3 py-2"
                        >
                            <option value="">{t("all_status")}</option>
                            <option value="queued">queued</option>
                            <option value="running">running</option>
                            <option value="succeeded">succeeded</option>
                            <option value="failed">failed</option>
                        </select>
                    </div>

                    <div className="overflow-x-auto rounded-lg bg-white shadow">
                        <table className="min-w-full divide-y divide-gray-200">
                            <thead className="bg-gray-50">
                            <tr>
                                {[t("run_id"), t("file_name"), t("version"), t("stage"), t("status"), t("update_time"), t("error")].map((title) => (
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
                            {jobs.length > 0 ? jobs.map((job) => (
                                <tr key={job.id} className="hover:bg-gray-50">
                                    <td className="px-4 py-4 text-sm text-gray-800">{job.id}</td>
                                    <td className="px-4 py-4 text-sm text-gray-800">{job.document_name || "-"}</td>
                                    <td className="px-4 py-4 text-sm text-gray-800">{job.version || "-"}</td>
                                    <td className="px-4 py-4 text-sm text-gray-800">{job.stage}</td>
                                    <td className="px-4 py-4 text-sm text-gray-800">{job.status}</td>
                                    <td className="px-4 py-4 text-sm text-gray-800">{formatTime(job.update_time)}</td>
                                    <td className="px-4 py-4 text-sm text-red-600">{job.error || "-"}</td>
                                </tr>
                            )) : (
                                <tr>
                                    <td colSpan={7} className="py-8 text-center text-gray-500">{t("no_data")}</td>
                                </tr>
                            )}
                            </tbody>
                        </table>
                    </div>

                    <Pagination page={jobPage} pageSize={jobPageSize} total={jobTotal} onPageChange={setJobPage} />
                </>
            )}

            {activeTab === "retrieval" && (
                <div className="space-y-4">
                    <div className="rounded-lg bg-white p-4 shadow">
                        <div className="grid gap-3 md:grid-cols-[1fr_auto]">
                            <textarea
                                value={query}
                                onChange={(event) => setQuery(event.target.value)}
                                rows={3}
                                placeholder={t("query_placeholder")}
                                className="w-full rounded border border-gray-300 px-3 py-2"
                            />
                            <button
                                onClick={debugRetrieve}
                                className="self-end rounded bg-claw-600 px-4 py-2 text-white hover:bg-claw-700"
                            >
                                {t("run_debug")}
                            </button>
                        </div>
                    </div>

                    <div className="grid gap-4 lg:grid-cols-[1.5fr_1fr]">
                        <div className="space-y-4">
                            <div className="rounded-lg bg-white p-4 shadow">
                                <h3 className="mb-3 text-lg font-semibold text-gray-800">{t("retrieval_debug")}</h3>
                                <div className="mb-3 text-sm text-gray-600">
                                    <span className="font-semibold">{t("query")}:</span> {debugResult?.query || "-"}
                                </div>
                                <div className="mb-3 rounded bg-slate-900 p-3 text-sm text-slate-100 whitespace-pre-wrap">
                                    {debugResult?.run?.answer || "-"}
                                </div>
                                <div className="text-sm text-gray-700">
                                    <span className="font-semibold">{t("citations")}:</span>{" "}
                                    {(debugResult?.run?.citations || []).join(", ") || "-"}
                                </div>
                            </div>

                            <div className="rounded-lg bg-white p-4 shadow">
                                <h3 className="mb-3 text-lg font-semibold text-gray-800">{t("retrieval_hits")}</h3>
                                <div className="space-y-3">
                                    {(debugResult?.hits || []).length > 0 ? debugResult.hits.map((hit) => (
                                        <div key={`${hit.chunk_id}-${hit.rank_position}`} className="rounded border border-gray-200 bg-slate-50 p-3">
                                            <div className="mb-2 flex flex-wrap gap-3 text-xs text-gray-600">
                                                <span className="rounded bg-slate-900 px-2 py-1 text-white">#{hit.rank_position}</span>
                                                <span>{hit.document_name}</span>
                                                <span>{hit.citation_label}</span>
                                                <span>dense {Number(hit.dense_score || 0).toFixed(4)}</span>
                                                <span>lexical {Number(hit.lexical_score || 0).toFixed(4)}</span>
                                                <span>rrf {Number(hit.rrf_score || 0).toFixed(4)}</span>
                                            </div>
                                            <pre className="whitespace-pre-wrap break-words text-sm text-gray-800">{hit.content}</pre>
                                        </div>
                                    )) : (
                                        <div className="rounded bg-slate-50 p-4 text-sm text-gray-500">{t("no_data")}</div>
                                    )}
                                </div>
                            </div>
                        </div>

                        <div className="rounded-lg bg-white p-4 shadow">
                            <h3 className="mb-3 text-lg font-semibold text-gray-800">{t("recent_retrieval_runs")}</h3>
                            <div className="space-y-2">
                                {retrievalRuns.length > 0 ? retrievalRuns.map((run) => (
                                    <button
                                        key={run.id}
                                        onClick={() => loadRetrievalRun(run.id)}
                                        className="w-full rounded border border-gray-200 p-3 text-left hover:border-claw-400 hover:bg-claw-50"
                                    >
                                        <div className="mb-1 text-sm font-semibold text-gray-800">#{run.id} {run.status}</div>
                                        <div className="text-sm text-gray-600">{run.query_text}</div>
                                        <div className="mt-2 text-xs text-gray-500">{formatTime(run.update_time)}</div>
                                    </button>
                                )) : (
                                    <div className="rounded bg-slate-50 p-4 text-sm text-gray-500">{t("no_data")}</div>
                                )}
                            </div>

                            <Pagination
                                page={retrievalPage}
                                pageSize={retrievalPageSize}
                                total={retrievalTotal}
                                onPageChange={setRetrievalPage}
                            />
                        </div>
                    </div>
                </div>
            )}

            <Modal
                visible={docModalVisible}
                title={docEditing ? t("edit_document") : t("add_text_document")}
                onClose={() => setDocModalVisible(false)}
            >
                <div className="space-y-4">
                    <input
                        type="text"
                        value={docFileName}
                        onChange={(event) => setDocFileName(event.target.value)}
                        placeholder={t("file_name")}
                        disabled={docEditing}
                        className="w-full rounded border border-gray-300 px-3 py-2"
                    />
                    <Editor
                        height="420px"
                        defaultLanguage="markdown"
                        value={docContent}
                        onChange={(value) => setDocContent(value ?? "")}
                        options={{
                            minimap: { enabled: false },
                            fontSize: 14,
                            automaticLayout: true,
                        }}
                    />
                    <div className="flex justify-end gap-2">
                        <button
                            onClick={() => setDocModalVisible(false)}
                            className="rounded bg-gray-200 px-4 py-2 text-gray-800 hover:bg-gray-300"
                        >
                            {t("cancel")}
                        </button>
                        <button
                            onClick={saveTextDocument}
                            className="rounded bg-claw-600 px-4 py-2 text-white hover:bg-claw-700"
                        >
                            {t("save")}
                        </button>
                    </div>
                </div>
            </Modal>

            <ConfirmModal
                visible={docDeleteVisible}
                message={t("delete_document_confirm")}
                onCancel={() => {
                    setDocDeleteVisible(false);
                    setDocToDelete(null);
                }}
                onConfirm={deleteDocument}
            />
        </div>
    );
}
