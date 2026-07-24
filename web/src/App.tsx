import { useEffect, useState } from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import { bootstrapFromPanel, isAuthed, subscribe } from "./store/session";
import Login from "./pages/Login";
import Bindings from "./pages/Bindings";

export default function App() {
  const [ready, setReady] = useState(false);
  const [, force] = useState(0);

  // Re-render when the session changes (e.g. cleared on 401).
  useEffect(() => subscribe(() => force((n) => n + 1)), []);

  useEffect(() => {
    let alive = true;
    (async () => {
      await bootstrapFromPanel();
      if (alive) setReady(true);
    })();
    return () => {
      alive = false;
    };
  }, []);

  if (!ready) {
    return (
      <div className="app">
        <p className="muted">加载中…</p>
      </div>
    );
  }

  return (
    <Routes>
      <Route
        path="/bindings"
        element={isAuthed() ? <Bindings /> : <Navigate to="/login" replace />}
      />
      <Route path="/login" element={<Login />} />
      <Route path="*" element={<Navigate to={isAuthed() ? "/bindings" : "/login"} replace />} />
    </Routes>
  );
}
