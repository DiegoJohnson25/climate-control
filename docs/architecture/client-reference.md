# Web Client — Reference

Low-level reference for the Web Client. Covers file structure, SWR hook
inventory, auth token handling, API call patterns, and component detail.

For architecture overview and design decisions see [`client.md`](client.md).

**Status:** Phase 6c complete. Phase 6d in progress.

---

## File structure

```
web-client/
├── src/
│   ├── api/
│   │   ├── auth.jsx       # token store, AuthContext, doRefresh, useAuth
│   │   ├── fetcher.js     # SWR global fetcher, 401 intercept, retry
│   │   ├── rooms.js       # updateDesiredState() — imperative PUT helper
│   │   └── users.js       # updateMe() — imperative fetch helper for PUT /users/me
│   ├── components/
│   │   ├── Nav.jsx               # sticky nav, theme toggle, user email, user menu
│   │   ├── ProtectedRoute.jsx    # silent refresh on mount, redirects to /login
│   │   ├── RoomCard.jsx          # dashboard room card, independent climate fetch
│   │   ├── TimezonePrompt.jsx    # dismissible UTC timezone setup banner
│   │   ├── TolerancesModal.jsx   # shared tolerances modal, showHints prop
│   │   └── ui/                   # populated by npx shadcn add as needed
│   ├── hooks/
│   │   ├── useUser.js          # GET /users/me
│   │   ├── useRooms.js         # GET /rooms, 30s polling
│   │   ├── useRoom.js          # GET /rooms/:id
│   │   ├── useClimate.js       # GET /rooms/:id/climate, 30s polling, 204 handling
│   │   ├── useDesiredState.js  # GET /rooms/:id/desired-state, revalidateOnFocus: false
│   │   └── useSchedules.js     # GET /rooms/:id/schedules
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
│   │       └── OverviewTab.jsx    # current state card + fully wired control panel
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
directly in event handlers.

`src/api/rooms.js` — imperative fetch for `PUT /rooms/:id/desired-state`.
Used by `ControlPanel` Apply handler. Parses the error response body and attaches
`.status` to thrown errors so callers can distinguish 422 validation failures from
500s:

```js
export async function updateDesiredState(roomId, payload) {
  const res = await fetch(`/api/v1/rooms/${roomId}/desired-state`, {
    method: 'PUT',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${getToken()}`,
    },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    const err = new Error(body.error || res.status)
    err.status = res.status
    throw err
  }
}
```

All other write operations use inline `fetch` calls in their respective components.
`updateMe` and `updateDesiredState` are extracted because they are shared across
multiple components.

---

## SWR hook inventory

| Hook | File | Endpoint | Interval | Exposes mutate |
|---|---|---|---|---|
| `useUser` | `hooks/useUser.js` | `GET /users/me` | none | yes |
| `useRooms` | `hooks/useRooms.js` | `GET /rooms` | 30s | yes |
| `useRoom(roomId)` | `hooks/useRoom.js` | `GET /rooms/:id` | none | yes |
| `useClimate(roomId)` | `hooks/useClimate.js` | `GET /rooms/:id/climate` | 30s | no |
| `useDesiredState(roomId)` | `hooks/useDesiredState.js` | `GET /rooms/:id/desired-state` | none | yes |
| `useSchedules(roomId)` | `hooks/useSchedules.js` | `GET /rooms/:id/schedules` | none | no |

`useDesiredState` uses `revalidateOnFocus: false` — desired state only changes on
explicit user action. Focus revalidation would clobber unsaved draft changes.
Revalidates only after a successful Apply via `mutateDesiredState()`.

---

## Route structure

```
/login                          LoginPage
/register                       RegisterPage
/dashboard                      DashboardPage
/rooms/:id                      RoomDetailPage (tab state local)
/devices                        DevicesPage (placeholder)
/                               redirect → /dashboard
```

---

## Component detail

### TolerancesModal.jsx

Shared modal component for editing room deadband tolerances. Used in two contexts:

**From `ControlPanel` dbpills** — `showHints={true}`. Hint text shows live
switching thresholds computed from draft targets:
`"Heater turns on below X°C, off above Y°C"`. Hints suppressed when the field
has a validation error (hint and error would conflict). `tempTarget`/`humTarget`
props come from the live draft state.

**From `RoomDetailPage` kebab** — `showHints={false}`. No hints — draft targets
are not available in that context. Pre-fills from `room.deadband_temp` and
`room.deadband_hum`.

Both callers handle the `onSave` PUT inline — `TolerancesModal` receives
`onSave({ deadband_temp, deadband_hum })` and calls it on successful validation.
Callers are responsible for the fetch and `mutateRoom()`.

Validated ranges: temperature 0.1–10.0°C, humidity 0.5–20.0%.

Modal state is consolidated into a single object to satisfy the
`react-hooks/set-state-in-effect` lint rule. Reinitialised on `[open, room]`
effect. `resetCount` is a separate `useState` incremented on open to force input
remount via `key` prop.

### RoomCard.jsx

Card used in the dashboard grid. Fetches its own climate data via `useClimate`
(30s polling) — independent per card. Flex-column layout with bottom section
pinned via `margin-top: auto`.

### DashboardPage.jsx

Fetches room list via `useRooms` (30s). Renders `TimezonePrompt` at top.
Add Room modal — `POST /rooms` with `{ name }` only (deadbands defaulted
server-side). 409 → "A room with that name already exists." Calls `mutateRooms()`
on 201.

### RoomDetailPage.jsx

Reads `roomId` from `useParams()`. Fetches room via `useRoom(roomId)`.

Tab state: `useState('overview')`.

Header: room name + pencil `cc-iconbtn` for rename + kebab menu. Kebab closes on
outside click via `useEffect` mousedown listener.

**Kebab actions:** Edit Name, Edit tolerances, Delete Room. "Edit tolerances"
opens `TolerancesModal` with `showHints={false}`.

**Rename modal:** pre-fills `renameValue` with `room?.name`. PUT sends full room
body (name + existing deadbands). 409 → specific error. Calls `mutateRoom()`.

**Tolerances modal (from kebab):** pre-fills from `room.deadband_temp`/`room.deadband_hum`.
PUT sends full room body (existing name + new deadbands). Calls `mutateRoom()`.

**Delete modal:** confirmation includes room name. DELETE calls `mutateRooms()`
then navigates to `/dashboard`.

Props passed to `OverviewTab`: `roomId`, `capabilities`, `room`, `mutateRoom`.

### OverviewTab.jsx

Props: `roomId`, `capabilities`, `room`, `mutateRoom`.

Calls `useClimate(roomId)`. Renders two cards in a CSS grid with
`align-items: stretch` so both cards reach equal height.

**Card 1 — CurrentStateCard:**
- Live/Stale/No data indicator dot computed from `climate?.time`
- Temperature + humidity readouts
- Heater + humidifier command rows — always rendered, `—` when null
- Active source badge + mode badge
- Target temperature + humidity with inline deadband display
- Deadband values sourced from `climate` snapshot (historical record)

**Card 2 — ControlPanel:**
- Draft state: `controlType`, `mode`, `tempTarget` (float|null),
  `humTarget` (float|null), `tempEnabled`, `humEnabled`
- Separate error state: `tempTargetError`, `humTargetError`
- `resetCount` — `useRef`, incremented on Revert and `useEffect` reinit to force
  uncontrolled input remount
- `isDirty` — compares draft against `desiredState`; gates Apply and Revert
- Control type seg → `manual_active`
- Mode seg → `mode`; tooltip explains disabled state
- Capability rows — togdot and content opacity are independent:
  - `tempHardDisabled` = `!capabilities.temperature || !isManual || !isAuto`
  - `tempContentDim` = `tempHardDisabled || !tempEnabled`
  - Togdot: `cc-togdot--disabled` when hard-disabled, `cc-togdot--on` when enabled,
    base (white) when available but not enabled
- Target inputs: uncontrolled (`defaultValue`), blur validation (5–40°C, 10–90%)
- Deadband pills: `cursor: pointer`, `onClick` opens `TolerancesModal`.
  Source: `room.deadband_temp`/`room.deadband_hum` — not climate snapshot —
  so pills update immediately after tolerances save via `mutateRoom()`
- Apply: validates, builds payload, calls `updateDesiredState`, then
  `mutateDesiredState()`. 422 responses surface API error message.
- Revert: resets draft to current `desiredState` values, increments `resetCount`
- `TolerancesModal` rendered with `showHints={true}`, `onSave` calls
  `PUT /rooms/:id` inline then `mutateRoom()`

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
| `cc-togdot` | Toggle dot — base state (white/neutral, clickable) |
| `cc-togdot--on` | Active toggle dot |
| `cc-togdot--disabled` | Disabled toggle dot (grey, not clickable) |
| `cc-dbpill` | Deadband pill with dashed border — clickable, opens TolerancesModal |
| `cc-row` | Flex row, align-items center, gap 12, min-height 32 |
| `cc-row--disabled` | Greyed row (opacity 0.5) |
| `cc-statusdot` | 8px status indicator dot |
| `cc-pop` | Popover/dropdown shell |
| `cc-modal-bg` | Modal overlay |
| `cc-modal` | Modal container |
| `cc-modal-head` | Modal header with border-bottom |
| `cc-modal-body` | Modal scrollable body |
| `cc-modal-foot` | Modal footer with border-top, surface-2 background |
| `cc-tooltip` | Tooltip wrapper — shows `data-tooltip` text above element on hover |
| `cc-tooltip--right` | Modifier — anchors tooltip to right edge instead of centering |

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

### Uncontrolled numeric input with reset key

```jsx
const resetCount = useRef(0)

// On reset (Revert or useEffect reinit):
resetCount.current += 1

// In JSX:
<input
  key={`field-name-${resetCount.current}`}
  className="cc-input cc-input--mono"
  defaultValue={value != null ? value.toFixed(1) : ''}
  onBlur={e => { /* validate and update float state */ }}
  style={{ borderColor: error ? 'var(--cc-danger)' : undefined }}
/>
```

### Tooltip usage

```jsx
<div
  className="cc-tooltip"
  data-tooltip={tooltip || undefined}
>
  {/* content */}
</div>

// Right-aligned variant (for elements near the right edge of a card):
<div
  className="cc-tooltip cc-tooltip--right"
  data-tooltip={tooltip || undefined}
>
  {/* content */}
</div>
```

Setting `data-tooltip={undefined}` (not an empty string) suppresses the tooltip
entirely — the `::after` pseudo-element only renders when the attribute is present.