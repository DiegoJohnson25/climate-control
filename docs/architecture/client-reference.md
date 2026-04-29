# Web Client — Reference

Low-level reference for the Web Client. Covers file structure, SWR hook
inventory, auth token handling, API call patterns, and component detail.

For architecture overview and design decisions see [`client.md`](client.md).

> **Status:** Phase 6 is in progress. This document will be completed once
> Phase 6 implementation is finalised. The architecture overview in
> [`client.md`](client.md) reflects the confirmed design.

---

## File structure

```
web-client/
├── src/
│   ├── api/
│   │   ├── auth.jsx       # token store, AuthContext, doRefresh, useAuth
│   │   └── fetcher.js     # SWR global fetcher, 401 intercept, retry
│   ├── components/
│   │   ├── Nav.jsx        # sticky nav, theme toggle, user menu
│   │   ├── ProtectedRoute.jsx  # silent refresh on mount, redirects to /login
│   │   ├── TimezonePrompt.jsx  # dismissible UTC timezone setup banner
│   │   └── ui/            # populated by npx shadcn add as needed
│   ├── lib/
│   │   ├── helpers.js     # timeAgo, fmtTime12, fmtMin12, fmtTick12
│   │   └── utils.js       # shadcn cn() helper
│   ├── pages/
│   │   ├── LoginPage.jsx
│   │   ├── RegisterPage.jsx
│   │   ├── DashboardPage.jsx
│   │   ├── RoomDetailPage.jsx
│   │   └── DevicesPage.jsx
│   ├── styles/
│   │   └── tokens.css     # --cc-* tokens + cc-* component classes
│   ├── App.jsx            # router, SWRConfig, AuthProvider, ThemeProvider
│   ├── index.css          # imports tokens.css + tailwindcss, base body styles
│   └── main.jsx           # entry point
├── mockup/                # static HTML/CSS mockup — visual reference only
├── dist/                  # Vite build output — served by NGINX
├── public/                # static assets (favicon.svg, icons.svg)
├── components.json        # shadcn config
├── index.html             # Vite entry point
├── jsconfig.json          # @ alias for VS Code intellisense
├── vite.config.js         # Tailwind plugin, @ alias, /api dev proxy
└── package.json
```

---

## Vite dev proxy

```js
// vite.config.js
server: {
  proxy: {
    '/api': 'http://localhost'  // forwards to NGINX on port 80
  }
}
```

All API calls in development go through the Vite dev proxy to NGINX, then to the
API Server. No CORS configuration required. Production behaviour is identical —
the browser and API Server are on the same origin via NGINX.

---

## Auth token storage

Access token stored in React context — not localStorage, not sessionStorage.

```js
// auth.js
let accessToken = null;

export function getToken()       { return accessToken; }
export function setToken(token)  { accessToken = token; }
export function clearToken()     { accessToken = null; }
```

Refresh deduplication — prevents concurrent refresh calls when multiple SWR
hooks 401 simultaneously:

```js
let refreshPromise = null;

export async function doRefresh() {
  if (!refreshPromise) {
    refreshPromise = fetch('/api/v1/auth/refresh', {
      method: 'POST',
      credentials: 'include',
    })
      .then(r => { if (!r.ok) throw new Error('refresh failed'); return r.json(); })
      .then(data => { setToken(data.access_token); return data; })
      .finally(() => { refreshPromise = null; });
  }
  return refreshPromise;
}
```

---

## SWR global fetcher

```js
// fetcher.js
export async function fetcher(url) {
  const res = await fetch(url, {
    credentials: 'include',
    headers: { Authorization: `Bearer ${getToken()}` },
  });

  if (res.status === 401) {
    await doRefresh();
    const retry = await fetch(url, {
      credentials: 'include',
      headers: { Authorization: `Bearer ${getToken()}` },
    });
    if (retry.status === 401) {
      clearToken();
      window.location.href = '/login';
      throw new Error('Unauthorized');
    }
    return retry.json();
  }

  if (!res.ok) throw new Error(`${res.status}`);
  return res.json();
}
```

---

## SWR hook inventory

> To be completed after Phase 6 implementation.

Planned hooks:

| Hook | Endpoint | Interval | Used in |
|---|---|---|---|
| `useRooms` | `GET /rooms` | 30s | Dashboard |
| `useClimate(roomId)` | `GET /rooms/:id/climate` | 30s | Dashboard, Overview tab |
| `useRoom(roomId)` | `GET /rooms/:id` | on demand | Room detail |
| `useClimateHistory(roomId, window)` | `GET /rooms/:id/climate/history` | 60s | History tab |
| `useSchedules(roomId)` | `GET /rooms/:id/schedules` | on demand | Schedules tab |
| `usePeriods(scheduleId)` | `GET /schedules/:id/periods` | on demand | Schedules tab |
| `useDevices` | `GET /devices` | on demand | Devices page |
| `useRoomDevices(roomId)` | `GET /rooms/:id/devices` | on demand | Devices tab |

---

## Route structure

```
/login                          LoginPage
/                               → redirect to /dashboard
/dashboard                      DashboardPage
/rooms/:id                      RoomDetailPage
/devices                        DevicesPage
```

All routes except `/login` are wrapped in `ProtectedRoute`.

---

## Design system

Design tokens are defined in `web-client/src/styles/tokens.css` as
`--cc-*` CSS custom properties. All component styles are defined as
`cc-*` CSS classes in the same file and are globally available.

Key token categories:
- `--cc-bg` / `--cc-surface` / `--cc-surface-2` — background layers
- `--cc-fg` / `--cc-fg-2` / `--cc-fg-3` / `--cc-fg-4` — text hierarchy
- `--cc-heat-*` / `--cc-cool-*` — thermal accent families (each has
  base, hover, tint, border, fg variants)
- `--cc-primary` / `--cc-primary-hover` / `--cc-primary-fg` —
  interaction primary (inverts in dark mode automatically)
- `--cc-border` / `--cc-border-strong` / `--cc-divider` — borders
- `--cc-success-*` / `--cc-warning-*` / `--cc-danger-*` / `--cc-info-*`
  — semantic status families
- `--cc-hold-*` / `--cc-grace-*` — control source badge accents
- `--cc-shadow-sm` / `--cc-shadow-md` / `--cc-shadow-lg` — shadows
- `--cc-radius-sm` / `--cc-radius-md` / `--cc-radius-lg` /
  `--cc-radius-pill` — border radii
- `--cc-font-sans` / `--cc-font-mono` — Inter and JetBrains Mono
- `--cc-fs-*` — type scale (xs through 3xl)
- `--cc-dur-*` / `--cc-ease` / `--cc-ease-soft` — motion tokens

Dark mode: all tokens override under `[data-theme="dark"]` on `<html>`.
Toggle via `document.documentElement.setAttribute('data-theme', 'dark')`.

Typography: Inter for all UI text. JetBrains Mono for all numeric
readouts with `font-variant-numeric: tabular-nums`. Time labels use
12-hour AM/PM format throughout (`fmtTime12`, `fmtMin12`, `fmtTick12`
in `src/lib/helpers.js`).