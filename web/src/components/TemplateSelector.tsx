"use client";

import { useState, useEffect } from "react";
import { getTemplates } from "@/lib/api";
import type { Template, Field } from "@/lib/api";

interface TemplateSelectorProps {
  onSelect: (fields: Field[]) => void;
}

export function TemplateSelector({ onSelect }: TemplateSelectorProps) {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [selected, setSelected] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getTemplates()
      .then((data) => {
        setTemplates(data);
        setLoading(false);
      })
      .catch((err) => {
        setError(err.message);
        setLoading(false);
      });
  }, []);

  const handleChange = (name: string) => {
    setSelected(name);
    if (!name) {
      onSelect([]);
      return;
    }
    const tmpl = templates.find((t) => t.name === name);
    if (tmpl) onSelect(tmpl.fields);
  };

  const selectedTemplate = templates.find((t) => t.name === selected);

  return (
    <div className="space-y-2">
      <label className="block text-sm font-medium text-gray-700">
        Template
      </label>
      {loading ? (
        <p className="text-sm text-gray-400">Loading templates...</p>
      ) : error ? (
        <p className="text-sm text-red-500">Failed to load templates: {error}</p>
      ) : (
        <>
          <select
            value={selected}
            onChange={(e) => handleChange(e.target.value)}
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            <option value="">None</option>
            {templates.map((t) => (
              <option key={t.name} value={t.name}>
                {t.name}
                {t.description ? ` — ${t.description}` : ""}
              </option>
            ))}
          </select>

          {selectedTemplate && (
            <div className="rounded-md bg-gray-50 px-3 py-2 text-sm text-gray-600">
              <p className="font-medium">{selectedTemplate.name}</p>
              <p className="mt-1 text-xs text-gray-400">
                {selectedTemplate.fields.map((f) => f.label).join(", ")}
              </p>
            </div>
          )}
        </>
      )}
    </div>
  );
}
