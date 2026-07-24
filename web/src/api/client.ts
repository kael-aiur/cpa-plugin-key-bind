import axios, { AxiosInstance } from "axios";
import { getSession, clearSession } from "../store/session";

// Build a fresh client per request using the current in-memory session.
export function apiClient(): AxiosInstance {
  const s = getSession();
  if (!s || !s.baseUrl) {
    throw new Error("not authenticated");
  }
  const instance = axios.create({
    baseURL: s.baseUrl,
    headers: {
      Authorization: "Bearer " + s.secretKey,
      "Content-Type": "application/json",
    },
    validateStatus: (code) => code >= 200 && code < 300,
  });

  instance.interceptors.response.use(
    (r) => r,
    (err) => {
      if (err?.response?.status === 401 || err?.response?.status === 403) {
        clearSession();
      }
      return Promise.reject(err);
    },
  );

  return instance;
}

export const PLUGIN_BASE = "/v0/management/plugins/key-bind";
export const CPA_BASE = "/v0/management";
