const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export interface Field {
  label: string;
  type: "text" | "list" | "enum" | "number" | "boolean" | "date" | "paragraph";
  options?: string[];
}

export interface Template {
  name: string;
  description?: string;
  fields: Field[];
}

export interface ExtractionRequest {
  url: string;
  fields?: Field[];
  template?: string;
  format?: "json" | "markdown" | "csv";
}

export interface ExtractionResult {
  url: string;
  content_type: string;
  extracted_at: string;
  data: Record<string, unknown>;
  raw_content: string;
  model: string;
  tokens_used: number;
  formatted: string;
}

export interface ApiError {
  error: string;
  code: string;
}

export async function extract(req: ExtractionRequest): Promise<ExtractionResult> {
  const res = await fetch(`${API_BASE}/api/v1/extract`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const err: ApiError = await res.json();
    throw new Error(err.error || `HTTP ${res.status}`);
  }
  return res.json();
}

export async function getTemplates(): Promise<Template[]> {
  const res = await fetch(`${API_BASE}/api/v1/templates`);
  if (!res.ok) throw new Error(`Failed to load templates: HTTP ${res.status}`);
  return res.json();
}

export async function getTemplate(name: string): Promise<Template> {
  const res = await fetch(`${API_BASE}/api/v1/templates/${encodeURIComponent(name)}`);
  if (!res.ok) throw new Error(`Template not found: ${name}`);
  return res.json();
}

export async function createTemplate(tmpl: Template): Promise<Template> {
  const res = await fetch(`${API_BASE}/api/v1/templates`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(tmpl),
  });
  if (!res.ok) {
    const err: ApiError = await res.json();
    throw new Error(err.error);
  }
  return res.json();
}

export async function deleteTemplate(name: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/v1/templates/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });
  if (!res.ok) throw new Error(`Failed to delete template: ${name}`);
}

export interface ModelInfo {
  id: string;
  name: string;
  provider: string;
}

export interface ModelsResponse {
  models: ModelInfo[];
  current: string;
}

export async function getModels(): Promise<ModelsResponse> {
  const res = await fetch(`${API_BASE}/api/v1/models`);
  if (!res.ok) throw new Error(`Failed to load models: HTTP ${res.status}`);
  return res.json();
}
