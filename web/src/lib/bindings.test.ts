import { describe, expect, it } from "vitest";
import {
  buildBinding,
  deleteBindingRecord,
  hashKey,
  normalizeAllow,
  previewKey,
  toggleBindingRecord,
  updateBindingRecord,
  validateBindings,
} from "./bindings";

const existing = {
  id: "kb_0123456789abcdef01234567",
  name: "Team A",
  key_hash: "sha256:f3abf2a6cc4f00987743db5f544ba345b4899ae31f326d8ee9c4816de153c9e0",
  key_preview: "sk-test",
  allow: ["claude"],
  enabled: true,
  future_field: "preserve-me",
};

describe("binding crypto helpers", () => {
  it("matches the Go SHA-256 test vector", async () => {
    await expect(hashKey(" sk-test ")).resolves.toBe(
      "sha256:f3abf2a6cc4f00987743db5f544ba345b4899ae31f326d8ee9c4816de153c9e0",
    );
  });

  it("matches the Go preview behavior", () => {
    expect(previewKey("1234567890123")).toBe("1234567...90123");
  });
});

describe("binding validation", () => {
  it("normalizes allow and preserves unknown fields", () => {
    const [binding] = validateBindings([{ ...existing, allow: [" claude ", "", "claude"] }]);
    expect(binding.allow).toEqual(["claude"]);
    expect(binding.future_field).toBe("preserve-me");
  });

  it("rejects duplicate hashes", () => {
    expect(() => validateBindings([existing, { ...existing, id: "kb_1123456789abcdef01234567" }]))
      .toThrow(/duplicate key_hash/);
  });

  it("does not retain plaintext keys", () => {
    const [binding] = validateBindings([{ ...existing, key: "sk-secret" }]);
    expect(binding).not.toHaveProperty("key");
  });
});

describe("binding mutations", () => {
  it("builds a binding without retaining plaintext", async () => {
    const binding = await buildBinding({
      id: "kb_2123456789abcdef01234567",
      name: " Team B ",
      key: "sk-test",
      allow: [" codex ", "codex"],
      enabled: true,
    });
    expect(binding).not.toHaveProperty("key");
    expect(binding.name).toBe("Team B");
    expect(binding.allow).toEqual(["codex"]);
  });

  it("updates by id and preserves unknown fields", () => {
    const updated = updateBindingRecord([existing], existing.id, { name: "Renamed" });
    expect(updated[0].name).toBe("Renamed");
    expect(updated[0].future_field).toBe("preserve-me");
  });

  it("toggles and deletes by id", () => {
    expect(toggleBindingRecord([existing], existing.id)[0].enabled).toBe(false);
    expect(deleteBindingRecord([existing], existing.id)).toEqual([]);
  });
});

describe("normalizeAllow", () => {
  it("trims, removes empty values, and keeps first occurrence order", () => {
    expect(normalizeAllow([" codex ", "", "claude", "codex"])).toEqual(["codex", "claude"]);
  });
});
