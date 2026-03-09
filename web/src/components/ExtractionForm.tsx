"use client";

import { extract } from "@/lib/api";
import { useExtractionStore } from "@/lib/stores";
import { useHistoryStore } from "@/lib/stores";
import { FieldBuilder } from "./FieldBuilder";
import { TemplateSelector } from "./TemplateSelector";
import type { Field } from "@/lib/api";

export function ExtractionForm() {
  const {
    url,
    fields,
    status,
    setUrl,
    setFields,
    setLoading,
    setSuccess,
    setError,
  } = useExtractionStore();
  const { addEntry } = useHistoryStore();

  const canSubmit =
    status !== "loading" &&
    url.trim().length > 0 &&
    fields.length > 0 &&
    fields.every((f) => f.label.trim().length > 0);

  const handleTemplateSelect = (templateFields: Field[]) => {
    if (templateFields.length > 0) {
      setFields(templateFields);
    }
  };

  const handleSubmit = async () => {
    setLoading();
    try {
      const result = await extract({ url, fields, format: "json" });
      setSuccess(result);
      addEntry(url, fields, result);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Extraction failed");
    }
  };

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-gray-700">URL</label>
        <input
          type="url"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://example.com/article"
          className="mt-1 w-full rounded-md border border-gray-300 px-4 py-3 text-base focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
        />
      </div>

      <TemplateSelector onSelect={handleTemplateSelect} />

      <FieldBuilder fields={fields} onFieldsChange={setFields} />

      {status === "error" && (
        <p className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-600">
          {useExtractionStore.getState().error}
        </p>
      )}

      <button
        onClick={handleSubmit}
        disabled={!canSubmit}
        className="w-full rounded-md bg-blue-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-gray-300"
      >
        {status === "loading" ? (
          <span className="flex items-center justify-center gap-2">
            <svg
              className="h-4 w-4 animate-spin"
              viewBox="0 0 24 24"
              fill="none"
            >
              <circle
                className="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="4"
              />
              <path
                className="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"
              />
            </svg>
            Extracting from {url}...
          </span>
        ) : (
          "Extract"
        )}
      </button>
    </div>
  );
}
