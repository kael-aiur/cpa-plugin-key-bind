import { apiClient, CPA_BASE } from "./client";
import type { ProviderOption } from "../types";

// GET /v0/management/api-keys -> { "api-keys": string[] }
export async function listApiKeys(): Promise<string[]> {
  const { data } = await apiClient().get<Record<string, unknown>>(CPA_BASE + "/api-keys");
  const keys = data["api-keys"] ?? data.apiKeys;
  return Array.isArray(keys) ? keys.map((k) => String(k)).filter(Boolean) : [];
}

export async function listProviderOptions(): Promise<ProviderOption[]> {
  const [providers, authFiles] = await Promise.all([
    listAIProviders().catch(() => [] as ProviderOption[]),
    listAuthFiles().catch(() => [] as ProviderOption[]),
  ]);
  const seen = new Set<string>();
  const out: ProviderOption[] = [];
  for (const o of [...providers, ...authFiles]) {
    if (o && !seen.has(o.value)) {
      seen.add(o.value);
      out.push(o);
    }
  }
  return out;
}

// GET /v0/management/auth-files -> { files: [{ name, provider?, type? }] }
async function listAuthFiles(): Promise<ProviderOption[]> {
  const { data } = await apiClient().get<Record<string, unknown>>(CPA_BASE + "/auth-files");
  const files = Array.isArray(data.files) ? (data.files as unknown[]) : [];
  const mapped = files.map((f) => {
    const r = (f ?? {}) as Record<string, unknown>;
    const name = String(r.name ?? "");
    const provider = String(r.provider ?? r.type ?? "").toLowerCase();
    if (!name) return null;
    return {
      value: `auth:${name}`,
      label: name,
      kind: "auth" as const,
      meta: provider || undefined,
    };
  });
  return mapped.filter((o) => o !== null) as ProviderOption[];
}

// Each provider config endpoint returns { "<endpoint>": [...] }, e.g.
// /codex-api-key -> { "codex-api-key": [...] }.
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
      const { data } = await apiClient().get<Record<string, unknown>>(CPA_BASE + suffix);
      const key = suffix.replace(/^\//, "");
      const arr = data[key] ?? data[provider];
      const count = Array.isArray(arr) ? arr.length : 0;
      if (count === 0) return null;
      return { value: provider, label: provider, kind: "provider" as const, meta: `${count} key(s)` };
    }),
  );
  const mapped = settled.map((r) => (r.status === "fulfilled" ? r.value : null));
  return mapped.filter((o) => o !== null) as ProviderOption[];
}
