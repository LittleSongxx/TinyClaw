import React, { useEffect, useMemo, useState } from "react";
import BotSelector from "../components/BotSelector";
import Pagination from "../components/Pagination";
import Toast from "../components/Toast";
import { useTranslation } from "react-i18next";

const SORT_OPTIONS = [
    { value: "high_used", labelKey: "high_used" },
    { value: "low_remaining", labelKey: "low_remaining" },
    { value: "usage_rate", labelKey: "usage_rate" },
    { value: "latest", labelKey: "latest_update" },
];

const PAGE_SIZES = [10, 20, 50];

function QuotaStats() {
    const { t } = useTranslation();
    const [bot, setBot] = useState(null);
    const [stats, setStats] = useState(null);
    const [loading, setLoading] = useState(false);
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(10);
    const [userId, setUserId] = useState("");
    const [sortBy, setSortBy] = useState("high_used");
    const [toast, setToast] = useState(null);

    useEffect(() => {
        if (!bot?.id) {
            setStats(null);
            return;
        }
        fetchStats(bot.id);
    }, [bot?.id, page, pageSize, sortBy]);

    const summaryCards = useMemo(() => {
        const summary = stats?.summary || {};
        return [
            { key: "total_users", label: t("user_num"), value: summary.total_users ?? 0 },
            { key: "quota", label: t("total_quota_token"), value: summary.total_quota_token ?? 0 },
            { key: "used", label: t("total_used_token"), value: summary.total_used_token ?? 0 },
            { key: "remaining", label: t("total_remaining_token"), value: summary.total_remaining_token ?? 0 },
            {
                key: "usage_rate",
                label: t("average_usage_rate"),
                value: `${Math.round((summary.average_usage_rate ?? 0) * 100)}%`,
            },
        ];
    }, [stats, t]);

    const fetchStats = async (botId, nextUserId = userId.trim()) => {
        setLoading(true);
        try {
            const params = new URLSearchParams({
                id: botId,
                page: String(page),
                pageSize: String(pageSize),
                sortBy,
            });
            if (nextUserId) {
                params.set("userId", nextUserId);
            }
            const response = await fetch(`/bot/user/quota/stats?${params.toString()}`);
            const data = await response.json();
            if (data.code !== 0) {
                throw new Error(data.message || "Failed to fetch quota stats");
            }
            setStats(data.data || null);
        } catch (err) {
            setToast({ message: err.message || "Failed to fetch quota stats", type: "error" });
        } finally {
            setLoading(false);
        }
    };

    const handleSearch = () => {
        if (!bot?.id) {
            setToast({ message: t("bot_choose"), type: "error" });
            return;
        }
        setPage(1);
        fetchStats(bot.id, userId.trim());
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
                        <h1 className="text-2xl font-semibold text-gray-900">{t("quota_stats_manage")}</h1>
                        <p className="mt-1 text-sm text-gray-600">{t("quota_stats_desc")}</p>
                    </div>
                    <div className="w-full xl:w-96">
                        <BotSelector value={bot} onChange={setBot} />
                    </div>
                </div>

                <div className="grid gap-3 md:grid-cols-[1.2fr,1fr,1fr,auto]">
                    <div>
                        <label className="mb-1 block text-sm font-medium text-gray-700">{t("search_user_id")}</label>
                        <input
                            type="text"
                            value={userId}
                            onChange={(event) => setUserId(event.target.value)}
                            placeholder={t("user_id_placeholder")}
                            className="w-full rounded-lg border border-gray-300 px-4 py-2 shadow-sm focus:border-claw-400 focus:outline-none focus:ring"
                        />
                    </div>
                    <div>
                        <label className="mb-1 block text-sm font-medium text-gray-700">{t("sort_by")}</label>
                        <select
                            value={sortBy}
                            onChange={(event) => {
                                setSortBy(event.target.value);
                                setPage(1);
                            }}
                            className="w-full rounded-lg border border-gray-300 px-4 py-2 shadow-sm focus:border-claw-400 focus:outline-none focus:ring"
                        >
                            {SORT_OPTIONS.map((item) => (
                                <option key={item.value} value={item.value}>
                                    {t(item.labelKey)}
                                </option>
                            ))}
                        </select>
                    </div>
                    <div>
                        <label className="mb-1 block text-sm font-medium text-gray-700">{t("page_size")}</label>
                        <select
                            value={pageSize}
                            onChange={(event) => {
                                setPageSize(Number(event.target.value));
                                setPage(1);
                            }}
                            className="w-full rounded-lg border border-gray-300 px-4 py-2 shadow-sm focus:border-claw-400 focus:outline-none focus:ring"
                        >
                            {PAGE_SIZES.map((size) => (
                                <option key={size} value={size}>
                                    {size}
                                </option>
                            ))}
                        </select>
                    </div>
                    <div className="flex items-end">
                        <button
                            type="button"
                            onClick={handleSearch}
                            className="rounded-lg bg-claw-600 px-4 py-2 font-semibold text-white hover:bg-claw-700"
                        >
                            {t("search")}
                        </button>
                    </div>
                </div>
            </section>

            <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
                {summaryCards.map((card) => (
                    <div key={card.key} className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                        <div className="text-sm text-slate-500">{card.label}</div>
                        <div className="mt-2 text-2xl font-semibold text-slate-900">{card.value}</div>
                    </div>
                ))}
            </section>

            <section className="grid gap-6 xl:grid-cols-[1.2fr,0.8fr]">
                <div className="rounded-xl bg-white p-6 shadow space-y-4">
                    <div className="flex items-center justify-between">
                        <h2 className="text-lg font-semibold text-gray-900">{t("usage_distribution")}</h2>
                        <button
                            type="button"
                            onClick={() => bot?.id && fetchStats(bot.id)}
                            className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white hover:bg-slate-800"
                        >
                            {loading ? t("loading") : t("refresh")}
                        </button>
                    </div>

                    <div className="space-y-3">
                        {(stats?.distribution || []).length === 0 && (
                            <div className="rounded-lg border border-dashed border-slate-300 p-4 text-sm text-slate-500">
                                {t("distribution_empty")}
                            </div>
                        )}

                        {(stats?.distribution || []).map((bucket) => {
                            const totalUsers = stats?.summary?.total_users || 0;
                            const percent = totalUsers > 0 ? Math.round((bucket.count / totalUsers) * 100) : 0;
                            return (
                                <div key={bucket.label} className="space-y-1">
                                    <div className="flex items-center justify-between text-sm text-slate-700">
                                        <span>{bucket.label}</span>
                                        <span>{bucket.count} / {percent}%</span>
                                    </div>
                                    <div className="h-3 rounded-full bg-slate-100">
                                        <div
                                            className="h-3 rounded-full bg-claw-600 transition-all"
                                            style={{ width: `${percent}%` }}
                                        />
                                    </div>
                                </div>
                            );
                        })}
                    </div>
                </div>

                <div className="space-y-6">
                    <MetricListCard
                        title={t("top_used_users")}
                        items={stats?.top_used || []}
                    />
                    <MetricListCard
                        title={t("lowest_remaining_users")}
                        items={stats?.lowest_remaining || []}
                    />
                </div>
            </section>

            <section className="rounded-xl bg-white p-6 shadow space-y-4">
                <div className="flex items-center justify-between">
                    <h2 className="text-lg font-semibold text-gray-900">{t("quota_rankings")}</h2>
                    <div className="text-sm text-slate-500">
                        {t("selected_items", { count: stats?.total || 0 })}
                    </div>
                </div>

                <div className="overflow-auto rounded-lg border border-slate-200">
                    <table className="min-w-full divide-y divide-slate-200">
                        <thead className="bg-slate-50">
                            <tr>
                                {[
                                    t("user_id"),
                                    t("total_used_token"),
                                    t("total_quota_token"),
                                    t("total_remaining_token"),
                                    t("usage_rate"),
                                    t("latest_update"),
                                ].map((title) => (
                                    <th key={title} className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                                        {title}
                                    </th>
                                ))}
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100 bg-white">
                            {(stats?.list || []).length === 0 && (
                                <tr>
                                    <td colSpan={6} className="px-4 py-8 text-center text-sm text-slate-500">
                                        {t("no_data")}
                                    </td>
                                </tr>
                            )}
                            {(stats?.list || []).map((item) => (
                                <tr key={item.user_id} className="hover:bg-slate-50">
                                    <td className="px-4 py-3 text-sm text-slate-900">{item.user_id}</td>
                                    <td className="px-4 py-3 text-sm text-slate-700">{item.token}</td>
                                    <td className="px-4 py-3 text-sm text-slate-700">{item.avail_token}</td>
                                    <td className="px-4 py-3 text-sm text-slate-700">{item.remaining_token}</td>
                                    <td className="px-4 py-3 text-sm text-slate-700">
                                        {Math.round((item.usage_rate || 0) * 100)}%
                                    </td>
                                    <td className="px-4 py-3 text-sm text-slate-700">{formatUnix(item.update_time)}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>

                <Pagination
                    page={page}
                    pageSize={pageSize}
                    total={stats?.total || 0}
                    onPageChange={setPage}
                />
            </section>
        </div>
    );
}

function MetricListCard({ title, items }) {
    return (
        <div className="rounded-xl bg-white p-6 shadow space-y-4">
            <h2 className="text-lg font-semibold text-gray-900">{title}</h2>
            <div className="space-y-3">
                {items.length === 0 && (
                    <div className="rounded-lg border border-dashed border-slate-300 p-4 text-sm text-slate-500">
                        No data
                    </div>
                )}
                {items.map((item) => (
                    <div key={`${title}-${item.user_id}`} className="rounded-xl border border-slate-200 p-4">
                        <div className="font-medium text-slate-900">{item.user_id}</div>
                        <div className="mt-2 grid gap-1 text-sm text-slate-600">
                            <div>used: {item.token}</div>
                            <div>quota: {item.avail_token}</div>
                            <div>remaining: {item.remaining_token}</div>
                            <div>usage: {Math.round((item.usage_rate || 0) * 100)}%</div>
                        </div>
                    </div>
                ))}
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

export default QuotaStats;
