# Web Client — Reference

Low-level reference for the Web Client. Covers file structure, SWR hook
inventory, auth token handling, API call patterns, and component detail.

For architecture overview and design decisions see [`client.md`](client.md).

**Status:** Phase 6b complete. Phase 6c in progress.

---

## File structure

```
web-client/
├── src/
│   ├── api/
│   │   ├── auth.jsx       # token store, AuthContext, doRefresh, useAuth
│   │   ├── fetcher.js     # SWR global fetcher, 401 intercept, retry
│   │   └── users.js       # updateMe() — imperative fetch helper for PUT /users/me
│   ├── components/
│   │   ├── Nav.jsx               # sticky nav, theme toggle, user email, user menu
│   │   ├── ProtectedRoute.jsx    # silent refresh on mount, redirects to /login
│   │   ├── RoomCard.jsx          # dashboard room card, independent climate fetch
│   │   ├── TimezonePrompt.jsx    # dismissible UTC timezone setup banner
│   │   └── ui/                   # populated by npx shadcn add as needed
│   ├── hooks/
│   │   ├── useUser.js        # GET /users/me
│   │   ├── useRooms.js       # GET /rooms, 30s polling
│   │   ├── useRoom.js        # GET /rooms/:id
│   │   ├── useClimate.js     # GET /rooms/:id/climate, 30s polling, 204 handling
│   │   └── useSchedules.js   # GET /rooms/:id/schedules
│   ├── lib/
│   │   ├── helpers.js     # timeAgo, fmtTime12, fmtMin12, fmtTick12
│   │   └── utils.js       # shadcn cn() helper
│   ├── pages/
│   │   ├── LoginPage.jsx
│   │   ├── RegisterPage.jsx
│   │   ├── DashboardPage.jsx
│   │   ├── RoomDetailPage.jsx
│   │   ├── DevicesPage.jsx        # placeholder until 6f
│   │   └── tabs/
│   │       └── OverviewTab.jsx    # current state card + control panel shell
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
// auth.jsx
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

## Custom fetcher — useClimate

`GET /rooms/:id/climate` returns 204 when no control log data exists yet. The
global fetcher throws on non-2xx responses but cannot distinguish 204 from an
error. `useClimate` uses an inline custom fetcher that handles 204 explicitly,
returning `null` as a valid no-data state. The 401 retry logic is replicated
manually since the global fetcher is bypassed.

```js
// hooks/useClimate.js
async (url) => {
  const res = await fetch(url, {
    credentials: 'include',
    headers: { Authorization: `Bearer ${getToken()}` },
  })
  if (res.status === 204) return null
  if (res.status === 401) {
    await doRefresh()
    const retry = await fetch(url, {
      credentials: 'include',
      headers: { Authorization: `Bearer ${getToken()}` },
    })
    if (retry.status === 204) return null
    if (!retry.ok) throw new Error(retry.status)
    return retry.json()
  }
  if (!res.ok) throw new Error(res.status)
  return res.json()
}
```

---

## Imperative API helpers

`src/api/users.js` — imperative fetch for `PUT /users/me`. Used by
`DashboardPage` to save timezone from `TimezonePrompt`. Not a SWR hook — called
directly in event handlers. Must attach Authorization header manually since it
bypasses the SWR fetcher.

```js
export async function updateMe(payload) {
  const res = await fetch('/api/v1/users/me', {
    method: 'PUT',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${getToken()}`,
    },
    body: JSON.stringify(payload),
  })
  if (!res.ok) throw new Error(res.status)
}
```

All other write operations (rename, delete, add room) are implemented as inline
`fetch` calls in their respective components rather than extracted helpers — only
`updateMe` is extracted because it is shared between `DashboardPage` and will be
used by the Account Settings modal in 6g.

---

## SWR hook inventory

| Hook | File | Endpoint | Interval | Exposes mutate |
|---|---|---|---|---|
| `useUser` | `hooks/useUser.js` | `GET /users/me` | none | yes |
| `useRooms` | `hooks/useRooms.js` | `GET /rooms` | 30s | yes |
| `useRoom(roomId)` | `hooks/useRoom.js` | `GET /rooms/:id` | none | yes |
| `useClimate(roomId)` | `hooks/useClimate.js` | `GET /rooms/:id/climate` | 30s | no |
| `useSchedules(roomId)` | `hooks/useSchedules.js` | `GET /rooms/:id/schedules` | none | no |

Hooks that expose `mutate` do so because there are write operations in the app
that need to invalidate that cache immediately. `useClimate` does not expose
`mutate` — climate data reflects control loop output, not direct user writes, so
immediate invalidation after a write would return stale data anyway.

---

## Route structure

```
/login                          LoginPage
/register                       RegisterPage
/                               → redirect to /dashboard
/dashboard                      DashboardPage
/rooms/:id                      RoomDetailPage
/devices                        DevicesPage (placeholder until 6f)
```

All routes except `/login` and `/register` are wrapped in `ProtectedRoute`.

---

## Component detail

### Nav.jsx

Sticky 56px top bar. Brand mark (Thermometer icon + "Climate Control" text),
Dashboard/Devices nav links with active underline indicator, theme toggle
(Sun/Moon icon), user menu dropdown.

User menu shows user email from `useUser` with truncation (`maxWidth: 160`,
`overflow: hidden`, `textOverflow: ellipsis`, `whiteSpace: nowrap`). Avatar
initial derived from `user?.email?.[0]?.toUpperCase()`. Falls back to `'U'`
and `'Account'` while user loads.

Kebab close on outside click via `useEffect` + `mousedown` listener — same
pattern as `RoomDetailPage` kebab.

Logout calls `POST /auth/logout` best-effort (fire-and-forget in try/catch),
then calls `logout()` from `useAuth` regardless of server response.

### ProtectedRoute.jsx

Attempts silent refresh on mount via `doRefresh()`. Lazy `useState` initializer
avoids synchronous setState-in-effect lint violation. Renders null while checking.
Redirects to `/login` on failure. Empty dependency array intentional — mount-only.

### TimezonePrompt.jsx

Shown when `user?.timezone === 'UTC'`. Auto-detects browser timezone via
`Intl.DateTimeFormat().resolvedOptions().timeZone`. Curated list of ~27 IANA
timezones in a `<select>`. Save button disabled when selected equals current
timezone. `onSave` calls `updateMe({ timezone })` then `mutate()` from `useUser`.
Dismissed state in localStorage under `cc-timezone-prompt-dismissed`.

Note: dismiss key is not per-user — acceptable for single-user self-hosted
deployment. Full timezone picker with grouped UTC offset labels deferred to 6g
Account Settings modal.

### RoomCard.jsx

Dashboard card for a single room. Independently calls `useClimate(room.id)` at
30s. Hover state via `useState` — border strengthens, shadow lifts,
`translateY(-1px)`.

Card uses `display: flex, flex-direction: column, height: 100%` so the bottom
badge section is always pinned to the bottom regardless of content. Grid uses
`align-items: stretch` so cards in the same row reach equal height.

**Top section (flex 1):** room name + mode badge row, then temp + humidity
readouts side by side. Readouts use `cc-readout-sm` with heat/cool color tokens.
`—` shown when climate is null or reading is null.

**Bottom section (margin-top auto):** always renders, may be empty. Contains:
- Heater badge — only if `climate?.heater_cmd !== null`. `cc-badge--heat` when on.
- Humidifier badge — only if `climate?.humidifier_cmd !== null`. `cc-badge--cool`
  when on.
- Control source badge — only when source is `manual_override`, `schedule`, or
  `grace_period`. Omitted when `none`. Uses `cc-badge` base + inline color
  overrides since `--cc-hold-border` and `--cc-grace-border` are not defined as
  CSS variables.

Mode badge: `cc-badge cc-badge--ok` for AUTO, `cc-badge` base for OFF.

### DashboardPage.jsx

Fetches room list via `useRooms` (30s). Renders `TimezonePrompt` at top.
Renders room grid below — `repeat(auto-fill, minmax(280px, 1fr))`, `gap: 16px`,
`align-items: stretch`.

Page header always renders (room count + Add Room button) regardless of loading
or empty state. Room grid renders nothing while loading or when rooms is empty —
proper loading/empty states deferred to 6g.

Add Room modal — `POST /rooms` with `{ name }` only (deadbands defaulted
server-side). 409 → "A room with that name already exists." Calls `mutateRooms()`
on 201.

### RoomDetailPage.jsx

Reads `roomId` from `useParams()`. Fetches room via `useRoom(roomId)`.

Tab state: `useState('overview')`. Tab bar uses an absolutely positioned 2px div
for the active underline indicator (bottom: -1px to overlap the tab bar border).

Header: cc-h1 room name + pencil `cc-iconbtn` for rename + kebab menu. Kebab
closes on outside click via `useEffect` mousedown listener.

**Rename modal:** pre-fills `renameValue` with `room?.name`. PUT sends full room
body (name + existing deadbands) since PUT is full replacement. 409 → specific
error. Calls `mutateRoom()` on success.

**Delete modal:** confirmation copy includes room name. DELETE calls
`mutateRooms()` then navigates to `/dashboard` on success.

Both modals: overlay click closes, `stopPropagation` on inner card, Enter key
submits (rename only), `autoFocus` on input.

### OverviewTab.jsx

Props: `roomId` (string).

Calls `useClimate(roomId)` and `useSchedules(roomId)`.

**Card 1 — Current state:**
- Header: "Current state" label + live indicator dot (green within 5 min,
  amber if stale, grey if no data — computed from `climate?.time` vs `Date.now()`)
- Readout grid: 2-column, Temperature left / Humidity right. Each column has a
  label row (icon + cc-meta text) above a `cc-readout` span with heat/cool color.
  Unit as inline `0.45em` span inside the readout.
- Actuator section: always renders. Heater row + Humidifier row. Each: icon +
  label + ON/OFF mono text. `—` when climate null or cmd null.
- Source + mode row: always renders. Source badge always present — "None" in
  neutral style when source is `none`/null. Mode badge only when source is
  `manual_override`, `schedule`, or `grace_period`.
- Targets section: always renders as 2-column grid. Each column: cc-meta label
  + mono value with inline deadband. `—` in `--cc-fg-4` when null.

**Card 2 — Control panel shell (ControlPanelShell):**

Local state: `controlType` ('schedule'/'manual'), `mode` ('OFF'/'AUTO'),
`tempTarget` (22.0 placeholder), `humTarget` (50.0 placeholder).

- Control type row: "Control type" label + `cc-seg` with Schedule/Manual buttons
- Schedule section: section label + bordered surface-2 box with status dot,
  schedule name (or "No active schedule"), "Overridden by manual" meta when
  `isManual`. Box dims to `opacity: 0.5` when `isManual`.
- Manual settings section: section label with "Schedule active" note when
  `!isManual`. Entire section body at `opacity: 0.5` when `!isManual`.
  - Mode row: `cc-seg` OFF/AUTO, `cc-seg--disabled` when `!isManual`
  - Capability rows: `cc-row` + `cc-row--disabled` when `manualRowDisabled`
    (`!isManual || !isAuto`). Each row: `cc-togdot` + icon + label + `cc-input
    cc-input--mono` (disabled) + `cc-dbpill` (cursor default, no onClick).
- Footer: disabled Revert + Apply buttons + "Control panel coming soon" meta note.

Placeholder values replaced with real desired state data in 6c after schema
migration. `useEffect` will be required to sync draft state when desired state
loads.

---

## Design system

Design tokens are defined in `web-client/src/styles/tokens.css` as `--cc-*` CSS
custom properties. All component styles are defined as `cc-*` CSS classes in the
same file and are globally available.

### Token reference

**Missing border tokens** — `--cc-hold-border`, `--cc-grace-border`,
`--cc-success-border` are not defined as CSS variables. Use inline rgba values:
- Hold border: `rgba(161, 98, 7, 0.30)`
- Grace border: `rgba(100, 116, 139, 0.30)`
- Success border: `rgba(22, 163, 74, 0.30)`
- Info border: `rgba(37, 99, 235, 0.30)`

**Correct token names** (common mistakes to avoid):
- Temperature color: `--cc-heat` (not `--cc-heat-base`)
- Humidity color: `--cc-cool` (not `--cc-cool-base`)
- Page padding: `--cc-page-pad-x` exists, `--cc-page-pad-y` does not — use `32px`

### cc-* class reference (relevant subset)

| Class | Purpose |
|---|---|
| `cc-card` | Surface card with border, shadow, border-radius |
| `cc-badge` | Base badge — pill shape, neutral surface-2 background |
| `cc-badge--heat` | Heat-toned badge (heater on state) |
| `cc-badge--cool` | Cool-toned badge (humidifier on state) |
| `cc-badge--ok` | Success-toned badge (AUTO mode) |
| `cc-readout` | Large mono numeric display (48px) |
| `cc-readout-sm` | Smaller mono numeric display (24px) |
| `cc-label` | Uppercase mono label |
| `cc-section-label` | Smaller uppercase mono section heading |
| `cc-meta` | Small muted text |
| `cc-btn` | Base button |
| `cc-btn--primary` | Primary filled button |
| `cc-btn--secondary` | Secondary outlined button |
| `cc-btn--ghost` | Ghost button |
| `cc-btn--danger` | Danger filled button |
| `cc-btn--sm` | Small button modifier |
| `cc-iconbtn` | 28px square icon button |
| `cc-input` | Base input |
| `cc-input--mono` | Mono font input (for numeric values) |
| `cc-seg` | Segmented control container |
| `cc-seg--disabled` | Disabled segmented control |
| `cc-togdot` | Toggle dot |
| `cc-togdot--on` | Active toggle dot |
| `cc-togdot--disabled` | Disabled toggle dot |
| `cc-dbpill` | Deadband pill with dashed border |
| `cc-row` | Flex row, align-items center, gap 12, min-height 32 |
| `cc-row--disabled` | Greyed row (opacity 0.5) |
| `cc-statusdot` | 8px status indicator dot |
| `cc-pop` | Popover/dropdown shell |
| `cc-modal-bg` | Modal overlay |
| `cc-modal` | Modal container |
| `cc-modal-head` | Modal header with border-bottom |
| `cc-modal-body` | Modal scrollable body |
| `cc-modal-foot` | Modal footer with border-top, surface-2 background |

---

## Patterns

### Direct fetch in event handlers

Write operations that aren't shared between components use inline `fetch` calls
rather than extracted helpers. Pattern:

```js
async function handleAction() {
  setError(null)
  setLoading(true)
  try {
    const res = await fetch('/api/v1/resource/id', {
      method: 'PUT',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${getToken()}`,
      },
      body: JSON.stringify(payload),
    })
    if (res.status === 409) { setError('Specific conflict message.'); return }
    if (!res.ok) { setError('Something went wrong.'); return }
    await mutate()
    setOpen(false)
  } catch {
    setError('Something went wrong.')
  } finally {
    setLoading(false)
  }
}
```

Key points:
- Always `await mutate()` after success — not fire-and-forget
- Distinguish 409 from generic errors with specific messages
- `finally` ensures loading state clears regardless of outcome
- Authorization header required on all direct fetch calls — not handled by SWR

### Modal pattern

```jsx
{open && (
  <div className="cc-modal-bg" onClick={() => setOpen(false)}>
    <div className="cc-modal" onClick={e => e.stopPropagation()}>
      <div className="cc-modal-head">...</div>
      <div className="cc-modal-body">...</div>
      <div className="cc-modal-foot">...</div>
    </div>
  </div>
)}
```

### Outside click close (kebab/dropdown)

```js
const ref = useRef(null)
useEffect(() => {
  if (!open) return
  function onMouseDown(e) {
    if (ref.current && !ref.current.contains(e.target)) setOpen(false)
  }
  document.addEventListener('mousedown', onMouseDown)
  return () => document.removeEventListener('mousedown', onMouseDown)
}, [open])
```