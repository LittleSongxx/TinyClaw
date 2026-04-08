import React, { useEffect, useMemo, useState } from "react";
import BotSelector from "../components/BotSelector";
import Toast from "../components/Toast.jsx";
import { useTranslation } from "react-i18next";
import { getSkillDescription, getSkillSection } from "../utils/skillDescriptions.js";

function formatList(values) {
    if (!values || values.length === 0) {
        return "-";
    }
    return values.join(", ");
}

function sourceBadgeClass(source) {
    switch (source) {
        case "local":
            return "border-emerald-200 bg-emerald-50 text-emerald-700";
        case "builtin":
            return "border-sky-200 bg-sky-50 text-sky-700";
        case "legacy":
            return "border-amber-200 bg-amber-50 text-amber-700";
        default:
            return "border-gray-200 bg-gray-50 text-gray-700";
    }
}

function SectionBlock({ title, value }) {
    return (
        <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
            <div className="mb-2 text-sm font-semibold text-gray-700">{title}</div>
            <div className="whitespace-pre-wrap text-sm leading-6 text-gray-600">{value || "-"}</div>
        </div>
    );
}

export default function Skills() {
    const { t } = useTranslation();
    const [botId, setBotId] = useState(null);
    const [skills, setSkills] = useState([]);
    const [selectedSkill, setSelectedSkill] = useState(null);
    const [validation, setValidation] = useState(null);
    const [warnings, setWarnings] = useState([]);
    const [search, setSearch] = useState("");
    const [sourceFilter, setSourceFilter] = useState("");
    const [descriptionLanguage, setDescriptionLanguage] = useState("zh");
    const [loading, setLoading] = useState(false);
    const [detailLoading, setDetailLoading] = useState(false);
    const [actionLoading, setActionLoading] = useState(false);
    const [toast, setToast] = useState({ show: false, message: "", type: "error" });

    const showToast = (message, type = "error") => {
        setToast({ show: true, message, type });
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

    const filteredSkills = useMemo(() => {
        const normalizedSearch = search.trim().toLowerCase();

        return skills.filter((item) => {
            const localizedDescription = getSkillDescription(item, descriptionLanguage).toLowerCase();
            const matchesSearch =
                normalizedSearch === "" ||
                item?.manifest?.id?.toLowerCase().includes(normalizedSearch) ||
                item?.manifest?.name?.toLowerCase().includes(normalizedSearch) ||
                localizedDescription.includes(normalizedSearch);
            const matchesSource = sourceFilter === "" || item?.source === sourceFilter;
            return matchesSearch && matchesSource;
        });
    }, [descriptionLanguage, search, skills, sourceFilter]);

    const currentWarnings = validation?.warnings || warnings;
    const stats = validation || {
        total: skills.length,
        local_count: skills.filter((item) => item.source === "local").length,
        builtin_count: skills.filter((item) => item.source === "builtin").length,
        legacy_count: skills.filter((item) => item.source === "legacy").length,
    };

    useEffect(() => {
        if (botId !== null) {
            fetchSkills();
            validateSkills(false);
        }
    }, [botId]);

    const fetchSkills = async () => {
        try {
            setLoading(true);
            const res = await fetch(`/bot/skills/list?id=${botId}`);
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to fetch skills");
                return;
            }

            const list = data.data?.list || [];
            setSkills(list);
            setWarnings(data.data?.warnings || []);

            const nextSkillId = selectedSkill?.manifest?.id;
            const nextSelected = list.find((item) => item?.manifest?.id === nextSkillId) || list[0] || null;
            setSelectedSkill(nextSelected);
        } catch (err) {
            showToast(`Failed to fetch skills: ${err.message}`);
        } finally {
            setLoading(false);
        }
    };

    const validateSkills = async (showSuccess = true) => {
        try {
            setActionLoading(true);
            const res = await fetch(`/bot/skills/validate?id=${botId}`);
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to validate skills");
                return;
            }

            setValidation(data.data);
            if (showSuccess) {
                showToast(t("skill_validation_loaded"), "success");
            }
        } catch (err) {
            showToast(`Failed to validate skills: ${err.message}`);
        } finally {
            setActionLoading(false);
        }
    };

    const reloadSkills = async () => {
        try {
            setActionLoading(true);
            const res = await fetch(`/bot/skills/reload?id=${botId}`, { method: "POST" });
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to reload skills");
                return;
            }

            showToast(t("skill_reloaded"), "success");
            await fetchSkills();
            await validateSkills(false);
        } catch (err) {
            showToast(`Failed to reload skills: ${err.message}`);
        } finally {
            setActionLoading(false);
        }
    };

    const openSkillDetail = async (skillId) => {
        try {
            setDetailLoading(true);
            const params = new URLSearchParams({ id: botId, skill_id: skillId });
            const res = await fetch(`/bot/skills/detail?${params.toString()}`);
            const data = await res.json();
            if (data.code !== 0) {
                showToast(data.message || "Failed to fetch skill detail");
                return;
            }
            setSelectedSkill(data.data);
        } catch (err) {
            showToast(`Failed to fetch skill detail: ${err.message}`);
        } finally {
            setDetailLoading(false);
        }
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

            <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
                <div>
                    <h2 className="text-2xl font-bold text-gray-800">{t("skills_manage")}</h2>
                    <p className="mt-1 text-sm text-gray-500">{t("skills_manage_desc")}</p>
                </div>

                <div className="flex items-center gap-3">
                    {renderDescriptionLanguageToggle(descriptionLanguage, setDescriptionLanguage)}
                    <button
                        onClick={() => validateSkills(true)}
                        disabled={!botId || actionLoading}
                        className="rounded-lg border border-claw-200 bg-white px-4 py-2 text-sm font-semibold text-claw-700 shadow-sm transition hover:border-claw-300 hover:bg-claw-50 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                        {t("validate_skills")}
                    </button>
                    <button
                        onClick={reloadSkills}
                        disabled={!botId || actionLoading}
                        className="rounded-lg bg-claw-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition hover:bg-claw-700 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                        {t("reload_skills")}
                    </button>
                </div>
            </div>

            <div className="mb-6 grid gap-4 rounded-lg bg-white p-4 shadow md:grid-cols-4">
                <BotSelector
                    value={botId}
                    onChange={(bot) => {
                        setBotId(bot.id);
                    }}
                />

                <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">{t("search")}</label>
                    <input
                        type="text"
                        value={search}
                        onChange={(event) => setSearch(event.target.value)}
                        placeholder={t("search_skill")}
                        className="w-full rounded border border-gray-300 px-3 py-2"
                    />
                </div>

                <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">{t("source")}</label>
                    <select
                        value={sourceFilter}
                        onChange={(event) => setSourceFilter(event.target.value)}
                        className="w-full rounded border border-gray-300 px-3 py-2"
                    >
                        <option value="">{t("all_status")}</option>
                        <option value="local">{t("local_skills")}</option>
                        <option value="builtin">{t("builtin_skills")}</option>
                        <option value="legacy">{t("legacy_skills")}</option>
                    </select>
                </div>

                <div className="grid grid-cols-2 gap-3 text-sm">
                    <div className="rounded-lg border border-gray-200 bg-gray-50 p-3">
                        <div className="text-gray-500">{t("skills")}</div>
                        <div className="mt-1 text-lg font-semibold text-gray-800">{stats.total || 0}</div>
                    </div>
                    <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-3">
                        <div className="text-emerald-700">{t("local_skills")}</div>
                        <div className="mt-1 text-lg font-semibold text-emerald-800">{stats.local_count || 0}</div>
                    </div>
                    <div className="rounded-lg border border-sky-200 bg-sky-50 p-3">
                        <div className="text-sky-700">{t("builtin_skills")}</div>
                        <div className="mt-1 text-lg font-semibold text-sky-800">{stats.builtin_count || 0}</div>
                    </div>
                    <div className="rounded-lg border border-amber-200 bg-amber-50 p-3">
                        <div className="text-amber-700">{t("legacy_skills")}</div>
                        <div className="mt-1 text-lg font-semibold text-amber-800">{stats.legacy_count || 0}</div>
                    </div>
                </div>
            </div>

            {currentWarnings.length > 0 && (
                <div className="mb-6 rounded-lg border border-amber-200 bg-amber-50 p-4 shadow-sm">
                    <div className="mb-2 text-sm font-semibold text-amber-800">{t("validation_warnings")}</div>
                    <div className="space-y-2 text-sm text-amber-900">
                        {currentWarnings.map((warning) => (
                            <div key={warning} className="rounded bg-white/70 px-3 py-2">
                                {warning}
                            </div>
                        ))}
                    </div>
                </div>
            )}

            <div className="grid min-h-0 flex-1 gap-6 xl:grid-cols-12">
                <div className="min-h-0 xl:col-span-5">
                    <div className="overflow-hidden rounded-xl bg-white shadow">
                        <div className="border-b border-gray-100 px-4 py-3 text-sm font-semibold text-gray-700">
                            {t("skills")}
                        </div>

                        <div className="max-h-[70vh] overflow-auto">
                            {loading ? (
                                <div className="p-6 text-sm text-gray-500">{t("loading")}</div>
                            ) : filteredSkills.length === 0 ? (
                                <div className="p-6 text-sm text-gray-500">{t("no_data")}</div>
                            ) : (
                                filteredSkills.map((item) => {
                                    const isActive = selectedSkill?.manifest?.id === item?.manifest?.id;
                                    return (
                                        <button
                                            key={item?.manifest?.id}
                                            onClick={() => openSkillDetail(item?.manifest?.id)}
                                            className={`w-full border-b border-gray-100 px-4 py-4 text-left transition hover:bg-gray-50 ${
                                                isActive ? "bg-claw-50" : "bg-white"
                                            }`}
                                        >
                                            <div className="flex items-start justify-between gap-3">
                                                <div>
                                                    <div className="text-sm font-semibold text-gray-800">
                                                        {item?.manifest?.name || item?.manifest?.id}
                                                    </div>
                                                    <div className="mt-1 text-xs text-gray-500">{item?.manifest?.id}</div>
                                                </div>
                                                <span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${sourceBadgeClass(item?.source)}`}>
                                                    {item?.source}
                                                </span>
                                            </div>

                                            <div className="mt-3 text-sm text-gray-600">
                                                {getSkillDescription(item, descriptionLanguage)}
                                            </div>

                                            <div className="mt-3 flex flex-wrap gap-2 text-xs text-gray-500">
                                                {(item?.manifest?.modes || []).map((mode) => (
                                                    <span key={`${item?.manifest?.id}-${mode}`} className="rounded-full bg-gray-100 px-2 py-1">
                                                        {mode}
                                                    </span>
                                                ))}
                                                <span className="rounded-full bg-gray-100 px-2 py-1">
                                                    {t("memory_mode")}: {item?.manifest?.memory || "-"}
                                                </span>
                                            </div>
                                        </button>
                                    );
                                })
                            )}
                        </div>
                    </div>
                </div>

                <div className="min-h-0 overflow-auto xl:col-span-7">
                    <div className="space-y-4 pr-1">
                        <div className="rounded-xl bg-white p-5 shadow">
                            <div className="mb-4 flex items-start justify-between gap-4">
                                <div>
                                    <div className="text-lg font-semibold text-gray-800">{t("skill_detail")}</div>
                                    <div className="mt-1 text-sm text-gray-500">
                                        {detailLoading ? t("loading") : selectedSkill?.manifest?.id || "-"}
                                    </div>
                                </div>
                                {selectedSkill && (
                                    <span className={`inline-flex rounded-full border px-3 py-1 text-xs font-semibold ${sourceBadgeClass(selectedSkill?.source)}`}>
                                        {selectedSkill?.source}
                                    </span>
                                )}
                            </div>

                            {selectedSkill ? (
                                <div className="grid gap-4 md:grid-cols-2">
                                    <SectionBlock
                                        title={t("description")}
                                        value={getSkillDescription(selectedSkill, descriptionLanguage)}
                                    />
                                    <SectionBlock title={t("path_label")} value={selectedSkill?.path} />
                                    <SectionBlock title={t("modes")} value={formatList(selectedSkill?.manifest?.modes)} />
                                    <SectionBlock title={t("memory_mode")} value={selectedSkill?.manifest?.memory} />
                                    <SectionBlock title={t("allowed_servers_label")} value={formatList(selectedSkill?.manifest?.allowed_servers)} />
                                    <SectionBlock title={t("allowed_tools_label")} value={formatList(selectedSkill?.manifest?.allowed_tools)} />
                                    <SectionBlock title={t("triggers")} value={formatList(selectedSkill?.manifest?.triggers)} />
                                    <SectionBlock title={t("priority_label")} value={String(selectedSkill?.manifest?.priority ?? "-")} />
                                </div>
                            ) : (
                                <div className="text-sm text-gray-500">{t("no_data")}</div>
                            )}
                        </div>

                        {selectedSkill && (
                            <div className="grid gap-4">
                                <SectionBlock
                                    title={t("when_to_use")}
                                    value={getSkillSection(selectedSkill, "when_to_use", descriptionLanguage)}
                                />
                                <SectionBlock
                                    title={t("when_not_to_use")}
                                    value={getSkillSection(selectedSkill, "when_not_to_use", descriptionLanguage)}
                                />
                                <SectionBlock
                                    title={t("instructions_label")}
                                    value={getSkillSection(selectedSkill, "instructions", descriptionLanguage)}
                                />
                                <SectionBlock
                                    title={t("output_contract_label")}
                                    value={getSkillSection(selectedSkill, "output_contract", descriptionLanguage)}
                                />
                                <SectionBlock
                                    title={t("failure_handling_label")}
                                    value={getSkillSection(selectedSkill, "failure_handling", descriptionLanguage)}
                                />
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}
