import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Binding } from "../types";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  patch: vi.fn(),
}));

vi.mock("./client", () => ({
  PLUGIN_BASE: "/v0/management/plugins/key-bind",
  apiClient: () => ({ get: mocks.get, patch: mocks.patch }),
}));

import { getPluginConfig, patchBindings } from "./pluginConfig";

const binding: Binding = {
  id: "kb_0123456789abcdef01234567",
  name: "Team A",
  key_hash: "sha256:f3abf2a6cc4f00987743db5f544ba345b4899ae31f326d8ee9c4816de153c9e0",
  key_preview: "sk-test",
  allow: ["claude"],
  enabled: true,
};

describe("plugin config API", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.patch.mockReset();
  });

  it("gets the host-owned plugin config", async () => {
    mocks.get.mockResolvedValue({ data: { enabled: true, bindings: [binding] } });
    await expect(getPluginConfig()).resolves.toEqual({ enabled: true, bindings: [binding] });
    expect(mocks.get).toHaveBeenCalledWith("/v0/management/plugins/key-bind/config");
  });

  it("patches only the bindings field", async () => {
    mocks.patch.mockResolvedValue({ data: { status: "ok" } });
    await patchBindings([binding]);
    expect(mocks.patch).toHaveBeenCalledWith(
      "/v0/management/plugins/key-bind/config",
      { bindings: [binding] },
    );
  });

  it("does not send plaintext keys in bindings", async () => {
    mocks.patch.mockResolvedValue({ data: { status: "ok" } });
    const bindingWithPlaintext = { ...binding, key: "sk-live-secret", custom: "keep" };
    await patchBindings([bindingWithPlaintext]);
    expect(mocks.patch).toHaveBeenCalledWith(
      "/v0/management/plugins/key-bind/config",
      { bindings: [{ ...binding, custom: "keep" }] },
    );
  });
});
