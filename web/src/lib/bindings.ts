import type { Binding } from "../types";

const ID_PATTERN = /^kb_[a-f0-9]{24}$/;
const HASH_PATTERN = /^sha256:[a-f0-9]{64}$/;
const GO_SPACE_CODE_POINTS = new Set([
  0x0009,
  0x000a,
  0x000b,
  0x000c,
  0x000d,
  0x0020,
  0x0085,
  0x00a0,
  0x1680,
  0x2028,
  0x2029,
  0x202f,
  0x205f,
  0x3000,
]);

function isGoSpace(codePoint: number): boolean {
  return GO_SPACE_CODE_POINTS.has(codePoint) || (codePoint >= 0x2000 && codePoint <= 0x200a);
}

function trimGoSpace(input: string): string {
  let start = 0;
  let end = input.length;
  while (start < end) {
    const codePoint = input.codePointAt(start);
    if (codePoint === undefined || !isGoSpace(codePoint)) break;
    start += codePoint > 0xffff ? 2 : 1;
  }
  while (end > start) {
    const codePoint = input.codePointAt(end - 1);
    if (codePoint === undefined || !isGoSpace(codePoint)) break;
    end -= codePoint > 0xffff ? 2 : 1;
  }
  return input.slice(start, end);
}

export interface BuildBindingInput {
  id?: string;
  name: string;
  key: string;
  allow: string[];
  enabled: boolean;
}

export type BindingChanges = Partial<Pick<Binding, "name" | "allow" | "enabled">>;

export async function hashKey(key: string): Promise<string> {
  const bytes = new TextEncoder().encode(trimGoSpace(key));
  const digest = await crypto.subtle.digest("SHA-256", bytes);
  const hex = Array.from(new Uint8Array(digest), (value) =>
    value.toString(16).padStart(2, "0"),
  ).join("");
  return `sha256:${hex}`;
}

export function previewKey(key: string): string {
  const normalized = trimGoSpace(key);
  if (normalized.length <= 12) return normalized;
  return `${normalized.slice(0, 7)}...${normalized.slice(-5)}`;
}

export function newBindingID(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(12));
  return `kb_${Array.from(bytes, (value) => value.toString(16).padStart(2, "0")).join("")}`;
}

export function normalizeAllow(input: string[]): string[] {
  const seen = new Set<string>();
  const output: string[] = [];
  for (const raw of input) {
    const value = raw.trim();
    if (!value || seen.has(value)) continue;
    seen.add(value);
    output.push(value);
  }
  return output;
}

export async function buildBinding(input: BuildBindingInput): Promise<Binding> {
  const key = trimGoSpace(input.key);
  if (!key) throw new Error("请选择 API Key");
  return {
    id: input.id ?? newBindingID(),
    name: input.name.trim(),
    key_hash: await hashKey(key),
    key_preview: previewKey(key),
    allow: normalizeAllow(input.allow),
    enabled: input.enabled,
  };
}

export function validateBindings(input: unknown): Binding[] {
  if (!Array.isArray(input)) throw new Error("bindings 必须是数组");
  const ids = new Set<string>();
  const hashes = new Set<string>();
  return input.map((raw, index) => {
    if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
      throw new Error(`bindings[${index}] 必须是对象`);
    }
    const source = raw as Record<string, unknown>;
    const id = String(source.id ?? "").trim();
    const keyHash = String(source.key_hash ?? "").trim();
    const keyPreview = String(source.key_preview ?? "").trim();
    if (!ID_PATTERN.test(id)) throw new Error(`bindings[${index}].id 格式无效`);
    if (!HASH_PATTERN.test(keyHash)) throw new Error(`bindings[${index}].key_hash 格式无效`);
    if (!keyPreview || keyPreview.length > 128) throw new Error(`bindings[${index}].key_preview 格式无效`);
    if (ids.has(id)) throw new Error(`bindings[${index}] duplicate id`);
    if (hashes.has(keyHash)) throw new Error(`bindings[${index}] duplicate key_hash`);
    const allowValues = source.allow === undefined ? [] : source.allow;
    if (!Array.isArray(allowValues)) {
      throw new Error(`bindings[${index}].allow 必须是数组`);
    }
    const enabledValue = source.enabled === undefined ? true : source.enabled;
    if (typeof enabledValue !== "boolean") {
      throw new Error(`bindings[${index}].enabled 必须是布尔值`);
    }
    ids.add(id);
    hashes.add(keyHash);
    const { key: _plaintextKey, ...sourceWithoutKey } = source;
    return {
      ...sourceWithoutKey,
      id,
      name: String(source.name ?? "").trim(),
      key_hash: keyHash,
      key_preview: keyPreview,
      allow: normalizeAllow(allowValues.map((value) => String(value))),
      enabled: enabledValue,
    } as Binding;
  });
}

export function updateBindingRecord(bindings: Binding[], id: string, changes: BindingChanges): Binding[] {
  let found = false;
  const updated = bindings.map((binding) => {
    if (binding.id !== id) return binding;
    found = true;
    return {
      ...binding,
      ...changes,
      name: changes.name === undefined ? binding.name : changes.name.trim(),
      allow: changes.allow === undefined ? binding.allow : normalizeAllow(changes.allow),
    };
  });
  if (!found) throw new Error(`绑定不存在：${id}`);
  return validateBindings(updated);
}

export function deleteBindingRecord(bindings: Binding[], id: string): Binding[] {
  const updated = bindings.filter((binding) => binding.id !== id);
  if (updated.length === bindings.length) throw new Error(`绑定不存在：${id}`);
  return validateBindings(updated);
}

export function toggleBindingRecord(bindings: Binding[], id: string): Binding[] {
  const binding = bindings.find((item) => item.id === id);
  if (!binding) throw new Error(`绑定不存在：${id}`);
  return updateBindingRecord(bindings, id, { enabled: !binding.enabled });
}
