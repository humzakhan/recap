"use client";

import { HistoryList } from "@/components/HistoryList";

export default function HistoryPage() {
  return (
    <div className="max-w-4xl mx-auto px-6 py-8">
      <h1 className="text-2xl font-bold mb-6">Extraction History</h1>
      <HistoryList />
    </div>
  );
}
