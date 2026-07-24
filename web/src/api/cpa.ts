import { apiClient, CPA_BASE } from "./client";
import type { ProviderOption } from "../types";

// --- platform api-keys ---

// GET /v0/management/api-keys may return string[] or an array of objects.
export async function listApiKeys(): Promise<string[]> {
  const { data } = await apiClient().get<unknown>(CPA_BASE + "/api-keys");
  return normalizeApiKeys(data);
}

function normalizeApiKeys(input: unknown): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  const push = (v: string) => {
    const t = v.trim();
    if (t && !seen.has(t)) {
      seen.add(t);
      out.push(t);
    }
  };
  if (Array.isArray(input)) {
    for (const item of input) {
      if (typeof item === "string") {
        push(item);
      } else if (item && typeof item === "object") {
        const r = item as Record<string, unknown>;
        push(String(r["api-key"] ?? r["apiKey"] ?? r["key"] ?? r["Key"] ?? ""));
      }
    }
  }
  return out;
}

// --- provider/account options for the multi-select ---

export async function listProviderOptions(): Promise<ProviderOption[]> {
  const [authFiles, providers] = await Promise.all([listAuthFiles(), listAIProviders()]);
  // De-dup by value (an auth id and a provider name never collide due to the
  // "auth:" prefix, but providers may repeat).
  const seen = new Set<string>();
  const out: ProviderOption[] = [];
  for (const o of [...authFiles, ...providers]) {
    if (o && !seen.has(o.value)) {
      seen.add(o.value);
      out.push(o);
    }
  }
  return out;
}

async function listAuthFiles(): Promise<ProviderOption[]> {
  try {
    const { data } = await apiClient().get<unknown>(CPA_BASE + "/auth-files");
    const files = extractArray(data, "auth_files");
    return files
      .map((f) => {
        const r = (f ?? {}) as Record<string, unknown>;
        const id = String(r["id"] ?? r["name"] ?? r["filename"] ?? "");
        const provider = String(r["provider"] ?? r["type"] ?? "").toLowerCase();
        const label = String(r["name"] ?? r["filename"] ?? r["id"] ?? id);
        const value = id ? `auth:${id}` : "";
        if (!value) return null;
        return { value, label, kind: "auth" as const, meta: provider || undefined };
      })
      .filter((o) => o !== null) as ProviderOption[];
  } catch {
    return [];
  }
}

// Each AI provider's API-key config endpoint. If it returns a non-empty list,
// expose that provider as one selectable option (covers all its accounts).
const AI_PROVIDER_ENDPOINTS: Array<[string, string]> = [
  ["claude", "/claude-api-key"],
  ["codex", "/codex-api-key"],
  ["gemini", "/gemini-api-key"],
  ["xai", "/xai-api-key"],
  ["vertex", "/vertex-api-key"],
  ["interactions", "/interactions-api-key"],
];

async function listAIProviders(): Promise<ProviderOption[]> {
  const settled = await Promise.allSettled(
    AI_PROVIDER_ENDPOINTS.map(async ([provider, suffix]) => {
      const { data } = await apiClient().get<unknown>(CPA_BASE + suffix);
      const count = extractArray(data).length;
      if (count === 0) return null;
      return {
        value: provider,
        label: provider,
        kind: "provider" as const,
        meta: `${count} key(s)`,
      };
    }),
  );
  return settled
    .map((r) => (r.status === "fulfilled" ? r.value : null))
    .filter((o) => o !== null) as ProviderOption[];
}

function extractArray(data: unknown, key?: string): unknown[] {
  if (Array.isArray(data)) return data;
  if (data && typeof data === "object") {
    const r = data as Record<string, unknown>;
    if (key && Array.isArray(r[key])) return r[key] as unknown[];
    if (Array.isArray(r["keys"])) return r["keys"] as unknown[];
  }
  return [];
}
