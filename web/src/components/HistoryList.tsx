"use client";

import { useHistoryStore, type HistoryEntry } from "@/lib/stores";
import { useExtractionStore } from "@/lib/stores";

function truncateUrl(url: string, max = 60): string {
  return url.length > max ? url.slice(0, max) + "..." : url;
}

function formatTimestamp(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function HistoryRow({ entry }: { entry: HistoryEntry }) {
  const { setUrl, setFields } = useExtractionStore();
  const { removeEntry } = useHistoryStore();

  const rerun = () => {
    setUrl(entry.url);
    setFields(entry.fields);
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  return (
    <div className="flex items-center justify-between gap-3 rounded-md border border-gray-200 px-4 py-3">
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-gray-800">
          {truncateUrl(entry.url)}
        </p>
        <p className="text-xs text-gray-400">
          {entry.fields.length} field{entry.fields.length !== 1 ? "s" : ""}{" "}
          &middot; {formatTimestamp(entry.timestamp)}
        </p>
      </div>
      <div className="flex gap-1">
        <button
          onClick={rerun}
          className="rounded px-2 py-1 text-xs text-blue-600 hover:bg-blue-50"
        >
          Re-run
        </button>
        <button
          onClick={() => removeEntry(entry.id)}
          className="rounded px-2 py-1 text-xs text-gray-400 hover:bg-red-50 hover:text-red-500"
        >
          Remove
        </button>
      </div>
    </div>
  );
}

export function HistoryList() {
  const { entries, clearHistory } = useHistoryStore();

  if (entries.length === 0) {
    return (
      <p className="py-8 text-center text-sm text-gray-400">
        No extractions yet. Run one to see it here.
      </p>
    );
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-gray-700">History</h3>
        <button
          onClick={clearHistory}
          className="text-xs text-gray-400 hover:text-red-500"
        >
          Clear all
        </button>
      </div>
      {entries.map((entry) => (
        <HistoryRow key={entry.id} entry={entry} />
      ))}
    </div>
  );
}
