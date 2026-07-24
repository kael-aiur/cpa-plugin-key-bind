// 跟随 Management-Center(同源 parent)的主题。
// 管理面板在 <html data-theme="dark"|"white"|无(light)> 上标记当前主题;
// 本页作为同源 iframe,读取 parent 的该属性并应用到自身 <html>。

const THEME_ATTR = "data-theme";

function readParentTheme(): "dark" | "light" {
  try {
    const doc = window.parent?.document;
    const t = doc?.documentElement?.getAttribute(THEME_ATTR);
    if (t === "dark") return "dark";
    // "white" 或无属性 → 亮色
    return "light";
  } catch {
    // 跨源或不可访问 → 退回系统偏好
    if (window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches) {
      return "dark";
    }
    return "light";
  }
}

export function applyThemeFromParent(): void {
  if (readParentTheme() === "dark") {
    document.documentElement.setAttribute(THEME_ATTR, "dark");
  } else {
    document.documentElement.removeAttribute(THEME_ATTR);
  }
}

// 监听 parent 主题变化(用户在管理面板切换主题时实时跟随)。
export function watchParentTheme(): () => void {
  try {
    const parentHtml = window.parent?.document?.documentElement;
    if (!parentHtml || typeof MutationObserver === "undefined") return () => {};
    const observer = new MutationObserver(() => applyThemeFromParent());
    observer.observe(parentHtml, { attributes: true, attributeFilter: [THEME_ATTR] });
    return () => observer.disconnect();
  } catch {
    return () => {};
  }
}
