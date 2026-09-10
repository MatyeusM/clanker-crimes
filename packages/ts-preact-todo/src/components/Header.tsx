import { Bot, Moon, Sun } from "lucide-preact";
import { useLocation } from "preact-iso";

import { formatThemeLabel, useTheme } from "../lib/theme";

export function Header() {
  const location: any = useLocation();
  const url = location?.url as string;
  const { theme, toggle }: any = useTheme();

  return (
    <header class="site-header">
      <div class="site-header__inner">
        <a
          class="brand"
          href="/">
          <span class="brand__icon">
            <Bot size={20} />
          </span>
          <span class="brand__text">Clanker TODO</span>
        </a>
        <div class="site-header__actions">
          <nav class="site-header__nav">
            <a
              class={
                url === "/"
                  ? "site-header__link site-header__link--active"
                  : "site-header__link"
              }
              href="/">
              Tasks
            </a>
            <a
              class={
                url === "/404"
                  ? "site-header__link site-header__link--active"
                  : "site-header__link"
              }
              href="/404">
              404
            </a>
          </nav>
          <button
            aria-label={formatThemeLabel(theme)}
            class="site-header__theme-toggle"
            onClick={toggle}
            title={formatThemeLabel(theme)}
            type="button">
            {theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}
          </button>
        </div>
      </div>
    </header>
  );
}
