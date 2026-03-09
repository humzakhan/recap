"use client";

import { useState, useEffect } from "react";
import { getTemplates } from "@/lib/api";
import type { Template } from "@/lib/api";

export default function TemplatesPage() {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getTemplates()
      .then(setTemplates)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="max-w-4xl mx-auto px-6 py-8">
      <h1 className="text-2xl font-bold mb-6">Templates</h1>
      {loading && <p className="text-gray-500">Loading templates...</p>}
      {error && <p className="text-red-600">{error}</p>}
      {!loading && templates.length === 0 && <p className="text-gray-500">No templates found.</p>}
      <div className="space-y-4">
        {templates.map((t) => (
          <div key={t.name} className="border rounded-lg p-4">
            <h2 className="font-semibold">{t.name}</h2>
            {t.description && <p className="text-sm text-gray-500 mt-1">{t.description}</p>}
            <div className="mt-2 flex flex-wrap gap-2">
              {t.fields.map((f) => (
                <span key={f.label} className="text-xs bg-gray-100 px-2 py-1 rounded">
                  {f.label}: {f.type}
                </span>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
