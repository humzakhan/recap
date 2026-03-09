"use client";

import { ExtractionForm } from "@/components/ExtractionForm";
import { ResultView } from "@/components/ResultView";
import { useExtractionStore } from "@/lib/stores";

export default function Home() {
  const { result, fields, error } = useExtractionStore();

  return (
    <div className="max-w-4xl mx-auto px-6 py-8">
      <ExtractionForm />

      {error && (
        <div className="mt-6 p-4 bg-red-50 border border-red-200 rounded text-red-700 text-sm">
          {error}
        </div>
      )}

      {result && (
        <div className="mt-8">
          <ResultView result={result} fields={fields} />
        </div>
      )}
    </div>
  );
}
