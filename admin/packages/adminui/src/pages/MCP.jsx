import React, { useEffect, useState } from "react";
import Toast from "../components/Toast";
import Modal from "../components/Modal";
import BotSelector from "../components/BotSelector";
import ConfirmModal from "../components/ConfirmModal.jsx";
import BulkActionToolbar from "../components/BulkActionToolbar.jsx";
import Editor from "@monaco-editor/react";
import {useTranslation} from "react-i18next";
import {getMcpDescription} from "../utils/mcpDescriptions";
import {
    getMcpAvailability,
    getMcpAvailabilityCounts,
    MCP_AVAILABILITY_ORDER,
    serviceMatchesAvailabilityFilter,
} from "../utils/mcpAvailability";

function BotMcpListPage() {
    const [botId, setBotId] = useState(null);
    const [mcpServices, setMcpServices] = useState([]);
    const [showEditModal, setShowEditModal] = useState(false);
    const [showPrepareModal, setShowPrepareModal] = useState(false);
    const [prepareServices, setPrepareServices] = useState([]);
    const [prepareTab, setPrepareTab] = useState("list");
    const [selectedPreparedService, setSelectedPreparedService] = useState(null);
    const [prepareEditJson, setPrepareEditJson] = useState("");
    const [editingService, setEditingService] = useState(null);
    const [editJson, setEditJson] = useState("");
    const [prepareSearch, setPrepareSearch] = useState("");
    const [descriptionLanguage, setDescriptionLanguage] = useState("zh");
    const [prepareDescriptionLanguage, setPrepareDescriptionLanguage] = useState("zh");
    const [availabilityFilter, setAvailabilityFilter] = useState(null);
    const [prepareAvailabilityFilter, setPrepareAvailabilityFilter] = useState(null);
    const [MCPToDelete, setMCPToDelete] = useState(null);
    const [confirmVisible, setConfirmVisible] = useState(false);
    const [toast, setToast] = useState({ show: false, message: "", type: "error" });
    const [selectionMode, setSelectionMode] = useState(false);
    const [selectedServiceNames, setSelectedServiceNames] = useState([]);
    const [batchAction, setBatchAction] = useState(null);

    const [confirmSyncVisible, setConfirmSyncVisible] = useState(false);

    const { t, i18n } = useTranslation();
    const availabilityLanguage = i18n.resolvedLanguage?.startsWith("zh") ? "zh" : "en";

    const handleSyncClick = () => {
        setConfirmSyncVisible(true); // 显示弹窗
    };

    const cancelSync = () => {
        setConfirmSyncVisible(false); // 取消
    };

    const confirmSync = async () => {
        try {
            const res = await fetch(`/bot/mcp/sync?id=${botId}`, {
                method: "POST",
            });
            const data = await res.json();
            if (data.code !== 0) return showToast(data.message || "Failed to fetch services");
            showToast(t("service_synced"), "success");
            setConfirmSyncVisible(false);
            await fetchMcpServices();
        } catch (error) {
            showToast(`${t("request_error")}: ${error.message}`);
        }
    };

    const showToast = (message, type = "error") => {
        setToast({ show: true, message, type });
    };

    useEffect(() => {
        if (botId !== null) {
            fetchMcpServices();
        }
    }, [botId]);

    useEffect(() => {
        setSelectedServiceNames((prev) => prev.filter((name) => mcpServices.some((svc) => svc.name === name)));
    }, [mcpServices]);

    const normalizeMcpServices = (inspectData) => {
        const mcpObj = inspectData?.mcpServers || {};
        const availabilityObj = inspectData?.availability || {};

        return Object.entries(mcpObj)
            .map(([name, config]) => ({
                name,
                config,
                availability: availabilityObj[name] || null,
            }))
            .sort((a, b) => a.name.localeCompare(b.name));
    };

    const fetchMcpServices = async () => {
        try {
            const res = await fetch(`/bot/mcp/get?id=${botId}`);
            const data = await res.json();
            if (data.code !== 0) return showToast(data.message || "Failed to fetch services");
            setMcpServices(normalizeMcpServices(data.data));
        } catch (err) {
            showToast("Request error: " + err.message);
        }
    };

    const handlePrepareClick = async () => {
        try {
            const res = await fetch(`/bot/mcp/prepare?id=${botId}`);
            const data = await res.json();
            if (data.code !== 0) return showToast(data.message || "Failed to prepare");
            setPrepareServices(normalizeMcpServices(data.data));
            setPrepareTab("list");
            setShowPrepareModal(true);
        } catch (err) {
            showToast("Prepare failed: " + err.message);
        }
    };

    const handleAddPreparedService = (name, config) => {
        setSelectedPreparedService(name);
        setPrepareEditJson(JSON.stringify(config, null, 2));
        setPrepareTab("json");
    };

    const handleSubmitPreparedService = async () => {
        try {
            const config = JSON.parse(prepareEditJson);
            const params = new URLSearchParams({ id: botId, name: selectedPreparedService });

            const res = await fetch(`/bot/mcp/update?${params.toString()}`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(config),
            });

            const data = await res.json();
            if (data.code !== 0) return showToast(data.message || "Failed to add");
            showToast(t("service_added"), "success");
            setShowPrepareModal(false);
            await fetchMcpServices();
        } catch (err) {
            showToast("Invalid JSON or request error: " + err.message);
        }
    };

    const openEditModal = (svc) => {
        setEditingService(svc.name);
        setEditJson(JSON.stringify(svc.config, null, 2));
        setShowEditModal(true);
    };

    const handleUpdateService = async () => {
        try {
            const config = JSON.parse(editJson);
            const params = new URLSearchParams({ id: botId, name: editingService });

            const res = await fetch(`/bot/mcp/update?${params.toString()}`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(config),
            });

            const data = await res.json();
            if (data.code !== 0) return showToast(data.message || "Failed to update MCP");

            showToast(t("service_updated"), "success");
            setShowEditModal(false);
            await fetchMcpServices();
        } catch (err) {
            showToast("Invalid JSON or request error: " + err.message);
        }
    };

    const toggleDisableService = async (name, disable) => {
        try {
            const params = new URLSearchParams({ id: botId, name, disable: disable ? "1" : "0" });
            const res = await fetch(`/bot/mcp/disable?${params.toString()}`);
            const data = await res.json();
            if (data.code !== 0) return showToast(data.message || "Failed to toggle");
            showToast(disable ? "Disabled" : "Enabled", "success");
            await fetchMcpServices();
        } catch (err) {
            showToast("Toggle failed: " + err.message);
        }
    };

    const filterServicesByAvailability = (services, activeStatus) =>
        services.filter((svc) => serviceMatchesAvailabilityFilter(svc, activeStatus));

    const filteredMcpServices = filterServicesByAvailability(mcpServices, availabilityFilter);

    const searchedPrepareServices = prepareServices.filter(svc =>
        svc.name.toLowerCase().includes(prepareSearch.toLowerCase())
    );

    const filteredPrepareServices = filterServicesByAvailability(searchedPrepareServices, prepareAvailabilityFilter);

    const handleDeleteClick = (name) => {
        setMCPToDelete(name);
        setConfirmVisible(true);
    };

    const cancelDelete = () => {
        setMCPToDelete(null);
        setConfirmVisible(false);
    };

    // 确认删除
    const confirmDelete = async () => {
        if (!MCPToDelete) return;
        try {
            const res = await fetch(`/bot/mcp/delete?id=${botId}&name=${MCPToDelete}`, { method: "DELETE" });
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to delete bot");
                return;
            }
            showToast(t("service_deleted"), "success");
            setConfirmVisible(false);
            setMCPToDelete(null);
            await fetchMcpServices();
        } catch (error) {
            showToast("Request error: " + error.message);
        }
    };

    const renderServiceDescription = (svc, language) =>
        getMcpDescription(svc.name, svc.config.description, language);

    const getServiceAvailability = (svc) =>
        getMcpAvailability(svc, availabilityLanguage);

    const renderAvailabilityBadge = (status, showCount = null) => {
        const styleMap = {
            ready: "border-emerald-200 bg-emerald-50 text-emerald-700",
            secret: "border-amber-200 bg-amber-50 text-amber-700",
            runtime: "border-sky-200 bg-sky-50 text-sky-700",
            setup: "border-rose-200 bg-rose-50 text-rose-700",
        };

        return (
            <span
                key={`${status}-${showCount ?? "plain"}`}
                className={`inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold ${styleMap[status]}`}
            >
                {t(`mcp_status_${status}`)}
                {showCount !== null ? ` ${showCount}` : ""}
            </span>
        );
    };

    const renderAvailabilityFilterBadge = (status, count, activeStatus, onChange) => {
        const styleMap = {
            ready: {
                active: "border-emerald-500 bg-emerald-500 text-white shadow-sm",
                inactive: "border-emerald-200 bg-emerald-50 text-emerald-700 hover:border-emerald-300 hover:bg-emerald-100",
            },
            secret: {
                active: "border-amber-500 bg-amber-500 text-white shadow-sm",
                inactive: "border-amber-200 bg-amber-50 text-amber-700 hover:border-amber-300 hover:bg-amber-100",
            },
            runtime: {
                active: "border-sky-500 bg-sky-500 text-white shadow-sm",
                inactive: "border-sky-200 bg-sky-50 text-sky-700 hover:border-sky-300 hover:bg-sky-100",
            },
            setup: {
                active: "border-rose-500 bg-rose-500 text-white shadow-sm",
                inactive: "border-rose-200 bg-rose-50 text-rose-700 hover:border-rose-300 hover:bg-rose-100",
            },
        };

        const isActive = activeStatus === status;
        const isDisabled = count === 0 && !isActive;
        const variant = styleMap[status];

        return (
            <button
                key={`${status}-${count}`}
                type="button"
                disabled={isDisabled}
                onClick={() => onChange(isActive ? null : status)}
                className={`inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold transition ${
                    isActive ? variant.active : variant.inactive
                } ${isDisabled ? "cursor-not-allowed opacity-50" : "cursor-pointer"}`}
            >
                {t(`mcp_status_${status}`)} {count}
            </button>
        );
    };

    const renderAllFilterBadge = (count, activeStatus, onChange) => {
        const isActive = activeStatus === null;

        return (
            <button
                type="button"
                onClick={() => onChange(null)}
                className={`inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold transition ${
                    isActive
                        ? "border-claw-500 bg-claw-500 text-white shadow-sm"
                        : "border-claw-200 bg-white text-claw-700 hover:border-claw-300 hover:bg-claw-50"
                }`}
            >
                {t("all_status")} {count}
            </button>
        );
    };

    const renderAvailabilityInfo = (svc, language) => {
        const availability = getServiceAvailability(svc);

        return (
            <div className="space-y-2">
                <div className="flex flex-wrap gap-2">
                    {availability.statuses.map((status) => renderAvailabilityBadge(status))}
                </div>
                <div>{renderServiceDescription(svc, language)}</div>
                <div className="text-xs text-gray-500">{availability.note}</div>
            </div>
        );
    };

    const renderDescriptionLanguageToggle = (selectedLanguage, onChange) => (
        <div className="flex items-center gap-3">
            <span className="text-sm font-medium text-gray-600">{t("description_language")}</span>
            <div className="inline-flex rounded-full border border-claw-200 bg-white p-1 shadow-sm">
                <button
                    type="button"
                    onClick={() => onChange("zh")}
                    className={`rounded-full px-3 py-1 text-sm font-medium transition ${
                        selectedLanguage === "zh"
                            ? "bg-claw-600 text-white shadow-sm"
                            : "text-gray-600 hover:text-claw-700"
                    }`}
                >
                    {t("chinese")}
                </button>
                <button
                    type="button"
                    onClick={() => onChange("en")}
                    className={`rounded-full px-3 py-1 text-sm font-medium transition ${
                        selectedLanguage === "en"
                            ? "bg-claw-600 text-white shadow-sm"
                            : "text-gray-600 hover:text-claw-700"
                    }`}
                >
                    {t("english")}
                </button>
            </div>
        </div>
    );

    const renderAvailabilityLegend = (services, activeStatus, onChange) => {
        const counts = getMcpAvailabilityCounts(services);

        return (
            <div className="rounded-xl border border-claw-100 bg-claw-50/70 p-3">
                <div className="text-sm font-medium text-gray-700">{t("mcp_availability_hint")}</div>
                <div className="mt-2 flex flex-wrap gap-2">
                    {renderAllFilterBadge(services.length, activeStatus, onChange)}
                    {MCP_AVAILABILITY_ORDER.map((status) =>
                        renderAvailabilityFilterBadge(status, counts[status], activeStatus, onChange)
                    )}
                </div>
            </div>
        );
    };

    const toggleSelectionMode = () => {
        setSelectionMode((prev) => !prev);
        setSelectedServiceNames([]);
    };

    const toggleServiceSelection = (name) => {
        setSelectedServiceNames((prev) =>
            prev.includes(name) ? prev.filter((item) => item !== name) : [...prev, name]
        );
    };

    const visibleServiceNames = filteredMcpServices.map((svc) => svc.name);
    const allVisibleSelected =
        visibleServiceNames.length > 0 && visibleServiceNames.every((name) => selectedServiceNames.includes(name));

    const handleSelectAllVisible = () => {
        setSelectedServiceNames((prev) => {
            if (allVisibleSelected) {
                return prev.filter((name) => !visibleServiceNames.includes(name));
            }
            return Array.from(new Set([...prev, ...visibleServiceNames]));
        });
    };

    const clearSelection = () => {
        setSelectedServiceNames([]);
    };

    const openBatchAction = (action) => {
        if (selectedServiceNames.length === 0) {
            showToast(t("no_selection"));
            return;
        }
        setBatchAction(action);
    };

    const runBatchAction = async () => {
        let targets = [...selectedServiceNames];

        if (batchAction === "enable") {
            targets = mcpServices
                .filter((svc) => selectedServiceNames.includes(svc.name) && svc.config.disabled)
                .map((svc) => svc.name);
        } else if (batchAction === "disable") {
            targets = mcpServices
                .filter((svc) => selectedServiceNames.includes(svc.name) && !svc.config.disabled)
                .map((svc) => svc.name);
        }

        if (targets.length === 0) {
            showToast(t("no_matching_batch_targets"));
            setBatchAction(null);
            return;
        }

        let success = 0;
        let failed = 0;

        for (const name of targets) {
            try {
                let res;
                if (batchAction === "delete") {
                    res = await fetch(`/bot/mcp/delete?id=${botId}&name=${name}`, { method: "DELETE" });
                } else {
                    const disable = batchAction === "disable" ? "1" : "0";
                    res = await fetch(`/bot/mcp/disable?id=${botId}&name=${name}&disable=${disable}`);
                }

                const data = await res.json();
                if (data.code === 0) {
                    success += 1;
                } else {
                    failed += 1;
                }
            } catch (error) {
                failed += 1;
            }
        }

        setBatchAction(null);
        clearSelection();

        if (failed > 0) {
            showToast(t("batch_operation_partial_failed", { success, failed }));
        } else {
            showToast(t("batch_operation_completed", { count: success }), "success");
        }

        await fetchMcpServices();
    };

    const getBatchConfirmMessage = () => {
        if (batchAction === "enable") {
            return t("batch_enable_confirm");
        }
        if (batchAction === "disable") {
            return t("batch_disable_confirm");
        }
        return t("batch_delete_confirm");
    };

    return (
        <div className="flex h-full min-h-0 flex-col overflow-hidden bg-gray-100 p-6">
            {toast.show && (
                <Toast message={toast.message} type={toast.type} onClose={() => setToast({ ...toast, show: false })} />
            )}

            <div className="flex justify-between items-center mb-6">
                <h2 className="text-2xl font-bold text-gray-800">{t("mcp_manage")}</h2>
                <div className="flex items-center gap-3">
                    {renderDescriptionLanguageToggle(descriptionLanguage, setDescriptionLanguage)}
                    <button
                        onClick={handlePrepareClick}
                        className="bg-claw-600 text-white px-4 py-2 rounded hover:bg-claw-700"
                    >
                        + {t("add_mcp")}
                    </button>
                    <button
                        onClick={handleSyncClick}
                        className="bg-green-600 text-white px-4 py-2 rounded hover:bg-green-700"
                    >
                        {t("sync_mcp")}
                    </button>
                </div>
            </div>

            <div className="flex space-x-4 mb-6 max-w-4xl flex-wrap">
                <div className="flex-1 min-w-[200px]">
                    <BotSelector
                        value={botId}
                        onChange={(bot) => {
                            setBotId(bot.id);
                        }}
                    />
                </div>
            </div>

            <div className="mb-4">
                {renderAvailabilityLegend(mcpServices, availabilityFilter, setAvailabilityFilter)}
            </div>

            <div className="mb-4">
                <BulkActionToolbar
                    selectionMode={selectionMode}
                    onToggleMode={toggleSelectionMode}
                    selectedCount={selectedServiceNames.length}
                    onSelectAllVisible={handleSelectAllVisible}
                    onClearSelection={clearSelection}
                    actions={
                        <>
                            <button
                                type="button"
                                onClick={() => openBatchAction("enable")}
                                disabled={selectedServiceNames.length === 0}
                                className="rounded-lg bg-green-600 px-3 py-2 text-sm font-semibold text-white transition hover:bg-green-700 disabled:cursor-not-allowed disabled:opacity-50"
                            >
                                {t("batch_enable")}
                            </button>
                            <button
                                type="button"
                                onClick={() => openBatchAction("disable")}
                                disabled={selectedServiceNames.length === 0}
                                className="rounded-lg bg-yellow-600 px-3 py-2 text-sm font-semibold text-white transition hover:bg-yellow-700 disabled:cursor-not-allowed disabled:opacity-50"
                            >
                                {t("batch_disable")}
                            </button>
                            <button
                                type="button"
                                onClick={() => openBatchAction("delete")}
                                disabled={selectedServiceNames.length === 0}
                                className="rounded-lg bg-red-600 px-3 py-2 text-sm font-semibold text-white transition hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50"
                            >
                                {t("batch_delete")}
                            </button>
                        </>
                    }
                />
            </div>

            <div className="min-h-0 flex-1 overflow-hidden rounded-lg shadow">
                <div className="h-full overflow-auto">
                    <table className="min-w-full bg-white divide-y divide-gray-200">
                        <thead className="bg-gray-50">
                        <tr>
                            {selectionMode && (
                                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                                    <input
                                        type="checkbox"
                                        checked={allVisibleSelected}
                                        onChange={handleSelectAllVisible}
                                        className="h-4 w-4 rounded border-gray-300 text-claw-600 focus:ring-claw-500"
                                    />
                                </th>
                            )}
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t("name")}</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t("description")}</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t("status")}</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t("action")}</th>
                        </tr>
                        </thead>
                        <tbody className="divide-y divide-gray-100">
                        {filteredMcpServices.map((svc) => (
                            <tr key={svc.name} className="hover:bg-gray-50">
                                {selectionMode && (
                                    <td className="px-4 py-4 text-sm text-gray-800">
                                        <input
                                            type="checkbox"
                                            checked={selectedServiceNames.includes(svc.name)}
                                            onChange={() => toggleServiceSelection(svc.name)}
                                            className="h-4 w-4 rounded border-gray-300 text-claw-600 focus:ring-claw-500"
                                        />
                                    </td>
                                )}
                                <td className="px-6 py-4 text-sm text-gray-800">{svc.name}</td>
                                <td className="px-6 py-4 text-sm text-gray-800 whitespace-pre-line">{renderAvailabilityInfo(svc, descriptionLanguage)}</td>
                                <td className="px-6 py-4 text-sm text-gray-800">{svc.config.disabled ? t("disable") : t("enable")}</td>
                                <td className="px-6 py-4 text-sm space-x-3">
                                    <button onClick={() => openEditModal(svc)} className="text-claw-600 hover:underline">{t("edit")}</button>
                                    {svc.config.disabled ? (
                                        <button onClick={() => toggleDisableService(svc.name, false)} className="text-green-600 hover:underline">{t("enable")}</button>
                                    ) : (
                                        <button onClick={() => toggleDisableService(svc.name, true)} className="text-yellow-600 hover:underline">{t("disable")}</button>
                                    )}
                                    <button onClick={() => handleDeleteClick(svc.name)} className="text-red-600 hover:underline">{t("delete")}</button>
                                </td>
                            </tr>
                        ))}
                        </tbody>
                    </table>
                </div>
            </div>

            <Modal
                visible={showEditModal}
                title={t("edit_mcp_service")}
                onClose={() => setShowEditModal(false)}
            >
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            {t("service_name")}
                        </label>
                        <input
                            type="text"
                            value={editingService || ""}
                            readOnly
                            className="w-full px-3 py-2 border border-gray-300 rounded bg-gray-100 text-gray-700"
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            {t("config_json")}
                        </label>
                        <div className="border rounded">
                            <Editor
                                height="300px"
                                defaultLanguage="json"
                                value={editJson}
                                onChange={(value) => setEditJson(value ?? "")}
                                options={{
                                    minimap: { enabled: false },
                                    fontSize: 14,
                                    automaticLayout: true,
                                    formatOnPaste: true,
                                    formatOnType: true,
                                }}
                            />
                        </div>
                    </div>
                    <div className="text-right">
                        <button
                            onClick={handleUpdateService}
                            className="bg-claw-600 text-white px-4 py-2 rounded hover:bg-claw-700"
                        >
                            {t("update")}
                        </button>
                    </div>
                </div>
            </Modal>

            <Modal visible={showPrepareModal} title={t("prepared_mcp_services")} onClose={() => setShowPrepareModal(false)}>
                <div className="max-h-[80vh] overflow-y-auto">
                    <div className="mb-4 flex items-center justify-between gap-4 border-b">
                        <div className="flex space-x-4">
                            <button className={`pb-2 ${prepareTab === "list" ? "border-b-2 border-claw-500 font-semibold" : "text-gray-500"}`} onClick={() => setPrepareTab("list")}>{t("service_list")}</button>
                            <button className={`pb-2 ${prepareTab === "json" ? "border-b-2 border-claw-500 font-semibold" : "text-gray-500"}`} onClick={() => setPrepareTab("json")}>{t("json_edit")}</button>
                        </div>
                        {renderDescriptionLanguageToggle(prepareDescriptionLanguage, setPrepareDescriptionLanguage)}
                    </div>

                    {prepareTab === "list" && (
                        <>
                            <div className="mb-4">
                                {renderAvailabilityLegend(searchedPrepareServices, prepareAvailabilityFilter, setPrepareAvailabilityFilter)}
                            </div>
                            <div className="mb-4">
                                <input
                                    type="text"
                                    placeholder={t("search_service_name")}
                                    value={prepareSearch}
                                    onChange={(e) => setPrepareSearch(e.target.value)}
                                    className="w-full px-4 py-2 border border-gray-300 rounded shadow-sm focus:outline-none focus:ring focus:border-claw-400"
                                />
                            </div>
                            <table className="min-w-full bg-white divide-y divide-gray-200">
                                <thead className="bg-gray-50">
                                <tr>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t("name")}</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t('description')}</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t('action')}</th>
                                </tr>
                                </thead>
                                <tbody className="divide-y divide-gray-100">
                                {filteredPrepareServices.map((svc) => (
                                    <tr key={svc.name} className="hover:bg-gray-50">
                                        <td className="px-6 py-4 text-sm text-gray-800">{svc.name}</td>
                                        <td className="px-6 py-4 text-sm text-gray-800 whitespace-pre-line">{renderAvailabilityInfo(svc, prepareDescriptionLanguage)}</td>
                                        <td className="px-6 py-4 text-sm">
                                            <button onClick={() => handleAddPreparedService(svc.name, svc.config)} className="bg-claw-600 hover:bg-claw-700 text-white px-3 py-1 rounded">{t("add")}</button>
                                        </td>
                                    </tr>
                                ))}
                                </tbody>
                            </table>
                        </>
                    )}

                    {prepareTab === "json" && (
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700">{t("service_name")}</label>
                                <input
                                    type="text"
                                    value={selectedPreparedService || ""}
                                    onChange={(e) => setSelectedPreparedService(e.target.value)}
                                    className="w-full px-3 py-2 border border-gray-300 rounded bg-white text-gray-700"
                                />
                            </div>

                            <div>
                                <label className="block text-sm font-medium text-gray-700">{t("config_json")}</label>
                                <div className="border rounded">
                                    <Editor
                                        height="300px"
                                        defaultLanguage="json"
                                        value={prepareEditJson}
                                        onChange={(value) => setPrepareEditJson(value ?? "")}
                                        options={{
                                            minimap: { enabled: false },
                                            fontSize: 14,
                                            automaticLayout: true,
                                            formatOnPaste: true,
                                            formatOnType: true,
                                        }}
                                    />
                                </div>
                            </div>

                            <div className="text-right">
                                <button
                                    onClick={handleSubmitPreparedService}
                                    className="bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded"
                                >
                                    {t("submit")}
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            </Modal>
            <ConfirmModal
                visible={confirmVisible}
                message={t("delete_mcp_confirm")}
                onConfirm={confirmDelete}
                onCancel={cancelDelete}
            />
            <ConfirmModal
                visible={batchAction !== null}
                title={batchAction ? t(`batch_${batchAction}`) : t("confirm")}
                message={getBatchConfirmMessage()}
                onConfirm={runBatchAction}
                onCancel={() => setBatchAction(null)}
            />
            <ConfirmModal
                visible={confirmSyncVisible}
                message={t("sync_mcp_confirm")}
                onConfirm={confirmSync}
                onCancel={cancelSync}
            />
        </div>
    );
}

export default BotMcpListPage;
