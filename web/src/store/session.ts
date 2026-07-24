// In-memory session only (never persisted to localStorage). The management key
// is CPA's full management credential; refreshing/closing the tab resets to
// login. The same-origin panel path (readPanelAuth) covers the iframe case.

import { readPanelAuth } from "./panelAuth";

export interface Session {
  baseUrl: string;
  secretKey: string;
}

let current: Session | null = null;
const listeners = new Set<() => void>();

function normalizeBase(url: string): string {
  let u = url.trim();
  if (u === "") return "";
  u = u.replace(/\/+$/, "");
  if (!/^https?:\/\//i.test(u)) u = "http://" + u;
  return u;
}

export function setSession(baseUrl: string, secretKey: string): Session {
  const session: Session = { baseUrl: normalizeBase(baseUrl), secretKey: secretKey.trim() };
  current = session;
  emit();
  return session;
}

export function clearSession(): void {
  current = null;
  emit();
}

export function getSession(): Session | null {
  return current;
}

export function isAuthed(): boolean {
  return current !== null && current.secretKey !== "" && current.baseUrl !== "";
}

export function subscribe(fn: () => void): () => void {
  listeners.add(fn);
  return () => {
    listeners.delete(fn);
  };
}

function emit(): void {
  for (const fn of listeners) fn();
}

// Restore a session from the official panel's saved key (same-origin iframe
// only, when the user checked "remember password" there).
export async function bootstrapFromPanel(): Promise<boolean> {
  const auth = readPanelAuth();
  if (!auth) return false;
  setSession(auth.apiBase, auth.managementKey);
  try {
    await verifySession(fetch);
    return true;
  } catch {
    clearSession();
    return false;
  }
}

// Confirm the key works by hitting the plugin's binds route.
export async function verifySession(fetchImpl: typeof fetch): Promise<Session> {
  const s = current;
  if (!s) throw new Error("no session");
  const res = await fetchImpl(s.baseUrl + "/v0/management/plugins/key-bind/binds", {
    headers: { Authorization: "Bearer " + s.secretKey },
  });
  if (!res.ok) {
    throw new Error("management key rejected (" + res.status + ")");
  }
  return s;
}
