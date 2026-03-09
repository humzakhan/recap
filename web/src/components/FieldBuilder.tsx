"use client";

import type { Field } from "@/lib/api";

const FIELD_TYPES = [
  { value: "text", label: "Text" },
  { value: "paragraph", label: "Paragraph" },
  { value: "list", label: "List" },
  { value: "number", label: "Number" },
  { value: "boolean", label: "Boolean" },
  { value: "date", label: "Date" },
  { value: "enum", label: "Enum" },
] as const;

interface FieldBuilderProps {
  fields: Field[];
  onFieldsChange: (fields: Field[]) => void;
}

export function FieldBuilder({ fields, onFieldsChange }: FieldBuilderProps) {
  const updateField = (index: number, updates: Partial<Field>) => {
    const updated = fields.map((f, i) =>
      i === index ? { ...f, ...updates } : f
    );
    onFieldsChange(updated);
  };

  const removeField = (index: number) => {
    onFieldsChange(fields.filter((_, i) => i !== index));
  };

  const addField = () => {
    onFieldsChange([...fields, { label: "", type: "text" }]);
  };

  return (
    <div className="space-y-2">
      <label className="block text-sm font-medium text-gray-700">
        Fields
      </label>

      {fields.map((field, index) => (
        <div key={index} className="flex items-start gap-2">
          <input
            type="text"
            value={field.label}
            onChange={(e) => updateField(index, { label: e.target.value })}
            placeholder="Field label"
            className="flex-1 rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
          <select
            value={field.type}
            onChange={(e) =>
              updateField(index, {
                type: e.target.value as Field["type"],
                options: e.target.value === "enum" ? field.options || [] : undefined,
              })
            }
            className="rounded-md border border-gray-300 px-2 py-1.5 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            {FIELD_TYPES.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
          {field.type === "enum" && (
            <input
              type="text"
              value={(field.options || []).join(", ")}
              onChange={(e) =>
                updateField(index, {
                  options: e.target.value
                    .split(",")
                    .map((s) => s.trim())
                    .filter(Boolean),
                })
              }
              placeholder="option1, option2, ..."
              className="flex-1 rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          )}
          <button
            onClick={() => removeField(index)}
            className="rounded-md px-2 py-1.5 text-gray-400 hover:bg-red-50 hover:text-red-500"
            title="Remove field"
          >
            &times;
          </button>
        </div>
      ))}

      <button
        onClick={addField}
        className="mt-1 rounded-md border border-dashed border-gray-300 px-3 py-1.5 text-sm text-gray-500 hover:border-gray-400 hover:text-gray-700"
      >
        + Add field
      </button>
    </div>
  );
}
