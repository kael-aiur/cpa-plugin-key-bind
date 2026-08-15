import { apiClient, PLUGIN_BASE } from "./client";
import type { Binding, KeyBindPluginConfig } from "../types";

const CONFIG_PATH = `${PLUGIN_BASE}/config`;

export async function getPluginConfig(): Promise<KeyBindPluginConfig> {
  const { data } = await apiClient().get<KeyBindPluginConfig>(CONFIG_PATH);
  return data ?? {};
}

export async function patchBindings(bindings: Binding[]): Promise<void> {
  const safeBindings = bindings.map(({ key: _plaintextKey, ...binding }) => binding);
  await apiClient().patch(CONFIG_PATH, { bindings: safeBindings });
}
