import { useCallback, useEffect, useState } from "react";
import {
  listBindings,
  createBinding,
  updateBinding,
  deleteBinding,
  toggleBinding,
  type BindingInput,
} from "../api/binds";
import { listApiKeys, listProviderOptions } from "../api/cpa";
import type { Binding, ProviderOption } from "../types";

type EditTarget = { mode: "new" } | { mode: "edit"; binding: Binding } | null;

export default function Bindings() {
  const [bindings, setBindings] = useState<Binding[]>([]);
  const [apiKeys, setApiKeys] = useState<string[]>([]);
  const [options, setOptions] = useState<ProviderOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [configReady, setConfigReady] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [target, setTarget] = useState<EditTarget>(null);

  const reload = useCallback(async () => {
    setConfigReady(false);
    setLoading(true);
    setError("");
    try {
      const [bs, keys, opts] = await Promise.all([
        listBindings(),
        listApiKeys().catch(() => [] as string[]),
        listProviderOptions().catch(() => [] as ProviderOption[]),
      ]);
      setBindings(bs);
      setApiKeys(keys);
      setOptions(opts);
      setConfigReady(true);
      return true;
    } catch (e) {
      setConfigReady(false);
      setError((e as Error).message || "加载插件配置失败");
      return false;
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    reload();
  }, [reload]);

  if (loading) {
    return (
      <div className="app">
        <p className="muted">加载中…</p>
      </div>
    );
  }

  return (
    <div className="app">
      {!target && (
        <div className="header">
          <div>
            <h1>Key Bind · 密钥供应商绑定</h1>
            <div className="sub">
              为每个 API Key 指定可用的供应商/账号;未绑定的 Key 按平台原策略放行
            </div>
          </div>
          <button
            className="btn primary"
            onClick={() => {
              setNotice("");
              setTarget({ mode: "new" });
            }}
            disabled={!configReady}
          >
            + 新建绑定
          </button>
        </div>
      )}

      {error && <div className="card"><div className="error">{error}</div></div>}
      {notice && <div className="card"><div className="success">{notice}</div></div>}

      {target ? (
        <BindingForm
          target={target}
          apiKeys={apiKeys}
          options={options}
          onCancel={() => setTarget(null)}
          onSaved={async () => {
            setTarget(null);
            if (await reload()) {
              setNotice("配置已保存，宿主正在应用。");
            }
          }}
        />
      ) : configReady ? (
        <BindingList
          bindings={bindings}
          onEdit={(b) => {
            setNotice("");
            setTarget({ mode: "edit", binding: b });
          }}
          onDelete={async (b) => {
            if (!window.confirm(`删除绑定「${b.name || b.key_preview}」?`)) return;
            setNotice("");
            try {
              await deleteBinding(b.id);
              if (await reload()) {
                setNotice("配置已保存，宿主正在应用。");
              }
            } catch (e) {
              setError((e as Error).message);
            }
          }}
          onToggle={async (b) => {
            setNotice("");
            try {
              await toggleBinding(b.id);
              if (await reload()) {
                setNotice("配置已保存，宿主正在应用。");
              }
            } catch (e) {
              setError((e as Error).message);
            }
          }}
        />
      ) : (
        !loading && (
          <div className="card empty">当前插件配置无法读取，请先修复 Management API 连接或配置内容。</div>
        )
      )}

    </div>
  );
}

function BindingList(props: {
  bindings: Binding[];
  onEdit: (b: Binding) => void;
  onDelete: (b: Binding) => void;
  onToggle: (b: Binding) => void;
}) {
  const { bindings, onEdit, onDelete, onToggle } = props;
  if (bindings.length === 0) {
    return <div className="card empty">还没有绑定记录。点击「新建绑定」创建。</div>;
  }
  return (
    <div className="card" style={{ padding: 0, overflow: "hidden" }}>
      <table>
        <thead>
          <tr>
            <th>名称</th>
            <th>Key</th>
            <th>允许的供应商</th>
            <th>状态</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {bindings.map((b) => (
            <tr key={b.id}>
              <td>{b.name || "—"}</td>
              <td>
                <code>{b.key_preview}</code>
              </td>
              <td>
                {b.allow.length === 0 ? (
                  <span className="muted">空(将禁用全部)</span>
                ) : (
                  b.allow.map((a) => {
                    const isAuth = a.startsWith("auth:");
                    return (
                      <span key={a} className={`tag ${isAuth ? "auth" : "group"}`}>
                        {isAuth ? a.slice(5) : a}
                      </span>
                    );
                  })
                )}
              </td>
              <td>
                <span
                  className={`badge ${b.enabled ? "on" : "off"}`}
                  style={{ cursor: "pointer" }}
                  onClick={() => onToggle(b)}
                  title="点击切换"
                >
                  {b.enabled ? "启用" : "停用"}
                </span>
              </td>
              <td style={{ whiteSpace: "nowrap" }}>
                <button className="btn sm" onClick={() => onEdit(b)}>
                  编辑
                </button>{" "}
                <button className="btn sm danger" onClick={() => onDelete(b)}>
                  删除
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function BindingForm(props: {
  target: Exclude<EditTarget, null>;
  apiKeys: string[];
  options: ProviderOption[];
  onCancel: () => void;
  onSaved: () => void | Promise<void>;
}) {
  const { target, apiKeys, options, onCancel, onSaved } = props;
  const editing = target.mode === "edit" ? target.binding : null;
  const [name, setName] = useState(editing?.name ?? "");
  const [key, setKey] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set(editing?.allow ?? []));
  const [enabled, setEnabled] = useState(editing?.enabled ?? true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const toggle = (v: string) =>
    setSelected((s) => {
      const n = new Set(s);
      if (n.has(v)) n.delete(v);
      else n.add(v);
      return n;
    });

  const providerOpts = options.filter((o) => o.kind === "provider");
  const authOpts = options.filter((o) => o.kind === "auth");

  const save = async () => {
    setError("");
    if (target.mode === "new" && !key.trim()) {
      setError("请选择 API Key");
      return;
    }
    const allow = Array.from(selected);
    if (
      allow.length === 0 &&
      !window.confirm("未选择任何供应商,该 Key 将无法使用任何账号。确定保存?")
    ) {
      return;
    }
    setBusy(true);
    try {
      if (target.mode === "new") {
        await createBinding({ name, key: key.trim(), allow, enabled });
      } else {
        const b = target.binding;
        const input: BindingInput = { id: b.id, allow };
        if (name !== b.name) input.name = name;
        if (enabled !== b.enabled) input.enabled = enabled;
        await updateBinding(input);
      }
      await onSaved();
    } catch (e) {
      setError((e as Error).message || "保存失败");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="card">
      <h2 style={{ marginTop: 0 }}>{target.mode === "new" ? "新建绑定" : "编辑绑定"}</h2>
      {editing && (
        <div className="muted" style={{ marginBottom: 12 }}>
          ID: {editing.id} · Key: {editing.key_preview}
        </div>
      )}

      <label className="field">
        <span className="lbl">名称(备注)</span>
        <input
          className="input"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="如:团队A"
        />
      </label>

      <label className="field">
        <span className="lbl">
          API Key{target.mode === "edit" ? "(编辑时不可改)" : ""}
        </span>
        <select value={key} onChange={(e) => setKey(e.target.value)} disabled={target.mode === "edit"}>
          <option value="">
            {target.mode === "edit" ? editing?.key_preview ?? "(不变)" : "— 选择 Key —"}
          </option>
          {apiKeys.map((k) => (
            <option key={k} value={k}>
              {k}
            </option>
          ))}
        </select>
      </label>

      <label className="field">
        <span className="lbl">允许使用的供应商 / 账号(多选)</span>
        <div className="checkbox-group">
          {providerOpts.length > 0 && (
            <div className="cb-section">
              <div className="cb-section-title">AI 供应商(该类型全部账号)</div>
              {providerOpts.map((o) => (
                <label key={o.value} className="cb-item">
                  <input
                    type="checkbox"
                    checked={selected.has(o.value)}
                    onChange={() => toggle(o.value)}
                  />
                  <span className="tag">{o.label}</span>
                  <span className="meta">{o.meta}</span>
                </label>
              ))}
            </div>
          )}
          {authOpts.length > 0 && (
            <div className="cb-section">
              <div className="cb-section-title">认证文件(精确到账号)</div>
              {authOpts.map((o) => (
                <label key={o.value} className="cb-item">
                  <input
                    type="checkbox"
                    checked={selected.has(o.value)}
                    onChange={() => toggle(o.value)}
                  />
                  <span>{o.label}</span>
                  {o.meta && <span className="tag group">{o.meta}</span>}
                </label>
              ))}
            </div>
          )}
          {options.length === 0 && (
            <div className="muted">未获取到供应商列表(请确认后端已配置账号/凭证)</div>
          )}
        </div>
      </label>

      <label className="cb-item" style={{ marginLeft: -4, marginTop: 4 }}>
        <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
        <span>启用此绑定</span>
      </label>

      {error && <div className="error">{error}</div>}

      <div className="form-actions">
        <button className="btn" onClick={onCancel} disabled={busy}>
          取消
        </button>
        <button className="btn primary" onClick={save} disabled={busy}>
          {busy ? "保存中…" : "保存"}
        </button>
      </div>
    </div>
  );
}
