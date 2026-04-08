import React, { useEffect, useState } from "react";
import Modal from "../components/Modal";
import Pagination from "../components/Pagination";
import Toast from "../components/Toast";
import ConfirmModal from "../components/ConfirmModal";
import BulkActionToolbar from "../components/BulkActionToolbar.jsx";
import {useTranslation} from "react-i18next";

function Users() {
    const [users, setUsers] = useState([]);
    const [search, setSearch] = useState("");
    const [modalVisible, setModalVisible] = useState(false);
    const [editingUser, setEditingUser] = useState(null);
    const [form, setForm] = useState({ id: 0, username: "", password: "" });

    const [page, setPage] = useState(1);
    const pageSize = 10;
    const [total, setTotal] = useState(0);

    const [toast, setToast] = useState({ show: false, message: "", type: "error" });

    const [confirmVisible, setConfirmVisible] = useState(false);
    const [userToDelete, setUserToDelete] = useState(null);
    const [selectionMode, setSelectionMode] = useState(false);
    const [selectedUserIds, setSelectedUserIds] = useState([]);
    const [batchDeleteVisible, setBatchDeleteVisible] = useState(false);

    const { t } = useTranslation();

    const showToast = (message, type = "error") => {
        setToast({ show: true, message, type });
    };

    useEffect(() => {
        fetchUsers();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [page]);

    useEffect(() => {
        setSelectedUserIds((prev) => prev.filter((id) => users.some((user) => user.id === id)));
    }, [users]);

    const fetchUsers = async () => {
        try {
            const res = await fetch(
                `/user/list?page=${page}&page_size=${pageSize}&username=${encodeURIComponent(search)}`
            );
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to fetch users");
                return;
            }
            setUsers(data.data.list);
            setTotal(data.data.total);
        } catch (error) {
            showToast("Request failed: " + error.message);
        }
    };

    const handleAddClick = () => {
        setForm({ id: 0, username: "", password: "" });
        setEditingUser(null);
        setModalVisible(true);
    };

    const handleEditClick = (user) => {
        setForm({ id: user.id, username: user.username, password: "" });
        setEditingUser(user);
        setModalVisible(true);
    };

    // 触发删除弹窗
    const handleDeleteClick = (userId) => {
        setUserToDelete(userId);
        setConfirmVisible(true);
    };

    // 取消删除弹窗
    const cancelDelete = () => {
        setUserToDelete(null);
        setConfirmVisible(false);
    };

    // 确认删除
    const confirmDelete = async () => {
        if (!userToDelete) return;
        try {
            const res = await fetch(`/user/delete?id=${userToDelete}`, {
                method: "GET",
            });
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to delete user");
                return;
            }
            showToast("User deleted", "success");
            setConfirmVisible(false);
            setUserToDelete(null);
            // 删除成功刷新数据，注意不要直接调用 fetchUsers() 导致死循环
            fetchUsers();
        } catch (error) {
            showToast("Delete failed: " + error.message);
        }
    };

    const handleSave = async () => {
        const url = editingUser ? "/user/update" : "/user/create";
        try {
            const res = await fetch(url, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(form),
            });

            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to save user");
                return;
            }

            await fetchUsers();
            showToast("User saved", "success");
            setModalVisible(false);
        } catch (error) {
            showToast("Save failed: " + error.message);
        }
    };

    const handlePageChange = (newPage) => {
        setPage(newPage);
    };

    const handleSearch = () => {
        setPage(1);
        fetchUsers();
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

    const visibleUserIds = users.map((user) => user.id);
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
                const res = await fetch(`/user/delete?id=${id}`, {
                    method: "GET",
                });
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

        setBatchDeleteVisible(false);
        clearSelection();

        if (failed > 0) {
            showToast(t("batch_operation_partial_failed", { success, failed }));
        } else {
            showToast(t("batch_operation_completed", { count: success }), "success");
        }

        await fetchUsers();
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
                <h2 className="text-2xl font-bold text-gray-800">{t("user_manage")}</h2>
                <button
                    onClick={handleAddClick}
                    className="bg-claw-600 hover:bg-claw-700 text-white px-4 py-2 rounded shadow"
                >
                    + {t("add_user")}
                </button>
            </div>

            <div className="flex mb-4 space-x-2">
                <input
                    type="text"
                    placeholder={t("username_placeholder")}
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="w-full sm:w-64 px-4 py-2 border border-gray-300 rounded shadow-sm focus:outline-none focus:ring focus:border-claw-400"
                />
                <button
                    onClick={handleSearch}
                    className="px-4 py-2 bg-claw-500 text-white rounded hover:bg-claw-600"
                >
                    {t("search")}
                </button>
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
                            {[t("id"), t("username"), t("create_time"), t("update_time"), t("action")].map((title) => (
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
                        {users.map((user) => (
                            <tr key={user.id} className="hover:bg-gray-50">
                                {selectionMode && (
                                    <td className="px-4 py-4 text-sm text-gray-800">
                                        <input
                                            type="checkbox"
                                            checked={selectedUserIds.includes(user.id)}
                                            onChange={() => toggleUserSelection(user.id)}
                                            className="h-4 w-4 rounded border-gray-300 text-claw-600 focus:ring-claw-500"
                                        />
                                    </td>
                                )}
                                <td className="px-6 py-4 text-sm text-gray-800">{user.id}</td>
                                <td className="px-6 py-4 text-sm text-gray-800">{user.username}</td>
                                <td className="px-6 py-4 text-sm text-gray-600">
                                    {new Date(user.create_time * 1000).toLocaleString()}
                                </td>
                                <td className="px-6 py-4 text-sm text-gray-600">
                                    {new Date(user.update_time * 1000).toLocaleString()}
                                </td>
                                <td className="px-6 py-4 space-x-2">
                                    <button
                                        onClick={() => handleEditClick(user)}
                                        className="text-claw-600 hover:underline text-sm"
                                    >
                                        {t("edit")}
                                    </button>
                                    <button
                                        onClick={() => handleDeleteClick(user.id)}
                                        className="text-red-600 hover:underline text-sm"
                                    >
                                        {t("delete")}
                                    </button>
                                </td>
                            </tr>
                        ))}
                        </tbody>
                    </table>
                </div>
            </div>

            <div className="shrink-0">
                <Pagination page={page} pageSize={pageSize} total={total} onPageChange={handlePageChange} />
            </div>

            <Modal
                visible={modalVisible}
                title={editingUser ? "Edit User" : "Add User"}
                onClose={() => setModalVisible(false)}
            >
                <input type="hidden" value={form.id} />

                <div className="mb-4">
                    <input
                        type="text"
                        placeholder="Username"
                        value={form.username}
                        onChange={(e) => setForm({ ...form, username: e.target.value })}
                        disabled={!!editingUser}
                        className="w-full px-4 py-2 border border-gray-300 rounded focus:outline-none focus:ring focus:border-claw-400"
                    />
                </div>

                <div className="mb-4">
                    <input
                        type="password"
                        placeholder="Password"
                        value={form.password}
                        onChange={(e) => setForm({ ...form, password: e.target.value })}
                        className="w-full px-4 py-2 border border-gray-300 rounded focus:outline-none focus:ring focus:border-claw-400"
                    />
                </div>

                <div className="flex justify-end space-x-2">
                    <button
                        onClick={() => setModalVisible(false)}
                        className="bg-gray-300 hover:bg-gray-400 text-gray-800 px-4 py-2 rounded"
                    >
                        {t("cancel")}
                    </button>
                    <button
                        onClick={handleSave}
                        className="bg-claw-600 hover:bg-claw-700 text-white px-4 py-2 rounded"
                    >
                        {t("save")}
                    </button>
                </div>
            </Modal>

            <ConfirmModal
                visible={confirmVisible}
                message="Are you sure you want to delete this user?"
                onConfirm={confirmDelete}
                onCancel={cancelDelete}
            />
            <ConfirmModal
                visible={batchDeleteVisible}
                message={t("batch_delete_confirm")}
                onConfirm={confirmBatchDelete}
                onCancel={() => setBatchDeleteVisible(false)}
            />
        </div>
    );
}

export default Users;
