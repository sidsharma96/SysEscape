---
name: ser-ui
description: Use when building Systems Escape Rooms UI components. Applies the Grafana-inspired dark theme design system.
---

## SER Design System Rules

Theme: Grafana-inspired dark monitoring dashboard.
Tailwind: v4 (CSS-first config). Tokens defined in @theme block in web/src/styles/globals.css.

Colors (from @theme):
- Backgrounds: surface-dark (page), panel-bg (cards/panels), surface-mid (hover)
- Text: white/gray-300 (primary), gray-400 (secondary), gray-500 (muted)
- Borders: gray-700/800, subtle 1px
- Signals: signal-ok (green), signal-warn (yellow/amber), signal-crit (red)

Components:
- Cards: panel-bg background, rounded-lg, border border-gray-800, hover:border-gray-600 transition
- Badges: small rounded-full px-2 py-0.5 text-xs font-medium, color from signal tokens
- Loading: skeleton pulses with surface-mid, never spinners
- Empty states: centered text-gray-500, no illustrations

Typography: system font stack (already in @theme). No Google Fonts.
Layout: CSS Grid, responsive (1 col mobile, 2 md, 3 lg). No asymmetry. Dashboard aesthetic.

TAILWIND v4 RULES:
- No tailwind.config.ts exists. All config is in CSS via @theme.
- Import is: @import "tailwindcss" (not @tailwind base/components/utilities).
- Custom colors are used as: bg-surface-dark, text-signal-ok (automatic from @theme --color-* vars).
- Do NOT create tailwind.config.ts or tailwind.config.js.
- Do NOT use @tailwind directives.
- Do NOT use theme() function in JS. Use CSS variables directly if needed.
- Do NOT import external fonts. Use system font stack.
- Do NOT add shadcn/ui.
