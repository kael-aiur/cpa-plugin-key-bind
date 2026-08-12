import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Binding } from "../types";

const mocks = vi.hoisted(() => ({
  getPluginConfig: vi.fn(),
  patchBindings: vi.fn(),
}));

vi.mock("./pluginConfig", () => ({
  getPluginConfig: mocks.getPluginConfig,
  patchBindings: mocks.patchBindings,
}));

import { createBinding, updateBinding } from "./binds";

const currentBinding: Binding = {
  id: "kb_0123456789abcdef01234567",
  name: "Team A",
  key_hash: "sha256:f3abf2a6cc4f00987743db5f544ba345b4899ae31f326d8ee9c4816de153c9e0",
  key_preview: "sk-test",
  allow: ["claude"],
  enabled: true,
};

describe("binding CRUD facade persistence", () => {
  beforeEach(() => {
    mocks.getPluginConfig.mockReset();
    mocks.patchBindings.mockReset();
    mocks.patchBindings.mockResolvedValue(undefined);
  });

  it("returns the host record from the GET after create PATCH", async () => {
    mocks.getPluginConfig
      .mockResolvedValueOnce({ bindings: [] })
      .mockImplementationOnce(async () => {
        const patched = mocks.patchBindings.mock.calls[0][0][0] as Binding;
        return { bindings: [{ ...patched, name: "Host-created", custom: "host" }] };
      });

    const created = await createBinding({
      name: "Local name",
      key: "sk-create-secret",
      allow: ["claude"],
      enabled: true,
    });

    expect(created.name).toBe("Host-created");
    expect(created.custom).toBe("host");
    expect(mocks.getPluginConfig).toHaveBeenCalledTimes(2);
    expect(mocks.patchBindings).toHaveBeenCalledTimes(1);
    expect(mocks.getPluginConfig.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.patchBindings.mock.invocationCallOrder[0],
    );
    expect(mocks.patchBindings.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.getPluginConfig.mock.invocationCallOrder[1],
    );
    expect(mocks.patchBindings.mock.calls[0][0][0]).not.toHaveProperty("key");
  });

  it("returns the host record from the GET after update PATCH", async () => {
    mocks.getPluginConfig
      .mockResolvedValueOnce({ bindings: [currentBinding] })
      .mockImplementationOnce(async () => {
        const patched = mocks.patchBindings.mock.calls[0][0][0] as Binding;
        return { bindings: [{ ...patched, name: "Host-updated", custom: "host" }] };
      });

    const updated = await updateBinding({ id: currentBinding.id, name: "Local name" });

    expect(updated.name).toBe("Host-updated");
    expect(updated.custom).toBe("host");
    expect(mocks.getPluginConfig).toHaveBeenCalledTimes(2);
    expect(mocks.patchBindings).toHaveBeenCalledTimes(1);
    expect(mocks.getPluginConfig.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.patchBindings.mock.invocationCallOrder[0],
    );
    expect(mocks.patchBindings.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.getPluginConfig.mock.invocationCallOrder[1],
    );
    expect(mocks.patchBindings.mock.calls[0][0][0]).not.toHaveProperty("key");
  });
});
