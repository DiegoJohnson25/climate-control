# Web Client — Architecture Reference

React SPA served as static files by NGINX. Consumes the API Server REST API via NGINX proxy. All API calls use `credentials: "include"` so the httpOnly refresh cookie is sent automatically.

**Status:** Phase 6 — in progress.

---

## Stack

| Technology | Purpose |
|---|---|
| React 19 | UI component framework |
| Vite | Build tooling and dev server |
| SWR | Server state — data fetching, polling, cache invalidation |
| shadcn/ui + Tailwind | Component library and utility CSS |
| Recharts | Time series charts for climate history |
| React Router | Client-side routing |

No TypeScript. No Redux. No TanStack Query.

---

## Auth implementation

Access token stored in React context (in-memory) — not localStorage, not sessionStorage. Lost on page refresh — the httpOnly refresh cookie handles silent re-authentication transparently.

**Login flow:** `POST /auth/login` → access token stored in context → redirect to Dashboard.

**Page load:** silent `POST /auth/refresh` attempted before rendering protected routes. Success → token stored, app renders. Failure → redirect to login.

**401 intercept:** SWR global fetcher intercepts 401 responses → triggers refresh → retries the original request once. If retry also 401s → redirect to login.

**Refresh deduplication:** a single in-flight refresh promise is shared across concurrent 401s. If two SWR hooks 401 simultaneously, only one refresh call fires — both hooks await the same promise. Prevents concurrent refresh races.

**Logout:** clear access token from context → `POST /auth/logout` to invalidate refresh token server-side → redirect to login.

---

## Navigation structure

```
Login
└── Dashboard (room cards grid)
      ├── Room detail (tabbed)
      │     ├── Overview     current state card + control panel
      │     ├── History      stacked climate charts, window selector
      │     ├── Schedules    schedule list, inline period accordion, period modal
      │     └── Devices      full management
      └── Devices page       flat device list, inline room assignment
```

Persistent top nav on all authenticated screens. Maximum depth: three clicks from login to any piece of data. Always lands on Dashboard on login — no single-room redirect.

---

## SWR polling intervals

| View | Endpoint | Interval |
|---|---|---|
| Dashboard | `GET /rooms` | 30s |
| Dashboard | `GET /rooms/:id/climate` (per room) | 30s |
| Overview tab | `GET /rooms/:id/climate` | 30s |
| History tab | `GET /rooms/:id/climate/history` | 60s + revalidate on focus |

---

## Capability-aware rendering

The client uses `GET /rooms/:id` sensor and actuator lists to determine which controls and indicators to render. Distinguishes structural nulls (device does not exist) from transient nulls (device exists, no recent reading):

- A room with no humidifier never shows humidity controls.
- A room with a humidifier but no recent reading shows the control, but the readout shows `—`.

This applies throughout: dashboard badges, control panel rows, history charts, schedule period columns, period modal inputs, tolerances modal fields.

---

## Control source label mapping

The `control_source` field from the API maps to user-facing labels. The word "override" does not appear in the UI — "Hold" is the established thermostat convention.

| API value | UI label |
|---|---|
| `manual_override` | Hold active |
| `schedule` | Schedule |
| `grace_period` | Grace period |
| `none` | Idle |

---

## History charts

Two stacked Recharts charts — temperature above, humidity below. Separated to avoid dual-axis clutter and to give each chart's y-axis unambiguous meaning.

Each chart renders:
- Primary line — `avg_temp` or `avg_hum`
- Dashed target ± deadband overlay (toggleable)
- Background opacity fill for actuator duty cycle — warm tint for heater, cool tint for humidifier, opacity proportional to 0.0–1.0 duty fraction (toggleable)
- `connectNulls={false}` — gaps mean no data for that bucket, not interpolated

Time tick intervals by window: 1h → 15min, 6h → 1h, 24h → 6h, 7d → 24h.

---

## Key design decisions

**Access token in React context, not localStorage** — localStorage is readable by any JavaScript on the page. React context keeps the token in memory, lost on page refresh, with the httpOnly cookie handling silent re-authentication transparently.

**Two stacked history charts, not one combined** — separating temperature and humidity eliminates dual-axis clutter, allows both duty cycle fills to be displayed without visual interference, and keeps each chart's y-axis meaningful.

**Duty cycle as opacity fill, not a line** — background fill shows duty cycle as contextual information without a third axis. Opacity proportional to the 0.0–1.0 fraction preserves nuance — partial on/off buckets are visible without cluttering the primary data lines.

**Hold as the manual control mechanism** — "Hold" is established thermostat language. It implies temporary, it implies taking manual control, and it requires no explanation. The word "override" is deliberately absent from the UI.

**Capability rows always render, absent capabilities greyed** — layout consistency matters. A greyed row with a tooltip communicates that the capability could exist but the room does not currently have the hardware — which is accurate and leaves the door open for future device additions.

---

## Pending backend changes (required before Phase 6b)

**Desired state schema change** — add `manual_active BOOLEAN` and `manual_mode TEXT` columns. Targets (`target_temp`, `target_hum`) must persist independently of whether Hold is active. A user can have saved targets while holding to OFF. The control loop derives mode from `manual_active` + expiry check rather than reading a `mode` column directly.

**`PUT /api/v1/users/me`** — update user timezone (IANA string). Required for Account settings modal and correct schedule period display in user local time.