// Reuse the official CPA management panel's saved management key when this web
// UI is loaded as a same-origin iframe inside that panel.
//
// The panel stores its auth state (apiBase +, only when "remember password" is
// checked, managementKey) in localStorage under key "cli-proxy-auth", run
// through a reversible XOR+base64 obfuscation (mirrors the panel's encryption).
// Because the panel and this plugin resource are same-origin (both served by
// CPA), they share the SAME localStorage and the SAME obfuscation key
// (host + userAgent are identical), so we decode the panel's stored blob here
// and skip a second login.

const ENC_PREFIX = "enc::v1::";
const SECRET_SALT = "cli-proxy-api-webui::secure-storage";
const STORAGE_KEY = "cli-proxy-auth";

export interface PanelAuth {
  apiBase: string;
  managementKey: string;
}

let cachedKeyBytes: Uint8Array | null = null;

function encodeText(text: string): Uint8Array {
  return new TextEncoder().encode(text);
}

function decodeText(bytes: Uint8Array): string {
  return new TextDecoder().decode(bytes);
}

function getKeyBytes(): Uint8Array {
  if (cachedKeyBytes) return cachedKeyBytes;
  try {
    cachedKeyBytes = encodeText(`${SECRET_SALT}|${window.location.host}|${navigator.userAgent}`);
  } catch {
    cachedKeyBytes = encodeText(SECRET_SALT);
  }
  return cachedKeyBytes;
}

function xorBytes(data: Uint8Array, keyBytes: Uint8Array): Uint8Array {
  const result = new Uint8Array(data.length);
  for (let i = 0; i < data.length; i++) {
    result[i] = data[i] ^ keyBytes[i % keyBytes.length];
  }
  return result;
}

function fromBase64(base64: string): Uint8Array {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes;
}

export function isEmbedded(): boolean {
  try {
    return window.self !== window.top;
  } catch {
    return false;
  }
}

// The persisted zustand envelope: { state: {...}, version: number }.
function extractAuth(parsed: unknown): PanelAuth | null {
  if (!parsed || typeof parsed !== "object") return null;
  const root = parsed as Record<string, unknown>;
  const state =
    (root.state as Record<string, unknown> | undefined) ?? (root as Record<string, unknown>);
  const apiBase = typeof state.apiBase === "string" ? state.apiBase : "";
  const managementKey = typeof state.managementKey === "string" ? state.managementKey : "";
  if (!apiBase || !managementKey) return null;
  return { apiBase, managementKey };
}

// Read the panel's saved auth from shared same-origin localStorage.
export function readPanelAuth(): PanelAuth | null {
  if (!isEmbedded()) return null;
  let raw: string | null;
  try {
    raw = localStorage.getItem(STORAGE_KEY);
  } catch {
    return null;
  }
  if (!raw || !raw.startsWith(ENC_PREFIX)) return null;
  let decoded: string;
  try {
    const encrypted = fromBase64(raw.slice(ENC_PREFIX.length));
    decoded = decodeText(xorBytes(encrypted, getKeyBytes()));
  } catch {
    return null;
  }
  try {
    return extractAuth(JSON.parse(decoded));
  } catch {
    return null;
  }
}
