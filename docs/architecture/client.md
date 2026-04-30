# Web Client — Architecture Reference

React SPA served as static files by NGINX. Consumes the API Server REST API via
NGINX proxy. All API calls use `credentials: "include"` so the httpOnly refresh
cookie is sent automatically.

**Status:** Phase 6b complete. Phase 6c in progress.

---

## Stack

| Technology | Purpose |
|---|---|
| React 19 | UI component framework |
| Vite | Build tooling and dev server |
| SWR | Server state — data fetching, polling, cache invalidation |
| shadcn/ui + Tailwind | Component library and utility CSS |
| Recharts | Time series charts for climate history |
| React Router v7 | Client-side routing |
| lucide-react | Icon library |

No TypeScript. No Redux. No TanStack Query.

---

## Auth implementation

Access token stored in React context (in-memory) — not localStorage, not
sessionStorage. Lost on page refresh — the httpOnly refresh cookie handles silent
re-authentication transparently.

**Login flow:** `POST /auth/login` → access token stored in context → redirect to
Dashboard.

**Page load:** silent `POST /auth/refresh` attempted before rendering protected
routes. Success → token stored, app renders. Failure → redirect to login.

**401 intercept:** SWR global fetcher intercepts 401 responses → triggers refresh
→ retries the original request once. If retry also 401s → redirect to login.

**Refresh deduplication:** a single in-flight refresh promise is shared across
concurrent 401s. If two SWR hooks 401 simultaneously, only one refresh call fires —
both hooks await the same promise. Prevents concurrent refresh races.

**Logout:** clear access token from context → `POST /auth/logout` to invalidate
refresh token server-side → redirect to login.

**Custom fetchers:** the global SWR fetcher handles 401 intercept and token
attachment for all standard endpoints. Endpoints that return non-standard status
codes (e.g. 204 for no-data) use inline custom fetchers that replicate the 401
retry logic manually — see `useClimate`.

---

## Navigation structure

```
Login / Register
└── Dashboard (room cards grid)
      ├── Room detail (tabbed)
      │     ├── Overview     current state card + control panel shell
      │     ├── History      stacked climate charts, window selector
      │     ├── Schedules    schedule list, inline period accordion, period modal
      │     └── Devices      full management
      └── Devices page       flat device list, inline room assignment
```

Persistent top nav on all authenticated screens. Maximum depth: three clicks from
login to any piece of data. Always lands on Dashboard on login — no single-room
redirect.

**Tab routing:** local `useState` only — URL stays `/rooms/:id` regardless of
active tab. No nested routes under `/rooms/:id`.

---

## SWR hook inventory

| Hook | Endpoint | Interval | Notes |
|---|---|---|---|
| `useUser` | `GET /users/me` | none | Exposes `mutate` — called after `PUT /users/me` |
| `useRooms` | `GET /rooms` | 30s | Exposes `mutate` — called after room create/delete |
| `useRoom(roomId)` | `GET /rooms/:id` | none | Exposes `mutate` — called after rename |
| `useClimate(roomId)` | `GET /rooms/:id/climate` | 30s | Custom fetcher — handles 204 as null |
| `useSchedules(roomId)` | `GET /rooms/:id/schedules` | none | Used by control panel shell |

All hooks that expose `mutate` call it immediately after a successful write so the
UI reflects changes without waiting for the next poll interval.

---

## Capability-aware rendering

Room capabilities come from the `capabilities` object on every room response:

```json
{ "capabilities": { "temperature": true, "humidity": false } }
```

Temperature capability = temperature sensor + heater both assigned to the room.
Humidity capability = humidity sensor + humidifier both assigned. The client never
infers capability from `ClimateReading` null fields — those indicate missing data,
not missing hardware.

**Always-show philosophy:** all rows and sections render regardless of capability
or data availability. Null values show `—`. Missing hardware greys rows via
`cc-row--disabled` or `opacity: 0.5`. No conditional hiding that causes layout
shifts.

**Dashboard cards:** climate reading nulls produce `—` display. No capability
checks needed at the card level.

**Overview current state card:** actuator rows always render. `heater_cmd: null`
shows `—` for the status, not a hidden row.

**Control panel:** capability-aware greying of input rows deferred to 6c.

---

## Control source label mapping

The `control_source` field from the API maps to user-facing labels. "Hold" and
"override" do not appear in the UI — "Manual" is the established term for user-
driven control.

| API value | UI label |
|---|---|
| `manual_override` | Manual |
| `schedule` | Schedule |
| `grace_period` | Grace period |
| `none` | None (muted, no badge variant) |

---

## Control panel design

The control panel (OverviewTab card 2) has two top-level states driven by a
"Control type" segmented control:

**Schedule** — the room follows its configured schedule. Schedule section shows
the active schedule name. Manual settings section (mode, targets) is greyed out
but still visible so the user can see their saved manual settings.

**Manual** — manual settings override the schedule. Schedule section dims with
"Overridden by manual" indicator. Manual settings section is interactive.

Mode (OFF/AUTO) is subordinate to Control type — only relevant when Manual is
selected. Capability rows (temperature, humidity) are subordinate to mode — only
interactive when Manual + AUTO.

This maps to `manual_active` (Control type toggle) and `mode`/targets in the
`desired_states` table. In 6b the control panel is a visual shell with placeholder
draft state — fully wired in 6c after the schema migration.

---

## History charts

Two stacked Recharts charts — temperature above, humidity below. Separated to
avoid dual-axis clutter and to give each chart's y-axis unambiguous meaning.

Each chart renders:
- Primary line — `avg_temp` or `avg_hum`
- Dashed target ± deadband overlay (toggleable)
- Background opacity fill for actuator duty cycle — warm tint for heater, cool
  tint for humidifier, opacity proportional to 0.0–1.0 duty fraction (toggleable)
- `connectNulls={false}` — gaps mean no data for that bucket, not interpolated

Time tick intervals by window: 1h → 15min, 6h → 1h, 24h → 6h, 7d → 24h.

---

## Design system

All visual styles come from `src/styles/tokens.css` as `--cc-*` CSS custom
properties. Component styles are `cc-*` CSS classes in the same file.

**Token categories:**
- `--cc-bg` / `--cc-surface` / `--cc-surface-2` — background layers
- `--cc-fg` / `--cc-fg-2` / `--cc-fg-3` / `--cc-fg-4` — text hierarchy
- `--cc-heat-*` / `--cc-cool-*` — thermal accent families (base, hover, tint,
  border, fg variants)
- `--cc-primary` / `--cc-primary-hover` / `--cc-primary-fg` — interaction primary
  (inverts in dark mode automatically)
- `--cc-border` / `--cc-border-strong` / `--cc-divider` — borders
- `--cc-success-*` / `--cc-warning-*` / `--cc-danger-*` / `--cc-info-*` — semantic
  status families
- `--cc-hold-*` / `--cc-grace-*` — control source badge accents
- `--cc-shadow-sm` / `--cc-shadow-md` / `--cc-shadow-lg` — shadows
- `--cc-radius-sm` / `--cc-radius-md` / `--cc-radius-lg` / `--cc-radius-pill` — radii
- `--cc-font-sans` / `--cc-font-mono` — Inter and JetBrains Mono
- `--cc-fs-*` — type scale (xs through 3xl)
- `--cc-dur-*` / `--cc-ease` / `--cc-ease-soft` — motion tokens

Dark mode: all tokens override under `[data-theme="dark"]` on `<html>`.
Toggle via `document.documentElement.setAttribute('data-theme', 'dark')`.

Typography: Inter for all UI text. JetBrains Mono for all numeric readouts with
`font-variant-numeric: tabular-nums`. Time labels use 12-hour AM/PM format
throughout (`fmtTime12`, `fmtMin12`, `fmtTick12` in `src/lib/helpers.js`).

**Responsive layout:** CSS grid with `minmax()` or proportional `fr` units
preferred over flexbox with fixed gaps — handles responsive reflow without awkward
whitespace. Two-column layouts use `repeat(auto-fit, minmax(Npx, 1fr))` so they
stack cleanly on narrow viewports.

**Button conventions:** title case throughout — "Add Room", "Delete Room",
"Log Out", "Save Timezone". Action buttons include a relevant lucide-react icon
where appropriate (Plus for add actions, etc.).

**Modals:** `cc-modal-bg` overlay + `cc-modal` pattern. Overlay click closes,
inner card click does not (stopPropagation). Enter key submits single-input modals.
409 conflicts show specific error messages. Successful mutations call `mutate()`
on relevant SWR hooks immediately.

---

## Key design decisions

**Access token in React context, not localStorage** — localStorage is readable by
any JavaScript on the page. React context keeps the token in memory, lost on page
refresh, with the httpOnly cookie handling silent re-authentication transparently.

**Two stacked history charts, not one combined** — separating temperature and
humidity eliminates dual-axis clutter, allows both duty cycle fills to be displayed
without visual interference, and keeps each chart's y-axis meaningful.

**Duty cycle as opacity fill, not a line** — background fill shows duty cycle as
contextual information without a third axis. Opacity proportional to the 0.0–1.0
fraction preserves nuance — partial on/off buckets are visible without cluttering
the primary data lines.

**"Manual" not "Hold" as the manual control term** — "Manual" is accurate and
self-explanatory. "Hold" implies temporary duration which is misleading when the
user sets an indefinite override. The word "Hold" and "override" are deliberately
absent from the UI.

**Control type as a peer selector, not an override toggle** — "Schedule" and
"Manual" are presented as equal alternatives via a segmented control, not as a
toggle that overrides something else. This accurately represents the system
behaviour and avoids implying that schedules are the default that must be overridden.

**Always-show, never-hide** — rows and sections always render with `—` for null
values and greyed states for unavailable capabilities. This prevents layout shifts
as data loads and communicates the full capability surface of the room — a greyed
row tells the user that capability could exist, not that it doesn't.

**Capabilities from room object, not climate readings** — `ClimateReading` nulls
indicate missing data, not missing hardware. Capability comes from the
`capabilities` object on the room response, which is derived from the actual device
graph in the database.

---

## Pending backend changes (required before Phase 6c)

**Desired state schema change** — add `manual_active BOOLEAN NOT NULL DEFAULT false`
and `manual_mode TEXT` columns. Targets (`target_temp`, `target_hum`) persist
independently of whether manual control is active. A user can have saved targets
while control type is set to Schedule. The control loop derives effective mode from
`manual_active` + expiry check rather than reading a `mode` column directly.

**`useDesiredState(roomId)` hook** — `GET /rooms/:id/desired-state`. Required to
pre-fill draft state in the control panel with real desired state values.

**`ControlPanelShell` → `ControlPanel`** — replace placeholder `useState` values
with real desired state data. Wire Apply → `PUT /rooms/:id/desired-state`. Wire
Revert → reset draft to desired state values. Use `useEffect` to sync draft when
desired state loads — `useState` only initialises once per mount.