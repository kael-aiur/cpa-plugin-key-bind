import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("./panelAuth", () => ({
  readPanelAuth: vi.fn(() => null),
}));

import { clearSession, setSession, verifySession } from "./session";

describe("session verification", () => {
  afterEach(() => {
    clearSession();
  });

  it("probes the host plugin config endpoint instead of the removed binds route", async () => {
    setSession("https://cpa.example.test", "management-secret");
    const fetchImpl = vi.fn(async (..._args: Parameters<typeof fetch>) =>
      new Response(null, { status: 200 }),
    );

    await verifySession(fetchImpl);

    expect(fetchImpl).toHaveBeenCalledWith(
      "https://cpa.example.test/v0/management/plugins/key-bind/config",
      { headers: { Authorization: "Bearer management-secret" } },
    );
    expect(fetchImpl.mock.calls[0][0]).not.toContain("/binds");
  });
});
