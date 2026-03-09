import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { Field, ExtractionResult } from "./api";

// Extraction job state
type ExtractionStatus = "idle" | "loading" | "success" | "error";

interface ExtractionState {
  url: string;
  fields: Field[];
  status: ExtractionStatus;
  result: ExtractionResult | null;
  error: string | null;
  setUrl: (url: string) => void;
  setFields: (fields: Field[]) => void;
  addField: (field: Field) => void;
  removeField: (index: number) => void;
  updateField: (index: number, field: Field) => void;
  setLoading: () => void;
  setSuccess: (result: ExtractionResult) => void;
  setError: (error: string) => void;
  reset: () => void;
}

export const useExtractionStore = create<ExtractionState>((set) => ({
  url: "",
  fields: [],
  status: "idle",
  result: null,
  error: null,
  setUrl: (url) => set({ url }),
  setFields: (fields) => set({ fields }),
  addField: (field) => set((state) => ({ fields: [...state.fields, field] })),
  removeField: (index) =>
    set((state) => ({ fields: state.fields.filter((_, i) => i !== index) })),
  updateField: (index, field) =>
    set((state) => ({
      fields: state.fields.map((f, i) => (i === index ? field : f)),
    })),
  setLoading: () => set({ status: "loading", error: null }),
  setSuccess: (result) => set({ status: "success", result, error: null }),
  setError: (error) => set({ status: "error", error }),
  reset: () =>
    set({ url: "", fields: [], status: "idle", result: null, error: null }),
}));

// History state (persisted to localStorage)
interface HistoryEntry {
  id: string;
  url: string;
  fields: Field[];
  result: ExtractionResult;
  timestamp: string;
}

interface HistoryState {
  entries: HistoryEntry[];
  addEntry: (url: string, fields: Field[], result: ExtractionResult) => void;
  removeEntry: (id: string) => void;
  clearHistory: () => void;
}

export const useHistoryStore = create<HistoryState>()(
  persist(
    (set) => ({
      entries: [],
      addEntry: (url, fields, result) =>
        set((state) => ({
          entries: [
            {
              id: crypto.randomUUID(),
              url,
              fields,
              result,
              timestamp: new Date().toISOString(),
            },
            ...state.entries,
          ].slice(0, 100), // Keep last 100
        })),
      removeEntry: (id) =>
        set((state) => ({
          entries: state.entries.filter((e) => e.id !== id),
        })),
      clearHistory: () => set({ entries: [] }),
    }),
    { name: "recap-history" }
  )
);

export type { HistoryEntry };
