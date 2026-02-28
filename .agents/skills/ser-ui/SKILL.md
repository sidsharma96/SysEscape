---
   name: ser-ui
   description: Use when building Systems Escape Rooms UI components. Applies the Grafana-inspired dark theme design system.
   ---
   
   ## SER Design System Rules
   
   Theme: Grafana-inspired dark monitoring dashboard.
   
   Colors (from tailwind.config.ts):
   - Backgrounds: surface-dark (page), panel-bg (cards/panels), surface-mid (hover)
   - Text: white/gray-300 (primary), gray-400 (secondary), gray-500 (muted)
   - Borders: gray-700/800, subtle 1px
   - Signals: signal-ok (green), signal-warn (yellow/amber), signal-crit (red)
   
   Components:
   - Cards: panel-bg background, rounded-lg, border border-gray-800, hover:border-gray-600 transition
   - Badges: small rounded-full px-2 py-0.5 text-xs font-medium, color from signal tokens
   - Loading: skeleton pulses with surface-mid, never spinners
   - Empty states: centered text-gray-500, no illustrations
   
   Typography: system font stack (already in tailwind config). No Google Fonts.
   Layout: CSS Grid, responsive (1 col mobile, 2 md, 3 lg). No asymmetry. Dashboard aesthetic.
   
   Do NOT: import external fonts, use purple/gradient accents, use Inter/Roboto, 
   add animations beyond subtle hover transitions, use shadcn/ui components.
