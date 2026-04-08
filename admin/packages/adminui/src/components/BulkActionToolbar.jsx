import React from "react";
import { useTranslation } from "react-i18next";

export default function BulkActionToolbar({
    selectionMode,
    onToggleMode,
    selectedCount,
    onSelectAllVisible,
    onClearSelection,
    actions,
    className = "",
}) {
    const { t } = useTranslation();

    return (
        <div className={`rounded-xl border border-gray-200 bg-white p-3 shadow-sm ${className}`}>
            <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex flex-wrap items-center gap-3">
                    <button
                        type="button"
                        onClick={onToggleMode}
                        className={`rounded-lg px-4 py-2 text-sm font-semibold transition ${
                            selectionMode
                                ? "bg-gray-900 text-white hover:bg-gray-800"
                                : "border border-claw-200 bg-claw-50 text-claw-700 hover:border-claw-300 hover:bg-claw-100"
                        }`}
                    >
                        {selectionMode ? t("exit_multi_select") : t("multi_select")}
                    </button>

                    {selectionMode && (
                        <span className="text-sm font-medium text-gray-600">
                            {t("selected_items", { count: selectedCount })}
                        </span>
                    )}
                </div>

                {selectionMode && (
                    <div className="flex flex-wrap items-center gap-2">
                        <button
                            type="button"
                            onClick={onSelectAllVisible}
                            className="rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm font-medium text-gray-700 transition hover:border-gray-300 hover:bg-gray-50"
                        >
                            {t("select_all_visible")}
                        </button>
                        <button
                            type="button"
                            onClick={onClearSelection}
                            className="rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm font-medium text-gray-700 transition hover:border-gray-300 hover:bg-gray-50"
                        >
                            {t("clear_selection")}
                        </button>

                        {actions}
                    </div>
                )}
            </div>
        </div>
    );
}
