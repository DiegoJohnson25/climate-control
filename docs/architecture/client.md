# Web Client — Architecture Reference

React SPA served as static files by NGINX. Consumes the API Server REST API via
NGINX proxy. All API calls use `credentials: "include"` so the httpOnly refresh
cookie is sent automatically.

**Status:** Phase 6c complete. Phase 6d in progress.

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
      │     ├── Overview     current state card + control panel
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
| `useRoom(roomId)` | `GET /rooms/:id` | none | Exposes `mutate` — called after rename, deadband save |
| `useClimate(roomId)` | `GET /rooms/:id/climate` | 30s | Custom fetcher — handles 204 as null |
| `useSchedules(roomId)` | `GET /rooms/:id/schedules` | none | Used by control panel |
| `useDesiredState(roomId)` | `GET /rooms/:id/desired-state` | none | `revalidateOnFocus: false` — prevents draft clobber. Exposes `mutate` — called after Apply |

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

**Control panel:** capability-aware greying of input rows. Togdot and content
opacity are independent — the togdot is never dimmed by content state so it
remains visually clickable even when the row is otherwise greyed.

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

Each capability row has an independent enable/disable toggle (the `cc-togdot`).
When a capability is available but the user does not want to regulate it, the
togdot is white (base state) and clickable even while the rest of the row is
greyed. This communicates that the row is intentionally inactive, not broken.

**`isDirty` gate:** Apply and Revert are only enabled when the draft differs from
the saved desired state. The Apply button uses `cc-btn--primary` (filled) when
dirty and `cc-btn--ghost` (outlined) when clean — only looks actionable when
there is something to send.

**Draft initialisation:** `useEffect` with `[desiredState]` dep populates the
draft when `useDesiredState` first resolves. `revalidateOnFocus: false` on the
hook prevents SWR from revalidating on tab-away, which would clobber unsaved
draft changes.

**Apply payload:** targets are sent as null only when the user has explicitly
toggled that capability off (`tempEnabled`/`humEnabled = false`). Mode and control
type do not affect whether target values are preserved — this ensures saved
preferences survive switching between Schedule and Manual or between AUTO and OFF.

**Validation:** target inputs and tolerance inputs validate on blur. Red outline
+ error message appear when the field loses focus with an invalid value. Apply is
blocked if any validation errors are present.

**Tolerances modal:** accessible from the `cc-dbpill` elements in the control
panel (with live threshold hints computed from draft targets) and from "Edit
tolerances" in the room detail kebab menu (without hints). Title: "Tolerances".
Subtitle: "Wider tolerances save energy but allow more drift."

This maps to `manual_active` (Control type toggle), `mode`, and targets in the
`desired_states` table.

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

**Tooltip utility classes:**
- `cc-tooltip` — adds a CSS `::after` pseudo-element tooltip anchored above the
  element, centered horizontally. Applied via `data-tooltip="..."` attribute.
  Only renders when `data-tooltip` is present — safe to set to `undefined` when
  no tooltip is needed.
- `cc-tooltip--right` — modifier that anchors the tooltip to the right edge of
  the element instead of centering. Used for right-aligned controls (e.g. the
  mode seg control) where a centered tooltip would clip off the card edge.

**Uncontrolled numeric inputs:** inputs that require stable cursor position during
typing use `defaultValue` instead of `value`. Programmatic resets (Revert, draft
reinit) use a `key` prop tied to a `useRef` counter (`resetCount.current`). The
counter is incremented on each reset, forcing React to remount the input with the
new `defaultValue`.

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

**`revalidateOnFocus: false` on `useDesiredState`** — desired state only changes
on explicit user action. Focus revalidation would clobber unsaved draft changes if
the user tabs away while editing. The hook revalidates only after a successful
Apply via `mutateDesiredState()`.

**Targets preserved across mode and control type changes** — the Apply payload
only nulls a target when the user has explicitly toggled that capability off.
Switching from Manual to Schedule, or from AUTO to OFF, preserves the saved target
values in the database. The control loop only reads them when `manual_active` is
true and mode is AUTO — storing them alongside other states is intentional.

**Deadband pills read from `room` object, not climate snapshot** — `climate`
includes deadband values snapshotted at tick time, but deadbands are a room
property. Reading from `room` means the pills update immediately after a
tolerances save via `mutateRoom()`, without waiting for the next climate poll.

**Tolerances modal accessible from two entry points** — dbpills in the control
panel open the modal with live threshold hints (computed from draft targets).
The room detail kebab opens the same modal without hints, since the draft is not
available in that context. `showHints` prop controls which variant renders.

---

## Known limitations and deferred items

**Target preservation when toggling capability off** — when the user is in Manual
+ AUTO mode and disables a capability (e.g. sets `tempEnabled = false`) then hits
Apply, that target is sent as null and cleared in the database. Re-enabling the
capability later requires re-entering the target. A proper fix would require
storing enabled/disabled intent separately from the target value. Not worth the
complexity for current project scope.

**Deferred to 6g:**
- Loading skeletons across all pages
- Empty states across all pages
- Account Settings modal with full timezone picker (grouped by UTC offset, ~40
  curated IANA entries with friendly labels — no external library needed)
- `TimezonePrompt` dismiss key is not per-user (`cc-timezone-prompt-dismissed` in
  localStorage) — acceptable for single-user self-hosted deployment