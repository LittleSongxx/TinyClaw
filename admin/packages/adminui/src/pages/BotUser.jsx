import React, { useEffect, useState } from "react";
import Pagination from "../components/Pagination";
import Modal from "../components/Modal";
import Toast from "../components/Toast";
import BotSelector from "../components/BotSelector";
import ConfirmModal from "../components/ConfirmModal";
import BulkActionToolbar from "../components/BulkActionToolbar.jsx";
import {useTranslation} from "react-i18next";

function BotUserListPage() {
    const [botId, setBotId] = useState(null);
    const [userIdSearch, setUserIdSearch] = useState("");
    const [users, setUsers] = useState([]);
    const [page, setPage] = useState(1);
    const [pageSize] = useState(10);
    const [total, setTotal] = useState(0);

    const [showModal, setShowModal] = useState(false);
    const [newUserId, setNewUserId] = useState("");
    const [newToken, setNewToken] = useState("");

    const [toast, setToast] = useState({ show: false, message: "", type: "error" });
    const [confirmVisible, setConfirmVisible] = useState(false);
    const [userToDelete, setUserToDelete] = useState(null);
    const [selectionMode, setSelectionMode] = useState(false);
    const [selectedUserIds, setSelectedUserIds] = useState([]);
    const [batchDeleteVisible, setBatchDeleteVisible] = useState(false);
    const showToast = (message, type = "error") => {
        setToast({ show: true, message, type });
    };

    const { t } = useTranslation();

    useEffect(() => {
        if (botId !== null) {
            fetchBotUsers();
        }
    }, [botId, page, userIdSearch]);

    useEffect(() => {
        setSelectedUserIds((prev) => prev.filter((id) => users.some((user) => user.user_id === id)));
    }, [users]);

    const fetchBotUsers = async () => {
        try {
            const params = new URLSearchParams({
                id: botId,
                page,
                pageSize,
            });
            if (userIdSearch.trim() !== "") {
                params.append("userId", userIdSearch.trim());
            }
            const res = await fetch(`/bot/user/list?${params.toString()}`);
            const data = await res.json();
            if (data.code !== 0) return showToast(data.message || "Failed to fetch users");
            setUsers(data.data.list || []);
            setTotal(data.data.total || 0);
        } catch (err) {
            showToast("Request error: " + err.message);
        }
    };

    const handleSubmitNewToken = async () => {
        try {
            const res = await fetch(`/bot/add/token?id=${botId}`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    botId,
                    user_id: newUserId,
                    token: Number(newToken),
                }),
            });

            const data = await res.json();
            if (data.code !== 0) return showToast(data.message || "Failed to submit token");

            setShowModal(false);
            setNewUserId("");
            setNewToken("");
            await fetchBotUsers();
            showToast("New token submitted", "success");
        } catch (err) {
            showToast("Submit new token failed: " + err.message);
        }
    };

    const handleUserIdSearchChange = (e) => {
        setUserIdSearch(e.target.value);
        setPage(1);
    };

    const handleDeleteClick = (userId) => {
        setUserToDelete(userId);
        setConfirmVisible(true);
    };

    const cancelDelete = () => {
        setUserToDelete(null);
        setConfirmVisible(false);
    };

    const confirmDelete = async () => {
        if (!userToDelete) return;

        try {
            const res = await fetch(`/bot/user/delete?id=${botId}&user_id=${encodeURIComponent(userToDelete)}`, {
                method: "DELETE",
            });
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || t("bot_user_delete_failed"));
                return;
            }

            showToast(t("bot_user_deleted"), "success");
            setConfirmVisible(false);
            setUserToDelete(null);
            await fetchBotUsers();
        } catch (err) {
            showToast(t("bot_user_delete_failed") + ": " + err.message);
        }
    };

    const toggleSelectionMode = () => {
        setSelectionMode((prev) => !prev);
        setSelectedUserIds([]);
    };

    const toggleUserSelection = (userId) => {
        setSelectedUserIds((prev) =>
            prev.includes(userId) ? prev.filter((id) => id !== userId) : [...prev, userId]
        );
    };

    const visibleUserIds = users.map((user) => user.user_id);
    const allVisibleSelected =
        visibleUserIds.length > 0 && visibleUserIds.every((id) => selectedUserIds.includes(id));

    const handleSelectAllVisible = () => {
        setSelectedUserIds((prev) => {
            if (allVisibleSelected) {
                return prev.filter((id) => !visibleUserIds.includes(id));
            }
            return Array.from(new Set([...prev, ...visibleUserIds]));
        });
    };

    const clearSelection = () => {
        setSelectedUserIds([]);
    };

    const openBatchDeleteConfirm = () => {
        if (selectedUserIds.length === 0) {
            showToast(t("no_selection"));
            return;
        }
        setBatchDeleteVisible(true);
    };

    const confirmBatchDelete = async () => {
        const ids = [...selectedUserIds];
        let success = 0;
        let failed = 0;

        for (const id of ids) {
            try {
                const res = await fetch(`/bot/user/delete?id=${botId}&user_id=${encodeURIComponent(id)}`, {
                    method: "DELETE",
                });
                const data = await res.json();
                if (data.code === 0) {
                    success += 1;
                } else {
                    failed += 1;
                }
            } catch (err) {
                failed += 1;
            }
        }

        setBatchDeleteVisible(false);
        clearSelection();

        if (failed > 0) {
            showToast(t("batch_operation_partial_failed", { success, failed }));
        } else {
            showToast(t("batch_operation_completed", { count: success }), "success");
        }

        await fetchBotUsers();
    };

    return (
        <div className="flex h-full min-h-0 flex-col overflow-hidden bg-gray-100 p-6">
            {toast.show && (
                <Toast
                    message={toast.message}
                    type={toast.type}
                    onClose={() => setToast({ ...toast, show: false })}
                />
            )}

            <div className="flex justify-between items-center mb-6">
                <h2 className="text-2xl font-bold text-gray-800">{t("bot_user_manage")}</h2>
                <button
                    onClick={() => setShowModal(true)}
                    className="bg-claw-600 text-white px-4 py-2 rounded hover:bg-claw-700"
                >
                    + {t("add_token")}
                </button>
            </div>

            <div className="flex space-x-4 mb-6 max-w-4xl flex-wrap">
                <div className="flex-1 min-w-[200px]">
                    <BotSelector
                        value={botId}
                        onChange={(bot) => {
                            setBotId(bot.id);
                            setUserIdSearch("");
                            setPage(1);
                        }}
                    />
                </div>

                <div className="flex-1 min-w-[200px]">
                    <label className="block font-medium text-gray-700 mb-1">{t("search_user_id")}:</label>
                    <input
                        type="text"
                        value={userIdSearch}
                        onChange={handleUserIdSearchChange}
                        placeholder={t("user_id_placeholder")}
                        className="w-full px-4 py-2 border border-gray-300 rounded shadow-sm focus:outline-none focus:ring focus:border-claw-400"
                    />
                </div>
            </div>

            <div className="mb-4">
                <BulkActionToolbar
                    selectionMode={selectionMode}
                    onToggleMode={toggleSelectionMode}
                    selectedCount={selectedUserIds.length}
                    onSelectAllVisible={handleSelectAllVisible}
                    onClearSelection={clearSelection}
                    actions={
                        <button
                            type="button"
                            onClick={openBatchDeleteConfirm}
                            disabled={selectedUserIds.length === 0}
                            className="rounded-lg bg-red-600 px-3 py-2 text-sm font-semibold text-white transition hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                            {t("batch_delete")}
                        </button>
                    }
                />
            </div>

            <div className="min-h-0 flex-1 overflow-hidden rounded-lg shadow">
                <div className="h-full overflow-auto">
                    <table className="min-w-full bg-white divide-y divide-gray-200">
                        <thead className="bg-gray-50">
                        <tr>
                            {selectionMode && (
                                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                    <input
                                        type="checkbox"
                                        checked={allVisibleSelected}
                                        onChange={handleSelectAllVisible}
                                        className="h-4 w-4 rounded border-gray-300 text-claw-600 focus:ring-claw-500"
                                    />
                                </th>
                            )}
                            {[t("id"), t("user_id"), t("mode"), t("token"), t("available_token"), t("create_time"), t("update_time"), t("action")].map((title) => (
                                <th
                                    key={title}
                                    className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                                >
                                    {title}
                                </th>
                            ))}
                        </tr>
                        </thead>
                        <tbody className="divide-y divide-gray-100">
                        {users.length > 0 ? (
                            users.map((user) => (
                                <tr key={user.id} className="hover:bg-gray-50">
                                    {selectionMode && (
                                        <td className="px-4 py-4 text-sm text-gray-800">
                                            <input
                                                type="checkbox"
                                                checked={selectedUserIds.includes(user.user_id)}
                                                onChange={() => toggleUserSelection(user.user_id)}
                                                className="h-4 w-4 rounded border-gray-300 text-claw-600 focus:ring-claw-500"
                                            />
                                        </td>
                                    )}
                                    <td className="px-6 py-4 text-sm text-gray-800">{user.id}</td>
                                    <td className="px-6 py-4 text-sm text-gray-800">{user.user_id}</td>
                                    <td className="px-6 py-4 text-sm text-gray-800">{user.llm_config}</td>
                                    <td className="px-6 py-4 text-sm text-gray-800">{user.token}</td>
                                    <td className="px-6 py-4 text-sm text-gray-800">{formatQuotaValue(user)}</td>
                                    <td className="px-6 py-4 text-sm text-gray-800">
                                        {new Date(user.create_time * 1000).toLocaleString()}
                                    </td>
                                    <td className="px-6 py-4 text-sm text-gray-800">
                                        {new Date(user.update_time * 1000).toLocaleString()}
                                    </td>
                                    <td className="px-6 py-4 text-sm text-gray-800">
                                        <button
                                            onClick={() => handleDeleteClick(user.user_id)}
                                            className="text-red-600 hover:underline"
                                        >
                                            {t("delete")}
                                        </button>
                                    </td>
                                </tr>
                            ))
                        ) : (
                            <tr>
                                <td colSpan={selectionMode ? 9 : 8} className="text-center py-6 text-gray-500">
                                    {t("no_users_found")}
                                </td>
                            </tr>
                        )}
                        </tbody>
                    </table>
                </div>
            </div>

            <div className="shrink-0">
                <Pagination page={page} pageSize={pageSize} total={total} onPageChange={setPage} />
            </div>

            <ConfirmModal
                visible={confirmVisible}
                title={t("delete")}
                message={t("delete_bot_user_confirm")}
                onConfirm={confirmDelete}
                onCancel={cancelDelete}
            />
            <ConfirmModal
                visible={batchDeleteVisible}
                title={t("batch_delete")}
                message={t("batch_delete_confirm")}
                onConfirm={confirmBatchDelete}
                onCancel={() => setBatchDeleteVisible(false)}
            />

            <Modal visible={showModal} title="Add New Token" onClose={() => setShowModal(false)}>
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-1">User ID:</label>
                        <input
                            type="text"
                            value={newUserId}
                            onChange={(e) => setNewUserId(e.target.value)}
                            className="w-full px-3 py-2 border border-gray-300 rounded"
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-1">Token:</label>
                        <input
                            type="text"
                            value={newToken}
                            onChange={(e) => setNewToken(e.target.value)}
                            className="w-full px-3 py-2 border border-gray-300 rounded"
                        />
                    </div>
                    <div className="text-right">
                        <button
                            onClick={handleSubmitNewToken}
                            className="bg-claw-600 hover:bg-claw-700 text-white px-4 py-2 rounded"
                        >
                            {t("submit")}
                        </button>
                    </div>
                </div>
            </Modal>
        </div>
    );
}

function formatQuotaValue(user) {
    if (user?.unlimited) {
        return "∞";
    }
    return user?.avail_token ?? "-";
}

export default BotUserListPage;
