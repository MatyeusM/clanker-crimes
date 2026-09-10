import { useCallback, useEffect, useState } from "preact/hooks";

export type Theme = "dark" | "light";

const STORAGE_KEY = "clanker-todo:theme";

function isBrowser(): boolean {
  return typeof document !== "undefined" && typeof localStorage !== "undefined";
}

export function getSystemTheme(): Theme {
  if (!isBrowser() || typeof window.matchMedia !== "function") {
    return "light";
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

export function getStoredTheme(): Theme | null {
  if (!isBrowser()) {
    return null;
  }
  const raw = localStorage.getItem(STORAGE_KEY);
  return raw === "dark" || raw === "light" ? raw : null;
}

export function getInitialTheme(): Theme {
  return getStoredTheme() ?? getSystemTheme();
}

export function applyTheme(theme: Theme): void {
  if (!isBrowser()) {
    return;
  }
  document.documentElement.dataset.theme = theme;
  document.documentElement.style.colorScheme = theme;
}

export function useTheme() {
  const [theme, setTheme] = useState<Theme>(() => getInitialTheme());

  useEffect(() => {
    applyTheme(theme);
    try {
      localStorage.setItem(STORAGE_KEY, theme);
    } catch {
      return;
    }
  }, [theme]);

  const toggle = useCallback(() => {
    setTheme((prev) => (prev === "dark" ? "light" : "dark"));
  }, []);

  return { theme, toggle };
}

export function formatThemeLabel(theme: Theme): string {
  return theme === "dark" ? "Switch to light mode" : "Switch to dark mode";
}

export function getThemeIconName(theme: Theme): string {
  return theme === "dark" ? "sun" : "moon";
}
