import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { setSession, verifySession } from "../store/session";

// Fallback login (cross-origin or "remember password" not checked in the panel).
// In the normal same-origin iframe case, bootstrapFromPanel already logged in.
export default function Login() {
  const nav = useNavigate();
  // Default to the current origin: when hosted by CPA, the API is same-origin.
  const [baseUrl, setBaseUrl] = useState(
    typeof window !== "undefined" ? window.location.origin : "http://127.0.0.1:8317",
  );
  const [secretKey, setSecretKey] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (!secretKey.trim()) {
      setError("请输入管理密钥");
      return;
    }
    setBusy(true);
    try {
      setSession(baseUrl, secretKey);
      await verifySession(fetch);
      nav("/bindings");
    } catch (err) {
      setError((err as Error).message || "登录失败");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="login-page">
      <div className="lp-title">Key Bind</div>
      <div className="lp-sub">连接到 CLIProxyAPI 管理接口</div>
      <form className="card" onSubmit={submit}>
        <label className="field">
          <span className="lbl">服务地址</span>
          <input
            className="input"
            value={baseUrl}
            onChange={(e) => setBaseUrl(e.target.value)}
            autoFocus
          />
        </label>
        <label className="field">
          <span className="lbl">管理密钥 (Management Key)</span>
          <input
            className="input"
            type="password"
            value={secretKey}
            onChange={(e) => setSecretKey(e.target.value)}
          />
        </label>
        {error && <div className="error">{error}</div>}
        <button className="btn primary" type="submit" disabled={busy}>
          {busy ? "验证中…" : "连接"}
        </button>
        <div className="lp-note">
          密钥仅保存在内存,关闭页面即清除。作为面板内嵌 iframe(同源)时可自动免登录。
        </div>
      </form>
    </div>
  );
}
