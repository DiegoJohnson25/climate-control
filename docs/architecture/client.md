# Web Client — Design Reference

All UI/UX and implementation decisions agreed on across planning sessions.
Use this as the source of truth when building Phase 6.

---

## Stack

| Technology | Purpose |
|---|---|
| React 19 | UI component framework |
| JavaScript (not TypeScript) | Overhead not justified for this scope |
| Vite | Build tooling and dev server |
| SWR | Server state — data fetching, polling, cache |
| shadcn/ui + Tailwind | Component library and utility CSS |
| Recharts | Time series charts for history tab |
| React Router | Client-side routing |

**No Redux** — SWR covers all server state. Local UI state via `useState` is
sufficient. No complex cross-component state sync problem exists.

**No TypeScript** — low React familiarity, offloading React work to Claude,
thin UI over a clean REST API. Not worth the overhead at this scope.

**No TanStack Query / React Query** — SWR is the agreed choice. Simpler mental
model. React Query's mutation machinery and advanced cache control are not
needed here.

---

## Design system

The `--cc-*` CSS variable token system from the mockup is the canonical source
for colors, spacing, and typography. Wire into Tailwind via `theme.extend.colors`
referencing the CSS variables — do not re-implement the tokens.

Typography conventions:
- Inter (sans-serif) for UI text
- JetBrains Mono for all numeric readouts and mono-labeled metadata
- `font-variant-numeric: tabular-nums` on every numeric readout — prevents
  layout shift as values change. Enforce via Tailwind utility or `.cc-readout*`

Layout constants:
- 1280px max container
- 24px horizontal padding
- 56px top nav height

Time label format: 12-hour AM/PM style throughout (`2pm`, `12am`, `9:30am`).
Applied to history chart x-axis ticks, clock picker display, period table
start/end times, and all other time displays.

Live pulse: port the `cc-pulse` 600ms fade animation from the mockup. Apply
the class to readout components when their value changes.

---

## Auth

- JWT access token stored in **React context** (in-memory state) — not
  localStorage, not sessionStorage. Lost on page refresh — user re-authenticates
  via refresh token silently.
- Refresh token stored in **httpOnly cookie** — XSS safe, sent automatically
  by the browser on requests to the same origin. All fetch calls use
  `credentials: "include"`.
- SWR global fetcher intercepts `401` responses → triggers refresh →
  retries the original request once. If refresh also returns 401 → redirect
  to login.
- Refresh calls are deduplicated: a single in-flight refresh promise is shared
  across concurrent 401s. First caller creates the promise; subsequent callers
  await the same one. Prevents concurrent refresh races.
- On logout: clear in-memory access token, call logout endpoint to invalidate
  refresh token server-side, redirect to login.
- On page load: attempt silent refresh before rendering protected routes. If
  refresh fails → redirect to login.

---

## Navigation structure

```
Login → Dashboard (room cards grid)
         ├─→ Room detail (tabbed)
         │     ├─ Overview tab    (current state + control panel, side-by-side)
         │     ├─ History tab     (stacked climate charts + window selector)
         │     ├─ Schedules tab   (schedule list, inline accordion, period modal)
         │     └─ Devices tab     (full management, not read-only)
         └─→ Devices page         (flat list, inline assignment dropdown)
```

Persistent top nav on all authenticated screens. Max depth: three clicks from
login to any piece of data.

No single-room redirect on login — always land on Dashboard regardless of
room count.

---

## Top nav

56px sticky header. Logo/wordmark left. Dashboard and Devices nav links with
active underline. User menu far right: avatar + mono email + chevron trigger.

User menu dropdown (240px):
- "Signed in as [email]" header
- Account settings (opens Settings modal)
- Divider
- Log out
- Divider
- Delete account (danger color)

---

## Login

Centered card, 360px wide. Email input, password input, Sign in button.
Mono dev-hint footer (hide in production — use neutral seed credentials
such as `operator@local.dev`, not project-specific names).
Nothing else on the page — no subtitle, no extra branding.

---

## Dashboard

Header: "Rooms" title + room count + Add room button (right).
Responsive grid: `repeat(auto-fill, minmax(280px, 1fr))`.

Each **RoomCard** shows:
- Room name + mode badge (header row)
- Temperature readout (heat tint) + humidity readout (cool tint)
- Divider
- Badge row: Heater on/off, Humidifier on/off, control source badge
- Badges only shown for capabilities the room actually has

Control source badge label mapping:
- `manual_override` → "Hold active"
- `schedule` → "Schedule"
- `grace_period` → "Grace period"
- `none` → "Idle"

Cards are clickable — navigate to Room detail. Hover lifts border + shadow.

SWR polls `/rooms` and `/rooms/:id/climate` every **30 seconds**.

---

## Room detail shell

- Back link to Dashboard
- Room name in header + inline edit icon (opens Edit room name modal)
- Kebab menu (⋮) in header: Edit name, Delete room
- Mode badge + source badge in header
- Tab strip: Overview | History | Schedules | Devices

---

## Room detail — Overview tab

Two-column grid. Both panels visible simultaneously — user watches readings
while adjusting controls.

### Current state card (left)

- Temperature readout with `last updated Xm ago` mono meta beneath
- Humidity readout with `last updated Xm ago` mono meta beneath
- "Live" indicator (green dot + "live" mono label top-right of card)
- Divider
- Heater state indicator (on/off) — only shown if room has heater
- Humidifier state indicator (on/off) — only shown if room has humidifier
- Active source indicator using the label mapping above
- Current targets from most recent control log: target temp and target hum,
  only shown for capabilities the room has

SWR polls `/rooms/:id/climate` every **30 seconds**.

### Control panel card (right)

Four sections top to bottom. Full-width rows — temperature and humidity are
never split left/right.

**Schedule section**

Label "Schedule". Container with subtle border and surface-2 background.

- Colored status dot + schedule name when a schedule is active
- "Overridden by Hold" mono meta to the right when Hold is active
- When Hold is active: entire section (dot + name) fades to muted opacity
  (visual subordination — communicates "bypassed" without text)
- When no active schedule: grey dot + "None" in muted color

**Mode section**

Label "Mode". Segmented control: OFF | AUTO.

- Selecting OFF collapses the capability rows entirely (hard collapse,
  no animation)
- Selecting AUTO reveals the capability rows

**Capability rows** (visible only when Mode is AUTO)

Always renders both Temperature and Humidity rows regardless of room
capability — layout consistency is important.

Each row: toggle dot (left) + capability label + input area (right).

- **Room has capability, toggle ON:** input field (left-aligned value, unit
  suffix immediately after, e.g. `22.0 °C`) + clickable deadband pill
  (`±0.5°C`) to the right. Clicking the deadband pill opens the Tolerances
  modal. Pill shows hover state (filled background) to indicate clickability.
- **Room has capability, toggle OFF:** input hidden, "Not regulating" in
  muted mono text. Toggle dot is interactive — clicking enables regulation.
- **Room does not have capability:** entire row greyed out. Toggle dot is
  non-interactive. Hovering the greyed toggle shows a tooltip: "No
  [temperature/humidity] sensor or actuator in this room." Native tooltip
  fallback via `title` attribute.

For rooms with only one capability, the toggle still renders for both rows —
consistency over compactness. The absent row is clearly greyed and non-interactive.

Inputs are pre-filled from saved desired state (not from current control log
readings). The user is editing their saved preferences, not the live state.

**Hold section**

Label "Hold". Segmented control: Off | On.

- When On is selected: duration chips appear inline — 30 min, 1h, 2h, 4h,
  Indefinite. One chip is active at a time.
- Hold is disabled (greyed, non-interactive) when Mode is AUTO and all
  capability toggles are off. Hint text: "Enable a capability to hold."
- Hold in OFF mode is always enabled — holding OFF stops regulation
  regardless of schedule.

**Card footer**

Revert (ghost, left) + Apply (primary, right), separated by top divider.

- Apply: saves desired state, activates Hold if On is selected. Client
  translates duration chip to `manual_override_until` timestamp
  (Indefinite → null with `manual_active = true`, timed → now + duration).
- Revert: resets all draft inputs to last saved desired state.

---

## Room detail — History tab

Two stacked charts — one for temperature, one for humidity. Both share the
same x-axis time scale and are controlled by the same window selector.

**Window selector** — Segmented: 1h | 6h | 24h | 7d. Default: 24h.
Must not stretch — wrap in `align-items: flex-start`. Changing the window
triggers an immediate re-fetch of both charts.

**Temperature chart**
- Primary line: average temperature, left y-axis, heat tint
- Dashed lines: target temp ± deadband — only shown when data has
  non-null targets. Default on, toggleable.
- Background fill: heater duty cycle as opacity — warm tint, opacity
  proportional to duty cycle fraction (0.0–1.0). Absent when null.
  Default on, toggleable.
- Per-chart toggle controls top-right: Target band on/off, Duty cycle on/off

**Humidity chart**
- Primary line: average humidity, left y-axis, cool tint
- Dashed lines: target humidity ± deadband — same toggle behaviour
- Background fill: humidifier duty cycle as opacity — cool tint,
  proportional opacity. Same toggle behaviour.

**Shared chart behaviour**
- Null values render as **gaps** in the line — `connectNulls={false}`.
  A gap means no data for that bucket. Never connect across gaps.
- X-axis tick plan (generateTimeTicks algorithm):
  - 1h window → 15 min ticks
  - 6h window → 1h ticks
  - 24h window → 6h ticks
  - 7d window → 24h ticks
- Ticks formatted in 12-hour AM/PM style in user local time (from timezone
  setting). 7d view adds weekday prefix.
- Y-axis: min/max labels + unit
- Charts are architecture-ready for scroll-to-zoom (zoom interaction is a
  future feature — do not implement in Phase 6 but structure the component
  so zoom can be added without a rewrite)

**Polling** — SWR polls `/rooms/:id/climate/history?window=<w>` every
**60 seconds** + `revalidateOnFocus: true`. Full re-fetch on each poll —
avoids mixing raw and bucketed data at the chart's right edge.

---

## Room detail — Schedules tab

- Add schedule button (top right)
- Empty state card when no schedules exist
- List of expandable schedule cards

**Schedule card (collapsed)**
- Chevron + schedule name + period count + Active/Inactive badge
- Activate / Deactivate button
- Kebab menu (⋮): Edit name, Delete schedule

**Schedule card (expanded)**
- `--cc-surface-2` background
- Period table: Days | Start | End | Target temp | Target hum | Actions
  - Days: 7 single-letter mono chips, filled for active days (M T W T F S S)
  - Target temp: heat mono — only shown if room has temp capability
  - Target hum: cool mono — only shown if room has humidity capability
  - Actions column: Edit icon + Delete icon
- Period delete: **inline confirm** — row transforms to show
  "Delete this period? [Yes] [Cancel]" with danger tint background.
  No separate modal.
- "Add period" ghost button at bottom of expanded card

---

## Room detail — Devices tab

Full management surface — not read-only.

- "Manage all devices →" link at top
- Register device button (top right)
- Table: Name | hw_id | Sensors | Actuators | Actions
- Actions column per row: Edit icon + Delete icon

---

## Devices page

- Header: "Devices" title + Register device button
- Table: Name | hw_id | Capabilities | Room | Actions
- Room column: inline select dropdown (all rooms + Unassigned)
- Actions column: Edit icon + Delete icon

---

## Modals — complete inventory

All modals share the Modal primitive: fixed overlay, centered card, header
(title + subtitle + close X), body, optional footer with right-aligned actions
on `--cc-surface-2`. Click-outside and X both close.

Button ordering convention throughout: Cancel (ghost, left) | Primary action
(right). Destructive primary actions use danger color.

| Modal | Trigger | Fields | Footer |
|---|---|---|---|
| Add room | Dashboard Add room | Name | Cancel / Save |
| Edit room name | Room header edit icon or kebab | Name (pre-filled) | Cancel / Save |
| Delete room | Room header kebab → Delete | "This will unassign all devices and delete all schedules for [name]. This cannot be undone." | Cancel / Delete (danger) |
| Register device | Devices tab or Devices page | hw_id (mono), display name, room (select, optional) | Cancel / Register |
| Edit device | Device row edit icon | Display name (pre-filled), room assignment (pre-filled, includes Unassigned) | Cancel / Save |
| Delete device | Device row delete icon | "This will permanently remove [name] from the system." | Cancel / Delete (danger) |
| Tolerances | Deadband pill click in control panel | Deadband temp input + live helper ("Heater turns on below X°C, off above Y°C"), deadband hum input + live helper — capability-aware | Cancel / Save |
| Add schedule | Schedules tab Add schedule | Name | Cancel / Save |
| Edit schedule | Schedule card kebab → Edit | Name (pre-filled) | Cancel / Save |
| Delete schedule | Schedule card kebab → Delete | "This will delete [name] and all its periods." | Cancel / Delete (danger) |
| Add / Edit period | "Add period" button or period edit icon | See period modal detail below | Cancel / Save |
| Account settings | User menu → Account settings | Timezone (IANA select), helper: "Used for schedule display and evaluation" | Cancel / Save |
| Delete account | User menu → Delete account | "All your data will be permanently deleted." Type email to confirm — enables Delete only when value matches | Cancel / Delete (danger, disabled until match) |

### Add / Edit period modal

640px wide. Header includes a picker-mode toggle: Clock | Timeline.

**Form fields:**
- Days picker: 7 toggle chips (M T W T F S S), at least one required
- Start time + End time (see time picker modes below)
- Target temperature: number input with °C suffix — only if room has
  temp capability
- Target humidity: number input with % suffix — only if room has
  humidity capability
- At least one target required. Validation: "Select at least one day",
  "At least one target must be set."

End time must be strictly greater than start time. Midnight-crossing periods
are not supported (consistent with backend constraint).

Save disabled when validation fails.

**Clock mode (default):**

Two time fields (mono pill buttons) showing selected times. Clicking a field
opens a ClockPicker **popover** anchored below the field (not an inline
stepper — the rest of the form remains visible).

ClockPicker (280px card):
- Header display: large mono HH:MM with hour/minute buttons + AM/PM toggle
- Clock face: circular SVG, tap/drag to select
- Hour step: 1–12 on circle, tap selects and auto-advances to minute step
- Minute step: majors at 0/15/30/45 labeled, minors as dots between
- Step indicator: two dots below the clock, one filled for current step
- Footer: Cancel + Confirm

**Timeline mode:**

Entered via header toggle. Two time fields collapse into a "Time range"
display strip.

Default (single-day) view:
- Horizontal 24-hour band, full-width
- Two draggable knobs — start and end. Highlighted region between them.
- Existing periods filtered by currently selected days (Option B) rendered
  as static colored blocks on the band
- Dragging updates time fields in real time, snapping to 15-min increments
- Hour scale below band at 3-hour intervals
- End clamped to > start + 15 min

Week view (entered via toggle button):
- 7 rows, one per day (Mon–Sun), each row is a mini 24-hour band
- Existing periods shown per day as colored blocks
- Current period being edited shown as a heat-tinted preview block on
  selected days only
- Not draggable in week view — drag in single-day view, preview here
- Footer hint: "Drag the handles in single-day view to adjust the time range."

---

## Desired state — backend change required

The current `desired_states` schema conflates manual control targets with
mode. This causes targets to be lost when mode is set to OFF.

Required schema change before Overview tab implementation:

```sql
-- Remove: mode column (OFF/AUTO) as the primary control mechanism
-- Add:
manual_mode           TEXT NOT NULL DEFAULT 'AUTO'  -- 'AUTO' | 'OFF', what gets held
manual_active         BOOLEAN NOT NULL DEFAULT false
-- Existing manual_override_until stays, semantics unchanged for timed holds
-- Existing target_temp, target_hum stay — no longer nulled on mode change
```

Targets persist independently of whether Hold is active or what manual_mode
is set to. A user can have saved targets and still Hold to OFF.

The control loop's `resolveEffectiveState` derives mode from `manual_active`
+ expiry check rather than reading mode directly.

`PUT /api/v1/users/me` endpoint also required (not yet implemented) for
timezone support from the Account settings modal.

---

## Users endpoint — backend change required

`PUT /api/v1/users/me` — update user settings. Minimum field: `timezone`
(IANA timezone string). Required for Account settings modal and for correct
schedule period display in user local time.

---

## Capability-aware rendering

The client uses `GET /rooms/:id` sensor/actuator list to determine which
controls and indicators to render. This applies to:

- Control panel capability rows (temp/humidity inputs and toggles)
- Tolerances modal (which fields appear)
- History charts (which lines and fills to render)
- Schedule period table (which target columns appear)
- Add/edit period modal (which target inputs appear)
- Dashboard RoomCard badges (which actuator states to show)
- Current state card indicators

A room with no humidifier never shows humidity controls. A room with a
humidifier but no recent reading shows the control (structural capability
exists) but the readout shows `—` (transient null). This distinction between
structural nulls and transient nulls must be preserved throughout.

---

## Key decisions and rationale

**Access token in React context, not localStorage** — localStorage is readable
by any JavaScript on the page (XSS risk). React context keeps the token in
memory, lost on refresh, with the httpOnly cookie handling silent
re-authentication transparently.

**Two stacked history charts, not one combined** — separating temperature and
humidity onto independent charts eliminates dual-axis clutter, allows both
duty cycle fills to be displayed without visual interference, and keeps each
chart's y-axis meaningful and unambiguous.

**Duty cycle as opacity fill, not a line** — the background fill approach
shows duty cycle as contextual information without requiring a third axis.
Opacity proportional to the 0.0–1.0 fraction preserves nuance (partial
on/off buckets are visible) without cluttering the primary data lines.

**`connectNulls={false}` on all chart lines** — gaps are meaningful. A gap
means no data for that bucket (device offline or no readings). Connecting
across it would imply continuous data that does not exist.

**Hold as the manual control mechanism, not a mode toggle** — "Hold" is
established thermostat language. It implies temporary, it implies taking
manual control, and it does not require explanatory text. The word "override"
is deliberately absent from the UI.

**Schedule section visual subordination on Hold** — fading the schedule
section when Hold is active communicates "this is being bypassed" without
any text. Re-brightening when Hold is deactivated communicates "schedule
is running again." No labels needed.

**Capability rows always render both, absent rows greyed** — layout
consistency matters. The user always knows where to look. A greyed row
with a tooltip communicates "this capability could exist here, your room
just does not have it yet" — which is honest and leaves the door open for
future device additions.

**Timeline picker uses Option B (filter by selected days)** — showing
existing periods for the days currently selected in the day picker gives
the user relevant context for conflict detection without showing irrelevant
data from days they are not editing.

**Devices tab is full management, not read-only** — requiring the user to
navigate to a separate page to edit a device they can already see is
unnecessary friction. The room-scoped devices tab and the global devices
page both offer full edit/delete.

**Inline confirm for period delete, modal for everything else** — periods
are the least consequential deletable entity (a schedule can have many,
they are easy to recreate). An inline confirm is less disruptive than a
modal for low-stakes destructive actions. Room and schedule deletion use
modals because the consequences are broader.

**No staleness warning in Phase 6** — the control log timestamp does not
reflect device health (the control loop runs on a fixed interval regardless
of device state). Proper device staleness detection requires `last_reading_at`
per measurement type on the climate response, which is a backend change.
Deferred to future-features.md.

**History chart zoom is architecture-ready but not implemented** — the
window state and tick algorithm are structured to support scroll-to-zoom
in a future iteration. The `generateTimeTicks` plan-based approach extends
naturally to finer zoom levels by adding more entries to the tick plan table.