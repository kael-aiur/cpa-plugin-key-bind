import React from "react";
import ReactDOM from "react-dom/client";
import { HashRouter } from "react-router-dom";
import App from "./App";
import { applyThemeFromParent, watchParentTheme } from "./themeSync";
import "./styles.css";

applyThemeFromParent();
watchParentTheme();

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <HashRouter>
      <App />
    </HashRouter>
  </React.StrictMode>,
);
