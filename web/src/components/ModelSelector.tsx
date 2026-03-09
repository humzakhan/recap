"use client";

import { useEffect, useState } from "react";
import { getModels } from "@/lib/api";
import type { ModelInfo } from "@/lib/api";

interface ModelSelectorProps {
  selectedModel: string;
  onModelChange: (model: string) => void;
}

export function ModelSelector({ selectedModel, onModelChange }: ModelSelectorProps) {
  const [models, setModels] = useState<ModelInfo[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getModels()
      .then((resp) => {
        setModels(resp.models);
        if (!selectedModel && resp.current) {
          onModelChange(resp.current);
        }
      })
      .catch(() => {
        // If the backend is not running, show empty selector.
      })
      .finally(() => setLoading(false));
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Group models by provider.
  const grouped = models.reduce<Record<string, ModelInfo[]>>((acc, m) => {
    if (!acc[m.provider]) acc[m.provider] = [];
    acc[m.provider].push(m);
    return acc;
  }, {});

  const providerLabel = (provider: string) => {
    switch (provider) {
      case "anthropic": return "Anthropic";
      case "openai": return "OpenAI";
      case "google": return "Google";
      default: return provider;
    }
  };

  if (loading) {
    return (
      <div>
        <label className="block text-sm font-medium text-gray-700">Model</label>
        <select disabled className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm text-gray-400">
          <option>Loading models...</option>
        </select>
      </div>
    );
  }

  return (
    <div>
      <label className="block text-sm font-medium text-gray-700">Model</label>
      <select
        value={selectedModel}
        onChange={(e) => onModelChange(e.target.value)}
        className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
      >
        {Object.entries(grouped).map(([provider, providerModels]) => (
          <optgroup key={provider} label={providerLabel(provider)}>
            {providerModels.map((m) => (
              <option key={m.id} value={m.id}>
                {m.name}
              </option>
            ))}
          </optgroup>
        ))}
      </select>
    </div>
  );
}
