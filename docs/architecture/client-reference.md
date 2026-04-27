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
│   │   ├── auth.js        # in-memory token store, refresh logic, deduplication
│   │   └── fetcher.js     # SWR global fetcher, 401 intercept, retry
│   ├── components/
│   │   ├── Nav.jsx        # persistent top nav — Dashboard + Devices links, UserMenu
│   │   └── ProtectedRoute.jsx  # redirects to /login if no token + refresh fails
│   ├── pages/
│   │   ├── LoginPage.jsx
│   │   ├── DashboardPage.jsx
│   │   ├── RoomDetailPage.jsx
│   │   └── DevicesPage.jsx
│   ├── App.jsx            # router, SWRConfig, auth context provider
│   └── main.jsx
├── mockup/                # static HTML/CSS mockup — served on port 8090 via make mockup
├── dist/                  # Vite build output — served by NGINX
├── index.html
└── vite.config.js
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
      .then(data => { setToken(data.access_token); })
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
/rooms/:id                      RoomDetailPage (tab: overview)
/rooms/:id/history              RoomDetailPage (tab: history)
/rooms/:id/schedules            RoomDetailPage (tab: schedules)
/rooms/:id/devices              RoomDetailPage (tab: devices)
/devices                        DevicesPage
```

All routes except `/login` are wrapped in `ProtectedRoute`.

---

## Design system

Design tokens defined in `web-client/mockup/styles/colors_and_type.css` as
`--cc-*` CSS variables. Component-level classes in
`web-client/mockup/styles/styles.css`.

Key token categories:
- `--cc-bg-*` — background layers (base, surface, raised, overlay)
- `--cc-text-*` — text hierarchy (primary, secondary, tertiary, disabled)
- `--cc-accent-*` — brand accent colour
- `--cc-status-*` — semantic status colours (heat, cool, ok, warn, off)
- `--cc-border-*` — border colours

Typography: Inter for all UI text. JetBrains Mono for all numeric readouts with
`font-variant-numeric: tabular-nums`. Time labels use 12-hour AM/PM format throughout.

See `web-client/mockup/README.md` for the full component inventory and interaction
state documentation.