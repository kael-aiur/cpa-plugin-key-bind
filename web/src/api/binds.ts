import { apiClient, PLUGIN_BASE } from "./client";
import type { Binding } from "../types";

export async function listBindings(): Promise<Binding[]> {
  const { data } = await apiClient().get<{ bindings: Binding[] }>(PLUGIN_BASE + "/binds");
  return data.bindings ?? [];
}

export interface BindingInput {
  id?: string;
  name?: string;
  key?: string;
  allow?: string[];
  enabled?: boolean;
}

export async function createBinding(input: {
  name: string;
  key: string;
  allow: string[];
  enabled: boolean;
}): Promise<Binding> {
  const { data } = await apiClient().post<Binding>(PLUGIN_BASE + "/binds", input);
  return data;
}

export async function updateBinding(input: BindingInput): Promise<Binding> {
  const { data } = await apiClient().put<Binding>(PLUGIN_BASE + "/binds", input);
  return data;
}

export async function deleteBinding(id: string): Promise<void> {
  await apiClient().delete(PLUGIN_BASE + "/binds", { params: { id } });
}
