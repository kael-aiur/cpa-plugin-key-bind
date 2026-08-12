import type { Binding } from "../types";
import {
  buildBinding,
  deleteBindingRecord,
  toggleBindingRecord,
  updateBindingRecord,
  validateBindings,
} from "../lib/bindings";
import { getPluginConfig, patchBindings } from "./pluginConfig";

export interface BindingInput {
  id?: string;
  name?: string;
  key?: string;
  allow?: string[];
  enabled?: boolean;
}

async function latestBindings(): Promise<Binding[]> {
  const config = await getPluginConfig();
  return validateBindings(config.bindings ?? []);
}

async function mutateBindings(
  mutation: (latest: Binding[]) => Binding[] | Promise<Binding[]>,
): Promise<Binding[]> {
  const latest = await latestBindings();
  const updated = validateBindings(await mutation(latest));
  await patchBindings(updated);
  return latestBindings();
}

export async function listBindings(): Promise<Binding[]> {
  return latestBindings();
}

export async function createBinding(input: {
  name: string;
  key: string;
  allow: string[];
  enabled: boolean;
}): Promise<Binding> {
  let created: Binding | undefined;
  await mutateBindings(async (latest) => {
    const next = await buildBinding(input);
    if (latest.some((binding) => binding.key_hash === next.key_hash)) {
      throw new Error("该 API Key 已存在绑定");
    }
    created = next;
    return [...latest, next];
  });
  if (!created) throw new Error("创建绑定失败");
  return created;
}

export async function updateBinding(input: BindingInput): Promise<Binding> {
  if (!input.id) throw new Error("id is required");
  const id = input.id;
  let updatedBinding: Binding | undefined;
  await mutateBindings((latest) => {
    const updated = updateBindingRecord(latest, id, {
      ...(input.name === undefined ? {} : { name: input.name }),
      ...(input.allow === undefined ? {} : { allow: input.allow }),
      ...(input.enabled === undefined ? {} : { enabled: input.enabled }),
    });
    const next = updated.find((binding) => binding.id === id);
    if (!next) throw new Error(`绑定不存在：${id}`);
    updatedBinding = next;
    return updated;
  });
  if (!updatedBinding) throw new Error(`绑定不存在：${id}`);
  return updatedBinding;
}

export async function deleteBinding(id: string): Promise<void> {
  await mutateBindings((latest) => deleteBindingRecord(latest, id));
}

export async function toggleBinding(id: string): Promise<void> {
  await mutateBindings((latest) => toggleBindingRecord(latest, id));
}
