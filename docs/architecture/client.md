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

**No TypeScript** — low React familiarity, offloading React work to Claude, thin
UI over a clean REST API. Not worth the overhead at this scope.

**Why SWR not React Query** — simpler mental model. React Query's mutation
machinery and advanced cache control are not needed here.

---

## Auth

- JWT access token stored **in memory** (JavaScript variable) — not localStorage,
  not sessionStorage. Lost on page refresh — user re-authenticates via refresh token.
- Refresh token stored in **httpOnly cookie** — XSS safe, sent automatically by
  the browser on requests to the same origin.
- SWR intercepts `401` responses → triggers refresh → retries the original request.
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
         │     ├─ History tab     (climate chart + window selector)
         │     ├─ Schedules tab   (schedule list, inline accordion, period modal)
         │     └─ Devices tab     (read-only, links to Devices page)
         └─→ Devices page         (flat list, inline assignment dropdown)
```

**Persistent top nav** — Dashboard and Devices accessible from anywhere.
Room detail is always one click from Dashboard. Max depth: three clicks from
login to any piece of data.

**No single-room redirect on login** — consistent behaviour regardless of room
count. Always land on Dashboard.

---

## Dashboard

Room cards grid. Each card shows:
- Room name
- Current temperature and humidity (from `/climate` endpoint)
- Current mode and control source
- Heater / humidifier state indicators

Cards are clickable — navigate to Room detail. SWR polls `/rooms` and
`/rooms/:id/climate` every **30 seconds**.

---

## Room detail — Overview tab

Side-by-side panel layout. Both panels visible simultaneously — user watches
readings while adjusting controls. No modal for control input.

**Left panel — current state:**
- Temperature reading with `last_updated` timestamp
- Humidity reading with `last_updated` timestamp
- Heater state indicator (on/off, or greyed out if room has no heater)
- Humidifier state indicator (on/off, or greyed out if room has no humidifier)
- Control source badge (`manual_override` / `schedule` / `grace_period` / `none`)

**Right panel — control panel:**
- Mode selector: OFF / AUTO toggle
- Target temperature input (shown only when mode is AUTO and room has temp capability)
- Target humidity input (shown only when mode is AUTO and room has humidity capability)
- Manual override toggle with duration selector

**Capability awareness** — the client uses the room's sensor/actuator list from
`GET /rooms/:id` to determine which controls and indicators to render. A room
with no humidifier never shows a humidifier indicator or humidity target input.
This also distinguishes structural nulls (no humidifier) from transient nulls
(humidifier exists, no recent reading) in the climate response.

**Polling** — SWR polls `/rooms/:id/climate` every **30 seconds**.

---

## Room detail — History tab

**Chart:** Recharts `LineChart` or `ComposedChart`. Lines rendered per
measurement type. Null values render as **gaps in the line** — `connectNulls={false}`
(Recharts default). A gap means the device was offline or readings were stale.
Do not connect across gaps.

**Lines rendered:**
- Average temperature (`avg_temp`)
- Average humidity (`avg_hum`)
- Target temperature band overlay (`target_temp ± deadband_temp`) — dashed lines,
  only shown when data has non-null targets
- Target humidity band overlay (`target_hum ± deadband_hum`) — dashed lines,
  only shown when data has non-null targets
- Heater duty cycle (`heater_duty`) — secondary axis or separate panel, 0.0–1.0
- Humidifier duty cycle (`humidifier_duty`) — secondary axis or separate panel, 0.0–1.0

Omit lines for capabilities the room doesn't have (use room capability data from
`GET /rooms/:id`).

**Window selector** — four buttons: `1h`, `6h`, `24h`, `7d`. Default: `24h`.
Selecting a window triggers an immediate re-fetch.

**Polling** — SWR polls `/rooms/:id/climate/history?window=<w>` every **60 seconds**.
Also re-fetches on tab focus (`revalidateOnFocus: true`).

**Data shape from API:**
```json
{
  "window": "6h",
  "bucket_seconds": 180,
  "points": [
    {
      "time": "2026-04-23T08:00:00Z",
      "avg_temp": 21.8,
      "avg_hum": 57.2,
      "heater_duty": 0.6,
      "humidifier_duty": null,
      "target_temp": 22.0,
      "target_hum": null,
      "deadband_temp": 0.5,
      "deadband_hum": null
    }
  ]
}
```

`bucket_seconds` is available for Recharts tick formatting if needed. Null fields
mean either the room lacks that capability (structural) or no data that bucket
(transient) — the client uses room capability data to distinguish.

---

## Room detail — Schedules tab

**Schedule list** — each schedule shows name, active/inactive status, and an
activate/deactivate toggle.

**Period list** — inline accordion expansion within the schedule row. Clicking
a schedule expands to show its periods. No navigation — stays on the same tab.

**Period editing** — modal form. Small form (days, start time, end time, targets),
modal is appropriate. No navigation needed.

**No `activatable` field yet** — the API does not currently precompute whether
a schedule can be activated. The client discovers non-activatable schedules on
attempted activation (422 response). Greying out the activate button is a future
enhancement.

---

## Room detail — Devices tab

Read-only. Shows devices assigned to this room — name, hw_id, sensors, actuators.
Link to Devices page for management. No assignment controls here.

---

## Devices page

Flat list of all user devices. Each device row shows:
- Device name, hw_id
- Current room assignment (or "Unassigned")
- Inline assignment dropdown — select a room or unassign

This is the only management surface for device assignment. Room detail Devices
tab is read-only and links here.

---

## Key decisions and rationale

**Access token in memory, not localStorage** — localStorage is readable by any
JavaScript on the page (XSS risk). In-memory is lost on refresh but the httpOnly
refresh cookie handles silent re-authentication transparently.

**History chart re-fetches full array on 60s poll** — appending a partial bucket
to an aggregated chart is inconsistent (mixing raw and bucketed data at the right
edge). Full re-fetch avoids this. Payload is ~2–4KB, acceptable on a 60s cadence.

**`connectNulls={false}` on all chart lines** — gaps are meaningful on a monitoring
dashboard. A gap means the device was offline. Connecting across it would imply
continuous data that doesn't exist.

**Target band overlay on history chart** — `target ± deadband` rendered as dashed
lines gives immediate visual context for whether the control loop was achieving
its goal. Only shown when the data has non-null targets.

**Capability-aware rendering** — client checks `GET /rooms/:id` sensor/actuator
list before rendering indicators and controls. Avoids showing a humidifier
indicator for a temp-only room. Also distinguishes "no humidifier" from "humidifier
offline" when interpreting null values in `/climate` response.

**No redirect to single room on login** — consistent UX regardless of how many
rooms a user has. Dashboard is always the landing page.

**Side-by-side overview layout, no modal** — user needs to watch live readings
while adjusting targets. A modal would hide the readings panel. Side-by-side
keeps both visible simultaneously.

**Schedule periods in inline accordion, not nested route** — periods are a
sub-list of the schedule, not a separate page. Accordion expansion within the
tab is the right scope for this amount of content.
