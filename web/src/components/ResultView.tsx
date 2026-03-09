"use client";

import { useState } from "react";
import type { ExtractionResult, Field } from "@/lib/api";

interface ResultViewProps {
  result: ExtractionResult;
  fields: Field[];
}

const TABS = ["Fields", "JSON", "Markdown", "Raw"] as const;
type Tab = (typeof TABS)[number];

function toSnakeCase(label: string): string {
  return label.toLowerCase().replace(/\s+/g, "_");
}

function FieldCard({ label, value }: { label: string; value: unknown }) {
  const renderValue = () => {
    if (value === null || value === undefined) {
      return <span className="italic text-gray-400">Not available</span>;
    }
    if (typeof value === "boolean") {
      return <span>{value ? "Yes" : "No"}</span>;
    }
    if (Array.isArray(value)) {
      return (
        <ul className="list-inside list-disc space-y-0.5">
          {value.map((item, i) => (
            <li key={i}>{String(item)}</li>
          ))}
        </ul>
      );
    }
    return <span className="whitespace-pre-wrap">{String(value)}</span>;
  };

  return (
    <div className="rounded-md border border-gray-200 px-4 py-3">
      <p className="text-xs font-medium uppercase tracking-wide text-gray-500">
        {label}
      </p>
      <div className="mt-1 text-sm text-gray-900">{renderValue()}</div>
    </div>
  );
}

export function ResultView({ result, fields }: ResultViewProps) {
  const [tab, setTab] = useState<Tab>("Fields");
  const [copied, setCopied] = useState(false);

  const copyJson = () => {
    navigator.clipboard.writeText(JSON.stringify(result.data, null, 2));
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div>
      <div className="flex border-b border-gray-200">
        {TABS.map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-4 py-2 text-sm font-medium ${
              tab === t
                ? "border-b-2 border-blue-500 text-blue-600"
                : "text-gray-500 hover:text-gray-700"
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      <div className="mt-4">
        {tab === "Fields" && (
          <div className="grid gap-3 sm:grid-cols-2">
            {fields.map((field) => (
              <FieldCard
                key={field.label}
                label={field.label}
                value={result.data[toSnakeCase(field.label)] ?? null}
              />
            ))}
          </div>
        )}

        {tab === "JSON" && (
          <div className="relative">
            <button
              onClick={copyJson}
              className="absolute right-2 top-2 rounded bg-gray-100 px-2 py-1 text-xs text-gray-600 hover:bg-gray-200"
            >
              {copied ? "Copied!" : "Copy"}
            </button>
            <pre className="overflow-x-auto rounded-md bg-gray-50 p-4 text-sm text-gray-800">
              {JSON.stringify(result.data, null, 2)}
            </pre>
          </div>
        )}

        {tab === "Markdown" && (
          <pre className="overflow-x-auto whitespace-pre-wrap rounded-md bg-gray-50 p-4 text-sm text-gray-800">
            {result.formatted || "No formatted output available."}
          </pre>
        )}

        {tab === "Raw" && (
          <pre className="max-h-96 overflow-auto whitespace-pre-wrap rounded-md bg-gray-50 p-4 text-sm text-gray-700">
            {result.raw_content || "No raw content available."}
          </pre>
        )}
      </div>

      <p className="mt-4 text-xs text-gray-400">
        Model: {result.model} | Tokens: {result.tokens_used} | Type:{" "}
        {result.content_type}
      </p>
    </div>
  );
}
