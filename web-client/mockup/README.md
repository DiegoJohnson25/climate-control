# Climate Control — Mockup Reference

Canonical design reference for Phase 6 (web client) implementation. These
files are a mockup, not a drop-in production codebase.

Use the JSX here to understand **visual structure, states, and class usage**.
Re-implement against the target stack (React 19 + Vite + SWR + shadcn/ui +
Tailwind + Recharts + React Router) — do not copy the `window.*` global
pattern or the raw SVG chart. Route all color, spacing, and typography
through the CSS tokens via Tailwind's `theme.extend`.

`docs/architecture/client.md` is the functional source of truth (polling
cadences, auth flow, backend contract, API endpoints). This README is the
visual / component source of truth.

Open `index.html` or `states.html` via a static server (`python -m
http.server` at this folder) to view the live mockup.

---

## Folder structure

```
docs/mockup/
├── README.md                 ← this file
├── index.html                ← app entry point (live dashboard)
├── components/
│   ├── ui.jsx                ← shared primitives: Button, Input, Modal, Readout…
│   ├── icons.jsx             ← window.Icon — Lucide-style inline SVGs
│   ├── shell.jsx             ← TopNav, Login
│   ├── app.jsx               ← App — router, modal dispatch, live state actions
│   ├── dashboard.jsx         ← Dashboard, RoomCard, SourceBadge, ModeBadge
│   ├── room-detail.jsx       ← RoomDetail (tab container + header)
│   ├── overview-tab.jsx      ← OverviewTab, CurrentStateCard, ControlPanel
│   ├── history-tab.jsx       ← HistoryTab, ChartPanel, Chart, generateTimeTicks
│   ├── schedules-tab.jsx     ← SchedulesTab, ScheduleCard, PeriodRow, Empty…
│   ├── period-modal.jsx      ← PeriodModal, ClockPicker, DayTimeline
│   ├── modals-simple.jsx     ← all CRUD + tolerances + account modals
│   └── devices.jsx           ← DevicesTab (room-scoped), DevicesPage (global), CapChip
├── styles/
│   ├── colors_and_type.css   ← base design tokens (colors, type, motion, layout)
│   └── styles.css            ← component-level CSS; @imports colors_and_type.css
└── reference/
    ├── states.jsx            ← states gallery entry point
    ├── states.html           ← StatesGallery — every interaction state, one page
    └── data.jsx              ← window.CCData — seed rooms/devices/schedules/history
```

### Corrections vs the v2 handoff

| # | Correction | Where applied |
|---|---|---|
| 1 | 12-hour AM/PM time format throughout (`2pm`, `12am`, `9:30am`) | `ui.jsx` helpers `fmtTime12` / `fmtMin12` / `fmtTick12`; `history-tab.jsx` tick formatter; `schedules-tab.jsx` period table + inline-confirm; `period-modal.jsx` time field buttons, timeline strip, timeline band label, existing-block labels, hour scale under `DayTimeline` |
| 2 | Login page: removed "Climate Control" brand lockup above card; seeded email is `operator@local.dev`; inner subtitle neutralized to just "Sign in" | `shell.jsx` `Login` |
| 3 | Delete account modal body → "All your data will be permanently deleted. This cannot be undone." | `modals-simple.jsx` `DeleteAccountModal` |
| 4 | `cc-pulse` 600ms live-value fade ported from v1 into the `Readout` component; triggers on any non-null value change | `ui.jsx` `Readout`; animation itself already defined in `colors_and_type.css` |
| 5 | `--cc-hold` changed from danger-red `#DC2626` to warm amber `#A16207`; `--cc-grace` changed from purple `#7C3AED` to cool slate `#64748B` | `styles.css` |
| 6 | `--cc-fg-invert` verified present in `colors_and_type.css` (light-mode value `#FAFAF9`, dark-mode override `#14130F`) | unchanged — already defined |
| 7 | All modals audited: `Cancel (ghost, left) | Primary (right)`. Period-row inline delete confirm intentionally keeps destructive-on-left (`Yes, delete` then `Cancel`) per brief | no changes needed — already compliant |
| 8 | `states.html` / `states.jsx` kept as-is (reference artifact) | unchanged |
| 9 | Room / device / schedule seed copy neutralized (Living Room, Bedroom, Nursery, Basement, Office, Storage Closet) | `data.jsx`; placeholders in `modals-simple.jsx`; login email in `shell.jsx` |

**Caveat on `states.jsx`:** since it is kept as-is per the instruction to not
modify reference artifacts, it still contains one grow-room string —
`ScheduleSectionDemo activeName="Vegetative stage"`. The live `index.html`
does not render this; it only appears in the standalone states gallery.
Leave in place or edit separately if the artifact is ever revised.

---

## Component inventory

### `icons.jsx` — `window.Icon`

Factory function per icon: `window.Icon.thermometer(size)`. All icons use
`fill: none; stroke: currentColor; strokeWidth: 1.5`. Replace with
`lucide-react` imports in production.

| Key | Icon |
|---|---|
| `thermometer` | brand mark + temperature label |
| `droplets` | humidity label |
| `flame` | heater indicator |
| `power` | control affordance |
| `cpu` | device type |
| `chevronLeft` / `chevronRight` / `chevronDown` / `chevronUp` | nav and accordion |
| `plus` | add actions |
| `x` | close |
| `check` | confirmation |
| `pencil` | edit |
| `trash` | delete |
| `kebab` | ⋮ more-actions menu |
| `clock` | time fields in period modal |
| `calendar` | empty-state schedules icon |
| `grid` / `rows` | timeline layout toggles |
| `settings` | account settings |
| `logOut` | logout |
| `wifi` | connectivity |
| `info` / `alert` | informational / warning contexts |
| `lock` | auth contexts |
| `pause` / `play` | schedule activate/deactivate |
| `dotsGrid` | decorative |

### `ui.jsx` — shared primitives + helpers

| Name | Role |
|---|---|
| `Button({variant, size, icon, iconRight, children})` | primary/secondary/ghost/danger + md/sm/lg sizes |
| `IconBtn({danger, active, title, children})` | 28px square icon button; used for edit/delete/close/kebab |
| `Input({mono})` | 32px text input; `mono` swaps family to JetBrains Mono with `tabular-nums` |
| `InputUnit({value, onChange, unit, mono, style})` | wrapper with absolute-positioned unit suffix (°C, %) inside the field |
| `Select({value, onChange, mono, children})` | native `<select>` styled to match `cc-input` |
| `Card({children, style})` | `.cc-card` surface |
| `Field({label, hint, children})` | label + control + optional mono hint text |
| `Badge({variant, dot, children})` | heat/cool/ok/warn/err/info + optional leading dot |
| `Readout({value, unit, tone, size, decimals})` | hero numeric. Applies `cc-readout--live` for 600ms on every non-null value change — the live pulse. `size`: `lg` (48px) \| `sm` (24px). `tone`: `heat` \| `cool` \| undefined |
| `Segmented({value, onChange, options, disabled})` | inline-flex pill toggle. **Never stretches** — must be wrapped in `align-items: flex-start` if needed |
| `ToggleDot({on, onClick, disabled, title})` | 18px round on/off dot used on capability rows |
| `Chip({on, onClick, children, disabled})` | mono uppercase pill — used for Hold duration selector |
| `DayChip({label, on, onClick, readonly})` | 26×26 (or 20×20 readonly) day-of-week picker chip |
| `DeadbandPill({value, unit, onClick})` | dashed pill `±0.5°C`; clickable, opens Tolerances modal |
| `Tooltip({text, children, forceShow})` | portal-rendered hover tooltip |
| `KebabMenu({items, align})` | ⋮ trigger + dropdown popover; items are `{label, onClick, danger, divider}` |
| `Modal({open, onClose, title, subtitle, children, footer, width})` | shared modal primitive; Esc + backdrop close |
| `timeAgo(iso)` | `Xs/Xm/Xh/Xd ago` |
| `fmtTime12("14:30")` | `"2:30pm"` — formats internal HH:MM to 12hr display |
| `fmtMin12(540)` | `"9am"` — formats minutes-of-day to 12hr display |
| `fmtTick12(ts, windowKey, tz)` | history-chart tick label; adds weekday prefix on `7d` |

### `shell.jsx`

| Name | Role |
|---|---|
| `TopNav({page, onNav, account, onOpenAccount, onOpenDeleteAccount, onLogout, forceMenuOpen})` | 56px sticky header. Brand + nav links + user menu. `forceMenuOpen` is for the states gallery |
| `Login({onSignIn})` | centered 360px card. Only the card on the page (plus neutral dev footer below) |

### `dashboard.jsx`

| Name | Role |
|---|---|
| `Dashboard({rooms, onOpenRoom, onAddRoom})` | page: header + responsive grid of room cards |
| `RoomCard({room, onOpen, forceHover})` | clickable card — name + mode badge, temp/hum readouts, capability-aware badge row |
| `SourceBadge({source})` | source-to-label map: `manual_override`→"Hold active", `schedule`→"Schedule", `grace_period`→"Grace period", `none`→"Idle" |
| `ModeBadge({mode})` | `auto`→"AUTO" (default), `off`→"OFF" (warn variant) |

### `room-detail.jsx`

| Name | Role |
|---|---|
| `RoomDetail({room, rooms, schedules, devices, tz, initialTab, onBack, on…, _forceTab})` | container with back link, header (name + edit icon + mode + source + kebab), tab strip, active tab |

### `overview-tab.jsx`

| Name | Role |
|---|---|
| `OverviewTab({room, schedules, onOpenTolerances, onApply, onRevert})` | two-column grid (`1fr 1.15fr`) |
| `CurrentStateCard({room})` | live-indicator header; temp + humidity readouts with last-updated mono meta; heater/humidifier state rows; active source; current targets with inline `±db` |
| `ControlPanel({room, schedules, onApply, onRevert, onOpenTolerances, _forceState})` | Schedule section + Mode segmented + capability rows (when Mode=AUTO) + Hold section + Apply/Revert footer. Holds local draft until Apply |

### `history-tab.jsx`

| Name | Role |
|---|---|
| `HistoryTab({room, tz})` | window selector + timezone readout + stacked ChartPanels per capability |
| `ChartPanel({title, icon, unit, data, color, dutyColor, valueKey, targetKey, dbKey, dutyKey, windowKey, tz, showXAxis})` | card with per-chart toggles (Target band, Duty cycle) wrapping a `Chart` |
| `Chart({…})` | raw SVG line chart with null-as-gap, target-band dashed lines, duty-cycle opacity fill columns, clean time ticks |
| `TinyToggle({on, onClick, label})` | small pill toggle used for the per-chart option switches |
| `generateTimeTicks(windowKey, minT, maxT, tz)` | plan-based tick generator (1h→15min, 6h→1h, 24h→6h, 7d→24h); labels via `fmtTick12` |

### `schedules-tab.jsx`

| Name | Role |
|---|---|
| `SchedulesTab({room, schedules, on…})` | header (count + Add button) + empty state or schedule list |
| `ScheduleCard({schedule, room, forceExpanded, on…, _forceConfirmPeriodId})` | collapsed header row + expanded period table |
| `PeriodRow({period, room, onEdit, onDelete, _forceConfirm})` | table row OR (if `_forceConfirm` / local state) inline danger-tinted "Delete this period? [Yes, delete] [Cancel]" row |
| `EmptyScheduleState({onAdd})` | centered card with calendar icon |

### `devices.jsx`

| Name | Role |
|---|---|
| `DevicesTab({room, devices, onRegister, onEditDevice, onDeleteDevice, onGoToGlobal})` | room-scoped, **full management** (not read-only). Link to global page + Register button + table with actions |
| `DevicesPage({devices, rooms, onRegister, onEditDevice, onDeleteDevice, onChangeRoom})` | global flat list, inline room select on each row |
| `CapChip({kind})` | tiny tinted pill for `temp` / `hum` / `heat` capabilities in device tables |

### `modals-simple.jsx`

All thin Modal wrappers. Footer pattern everywhere: `Cancel (ghost, left) | Primary (right)`.

| Name | Fields | Confirm action |
|---|---|---|
| `AddRoomModal` | Name | Save |
| `EditRoomNameModal` | Name (pre-filled) | Save |
| `DeleteRoomModal` | explanation body | Delete (danger) |
| `RegisterDeviceModal` | hw_id (mono), Display name, Room (optional) | Register |
| `EditDeviceModal` | Display name, Room (includes Unassigned) | Save |
| `DeleteDeviceModal` | explanation body | Delete (danger) |
| `AddScheduleModal` | Name | Save |
| `EditScheduleModal` | Name (pre-filled) | Save |
| `DeleteScheduleModal` | explanation body | Delete (danger) |
| `TolerancesModal` | Temp tolerance (if room has temp), Hum tolerance (if room has hum), live helper text | Save |
| `AccountSettingsModal` | Timezone select + helper | Save |
| `DeleteAccountModal` | plain-language body + type-email-to-confirm gate | Delete account (danger, disabled until match) |

### `period-modal.jsx`

| Name | Role |
|---|---|
| `PeriodModal({open, onClose, mode, room, schedule, period, allPeriods, onSave, _forcePickerMode, _forceClockStep, _forceActiveTimeField, _forceWeekView})` | Add/Edit period. 640px wide. Header has Clock/Timeline toggle. Validates: `atLeastOneDay && atLeastOneTarget && endMin > startMin` |
| `ClockPicker({value, onChange, onConfirm, onCancel, _forceStep})` | 280px popover. Hour face → auto-advance to minute face. AM/PM toggle. Confirm writes back via `onChange`; Cancel closes |
| `DayTimeline({startMin, endMin, onChange, existingBlocks, height, editable, showHours, label})` | 24h horizontal band with draggable start/end knobs, heat-tinted selection region, existing blocks as read-only grey fills, 15-min snap |

### `app.jsx`

| Name | Role |
|---|---|
| `App` | top-level state (rooms, devices, schedules, account, modal), page switcher, action handlers, modal dispatch table. Single active modal at a time via `{ kind, ctx }` shape |

### `states.jsx`

Reference gallery only. Not used at runtime — rendered from `states.html`.

---

## CSS token reference

All tokens defined in `colors_and_type.css` (light-mode `:root` plus
`[data-theme="dark"]` override block). `styles.css` layers in three extra
source-badge accents on top.

### Fonts

| Token | Value / family |
|---|---|
| `--cc-font-sans` | Inter 400/500/600/700 with system fallbacks |
| `--cc-font-mono` | JetBrains Mono 400/500/600 with system fallbacks |

### Type scale (px)

| Token | px |
|---|---|
| `--cc-fs-xs` | 12 |
| `--cc-fs-sm` | 13 |
| `--cc-fs-base` | 14 |
| `--cc-fs-md` | 16 |
| `--cc-fs-lg` | 18 |
| `--cc-fs-xl` | 24 |
| `--cc-fs-2xl` | 32 |
| `--cc-fs-3xl` | 48 |

### Line heights, weights, tracking

| Token | Value |
|---|---|
| `--cc-lh-tight` | 1.1 |
| `--cc-lh-display` | 1.2 |
| `--cc-lh-body` | 1.45 |
| `--cc-lh-mono` | 1.0 |
| `--cc-fw-regular` | 400 |
| `--cc-fw-medium` | 500 |
| `--cc-fw-semibold` | 600 |
| `--cc-fw-bold` | 700 |
| `--cc-tracking-tight` | -0.015em |
| `--cc-tracking-normal` | 0 |
| `--cc-tracking-wide` | 0.04em (use on ALL-CAPS labels) |

### Surface / neutrals (light mode)

| Token | Value | Use |
|---|---|---|
| `--cc-bg` | `#FAFAF9` | app background |
| `--cc-surface` | `#FFFFFF` | cards, modals |
| `--cc-surface-2` | `#F5F5F4` | sunken panels, schedule-card expanded body, modal footer |
| `--cc-overlay` | `rgba(11,13,14,.50)` | backdrops |
| `--cc-neutral-50…900` | `#FAFAF9` → `#14130F` | scrollbar thumb, dark-mode scaffolding |

### Foreground text

| Token | Value | Use |
|---|---|---|
| `--cc-fg` | `#14130F` | primary |
| `--cc-fg-2` | `#3D3B36` | secondary |
| `--cc-fg-3` | `#78766F` | tertiary / meta / helper |
| `--cc-fg-4` | `#A8A6A1` | disabled / placeholder / em-dash for null readouts |
| `--cc-fg-invert` | `#FAFAF9` | text on dark fills — ClockPicker AM/PM active state uses this |

### Borders

| Token | Value |
|---|---|
| `--cc-border` | `#E7E6E4` |
| `--cc-border-strong` | `#D3D2CE` |
| `--cc-divider` | `#F4F4F3` |

### Thermal accents

| Token | Value | Use |
|---|---|---|
| `--cc-heat` | `#D97706` | temperature tone, heater state, target band stroke |
| `--cc-heat-hover` | `#B45309` | reserved for heat-accented hover |
| `--cc-heat-tint` | `rgba(217,119,6,0.10)` | badge/fill tint, period preview band |
| `--cc-heat-border` | `rgba(217,119,6,0.30)` | tinted badge border |
| `--cc-heat-fg` | `#92400E` | text on heat tint |
| `--cc-cool` | `#0891B2` | humidity tone, humidifier state, ring |
| `--cc-cool-hover` | `#0E7490` | reserved |
| `--cc-cool-tint` | `rgba(8,145,178,0.10)` | badge tint |
| `--cc-cool-border` | `rgba(8,145,178,0.30)` | tinted badge border |
| `--cc-cool-fg` | `#155E75` | text on cool tint |

### Semantic

| Token | Value | Use |
|---|---|---|
| `--cc-success` / `-tint` / `-fg` | `#16A34A` / rgba / `#166534` | ok badges, live indicator dot |
| `--cc-warning` / `-tint` / `-fg` | `#CA8A04` / rgba / `#854D0E` | Mode=OFF badge, "hide in production" footer note |
| `--cc-danger` / `-tint` / `-fg` | `#DC2626` / rgba / `#991B1B` | delete buttons, inline-confirm row tint, danger icon color |
| `--cc-info` / `-tint` / `-fg` | `#2563EB` / rgba / `#1E40AF` | schedule-active dot + badge |
| `--cc-primary` / `-hover` / `-fg` | `#14130F` / `#26251F` / `#FAFAF9` | primary buttons, active segmented segment, active day chip, clock hand |
| `--cc-ring` | `#0891B2` | focus ring on inputs; active time-field outline |
| `--cc-ring-offset` | `#FAFAF9` | — |

### Source-state accents (introduced in `styles.css`)

| Token | Value | Use |
|---|---|---|
| `--cc-hold` / `-tint` / `-fg` | `#A16207` / rgba / `#713F12` | "Hold active" source dot (warm amber — distinct from danger and from heat) |
| `--cc-grace` / `-tint` / `-fg` | `#64748B` / rgba / `#334155` | "Grace period" source dot (cool slate — transitional) |

### Spacing (4px base)

`--cc-space-0 .. -24` = `0, 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 96` px.

### Radii

| Token | Value |
|---|---|
| `--cc-radius-sm` | 4px |
| `--cc-radius-md` | 6px |
| `--cc-radius-lg` | 10px |
| `--cc-radius-pill` | 999px |

### Shadows

| Token | Use |
|---|---|
| `--cc-shadow-sm` | card default |
| `--cc-shadow-md` | card hover, tooltip |
| `--cc-shadow-lg` | modal, popover, clock picker |

### Motion

| Token | Value | Use |
|---|---|---|
| `--cc-dur-fast` | 120ms | hover, border-color transitions |
| `--cc-dur-base` | 180ms | most transitions |
| `--cc-dur-slow` | 240ms | emphasis |
| `--cc-dur-fade` | 600ms | live-value pulse |
| `--cc-ease` | `cubic-bezier(0.2, 0, 0.2, 1)` | default |
| `--cc-ease-soft` | `cubic-bezier(0.32, 0.72, 0, 1)` | softer |

### Layout

| Token | Value |
|---|---|
| `--cc-nav-h` | 56px |
| `--cc-max-width` | 1280px |
| `--cc-card-pad` | 20px |
| `--cc-page-pad-x` | 24px |

### Dark mode

`[data-theme="dark"]` overrides surfaces, neutrals, foregrounds, thermals,
primary, ring, and shadows. Structure is identical; values shift. A dark-
mode toggle is not implemented — this is groundwork for a future feature.

---

## CSS class reference

Classes defined in `colors_and_type.css` (under the `SEMANTIC ELEMENT
STYLES` section) and `styles.css`.

### Layout / structure

| Class | Styles | Used by |
|---|---|---|
| `.cc` | page scope: box-sizing, font family/size/color, min-height | `<div className="cc">` wrappers in `App`, `StatesGallery` |
| `.cc-h1`, `.cc-h2`, `.cc-h3`, `.cc-h4` | display headings — 32/24/18/16 px with `--cc-tracking-tight` | page/section titles |
| `.cc-body`, `.cc p` | body text at 14px `--cc-fg-2` | explanatory copy in modals, empty states |
| `.cc-meta` | 12px `--cc-fg-3` | last-updated labels, "for" helper next to Hold duration |
| `.cc-label` | 12px mono uppercase `--cc-tracking-wide` `--cc-fg-3` | field labels (via `Field`) |
| `.cc-code`, `.cc-mono` | mono at 0.92em | inline code/mono snippets |
| `.cc-section-label` | same as `.cc-label` but as a block element | section titles inside the control panel, "N schedules" kicker |

### Surfaces / primitives

| Class | Styles | Used by |
|---|---|---|
| `.cc-card` | white surface, 1px border, 6px radius, `--cc-shadow-sm`, 20px padding | `Card`, RoomCard, every modal content body, schedule cards (custom padding) |
| `.cc-badge`, `.cc-badge--heat/--cool/--ok/--warn/--err/--info` | mono uppercase pill badge with per-variant tinted bg + border + fg | `Badge` |
| `.cc-dot` | 6px currentColor circle | leading dot inside a badge when `dot={true}` |
| `.cc-readout` / `.cc-readout-sm` | hero numeric — 48px / 24px mono `--cc-fw-medium` `--cc-lh-mono` `tabular-nums` `--cc-tracking-tight` | `Readout` |
| `.cc-readout--live` | applies the `cc-pulse` keyframe for 600ms using `--cc-dur-fade` + `--cc-ease` | `Readout` adds on value change |

### Buttons and inputs

| Class | Styles | Used by |
|---|---|---|
| `.cc-btn` | 32px flex button with focus-visible ring | all `Button` variants |
| `.cc-btn--primary/--secondary/--ghost/--danger` | filled / outlined / transparent / red | `Button` variants |
| `.cc-btn--sm` / `--lg` / `--icon` | 26px / 40px / square 32px | `Button size="sm"` etc. |
| `.cc-iconbtn` | 28px square transparent icon button | `IconBtn`; kebab triggers, modal close, row edit/delete |
| `.cc-iconbtn--active` | surface-2 fill + strong border | open kebab menu trigger |
| `.cc-iconbtn--danger` | danger foreground, danger-tint hover | delete row icon |
| `.cc-input` | 32px input with ring focus | `Input`, `Select` |
| `.cc-input--mono` | mono family + tabular-nums on the input | `Input mono`, `InputUnit` |
| `.cc-input-unit` | wrapper for input + absolutely-positioned unit suffix | `InputUnit` |
| `.cc-input-unit .unit` | mono 11px unit label right-inside the field | `InputUnit` |

### Toggles, chips, pills

| Class | Styles | Used by |
|---|---|---|
| `.cc-seg` | inline-flex segmented container, 2px inner padding, border, radius 6 | `Segmented` |
| `.cc-seg button.is-on` | primary fill + primary-fg text | active segment |
| `.cc-seg--disabled` | 0.45 opacity + `pointer-events: none` | disabled segmented (Hold disabled state) |
| `.cc-togdot` | 18px round on/off dot (surface bg, strong border) | `ToggleDot` |
| `.cc-togdot--on` | success-filled with inner white scale-0.5 dot | ToggleDot when on |
| `.cc-togdot--disabled` | neutral-100 fill, not-allowed cursor | capability absent |
| `.cc-chip` | mono uppercase pill with border | `Chip` — Hold duration chips |
| `.cc-chip--on` | primary fill + primary-fg text + primary border | selected duration |
| `.cc-daychip` | 26×26 mono rounded square | `DayChip` — period day picker |
| `.cc-daychip--on` | primary fill + primary-fg text | active day |
| `.cc-daychip--read` | 20×20 smaller readonly variant | period table day chips |
| `.cc-dbpill` | dashed pill, mono 11px `--cc-fg-3` | `DeadbandPill`; hover → surface-2 fill + darker border |
| `.cc-row` | flex row 12px gap, min-height 32 | control panel section rows |
| `.cc-row--disabled` | 0.5 opacity | absent-capability row |
| `.cc-statusdot` | 8px solid circle | Schedule section dot |

### Tables, popovers, overlays

| Class | Styles | Used by |
|---|---|---|
| `.cc-table` | 100% width, 13px, separated rows | schedule/device tables |
| `.cc-table th` | 10×14 padding, mono uppercase 11px `--cc-fg-3` | column headers |
| `.cc-table td` | 10×14 padding, vertical-align middle | cells |
| `.cc-tt` | positioned tooltip — dark neutral-900 surface | `Tooltip` (portal) |
| `.cc-pop` | white popover with strong border + `--cc-shadow-lg` | `KebabMenu` dropdown, TopNav user menu |
| `.cc-pop button` | block full-width left-aligned menu item | popover items |
| `.cc-pop button.danger` | danger text + danger-tint hover | Delete entries |
| `.cc-pop hr` | 1px divider | menu dividers |
| `.cc-modal-bg` | fixed inset 0, overlay rgba(20,19,15,.40) | `Modal` backdrop |
| `.cc-modal` | centered surface card, 460 wide default | `Modal` body |
| `.cc-modal-head` | header row with title + close, 18px padding, divider below | `Modal` header |
| `.cc-modal-body` | 20px padded scrollable body | `Modal` content |
| `.cc-modal-foot` | 14×20 footer, surface-2 bg, right-aligned flex | `Modal` actions |

### Animation

| Class | Styles |
|---|---|
| `@keyframes cc-pulse` | 0% → `--cc-fg`, 18% → `--cc-cool`, 100% → `--cc-fg` |
| `.cc-readout--live` | `animation: cc-pulse var(--cc-dur-fade) var(--cc-ease)` |

---

## Interaction states reference

Every component that has multiple states, the states themselves, and how
they're triggered (prop name, class, or `_force…` demo flag used in
`states.jsx`).

### RoomCard (`dashboard.jsx`)

| State | Trigger |
|---|---|
| Default | `forceHover={false}` or no prop |
| Hover | `forceHover` prop (or natural hover) — lifts border, shadow, 1px translate |
| Capability-aware badges | `room.hasTemp` / `room.hasHum` — each badge only appears if the room has that capability |
| Temp/hum readout absent | `room.tempC == null` / `room.humPct == null` — renders `—` in `--cc-fg-4` |

### Schedule section (`overview-tab.jsx`, inside `ControlPanel`)

| State | Trigger | Visual |
|---|---|---|
| Active schedule, Hold off | `activeScheduleName !== null && !holdActive` | info-colored dot + name in `--cc-fg` |
| Hold active | `holdActive === true` | entire section `opacity: 0.5`, dot grey, "Overridden by Hold" meta on right |
| No schedule | `activeScheduleName == null` | grey dot, "None" in `--cc-fg-4` |

### Mode section (`overview-tab.jsx`)

| State | Trigger |
|---|---|
| AUTO selected | `mode === "auto"` — capability rows render below |
| OFF selected | `mode === "off"` — capability rows are not rendered (hard collapse, no animation) |

### Capability row (`overview-tab.jsx` `CapRow`)

| State | Trigger | Visual |
|---|---|---|
| ON + capability present | `has && on` | green `ToggleDot` + input + clickable `DeadbandPill` |
| OFF + capability present | `has && !on` | grey `ToggleDot` + label + "Not regulating" in `cc-meta`; dot still clickable |
| Capability absent | `!has` | `cc-row cc-row--disabled` + disabled `ToggleDot` wrapped in `Tooltip`; label uses `--cc-fg-4`; right side shows "Not available"; native `title` fallback |

### Hold section (`overview-tab.jsx`)

| State | Trigger | Visual |
|---|---|---|
| Off | `!hold.on` | just the segmented control |
| On | `hold.on && !holdDisabled` | segmented + "for" meta + five duration `Chip`s (30 min / 1h / 2h / 4h / Indefinite) |
| Disabled | `mode === "auto" && !anyCapOn` (computed `holdDisabled`) | `cc-seg cc-seg--disabled` + "Enable a capability to hold" meta |

### Deadband pill (`ui.jsx` `DeadbandPill` + `styles.css`)

| State | Trigger |
|---|---|
| Default | `.cc-dbpill` — dashed border, mono 11, `--cc-fg-3`, transparent bg |
| Hover | `:hover` → `--cc-surface-2` bg, `--cc-fg-3` border, `--cc-fg` text |

### Readout (`ui.jsx`)

| State | Trigger | Visual |
|---|---|---|
| Null | `value == null` | `—` in `--cc-fg-4` |
| Present (tone=heat) | `tone="heat"` + non-null value | `--cc-heat` colored mono |
| Present (tone=cool) | `tone="cool"` | `--cc-cool` colored mono |
| Live pulse | any non-null value change → internal `useEffect` applies `cc-readout--live` for 600ms | color fades `fg` → `cool` → `fg` over 600ms |

### Schedule card (`schedules-tab.jsx`)

| State | Trigger |
|---|---|
| Collapsed | default, or after toggle |
| Expanded | `forceExpanded` prop (states gallery) or local click on chevron |
| Active | `schedule.active === true` → "Active" badge + "Deactivate" ghost button |
| Inactive | `schedule.active === false` → "Inactive" badge + "Activate" secondary button |

### Period row (`schedules-tab.jsx` `PeriodRow`)

| State | Trigger |
|---|---|
| Normal | default |
| Inline delete confirm | `_forceConfirm` prop or local state after delete-icon click — renders danger-tinted row with `[Yes, delete] [Cancel]` |

### Time picker — clock mode (`period-modal.jsx` `ClockPicker`)

| State | Trigger |
|---|---|
| Hour selection | `_forceStep="hour"` or initial / after cancel. Clicking a number auto-advances to minute step |
| Minute selection | `_forceStep="minute"` or after hour selection. Majors (0/15/30/45) labeled; minors are 2px dots |
| AM active | `period === "AM"` — AM button gets `--cc-fg` fill, `--cc-fg-invert` text |
| PM active | `period === "PM"` — same pattern on PM button |

### Time picker — timeline mode (`period-modal.jsx`)

| State | Trigger |
|---|---|
| Single-day band | `pickerMode === "timeline" && !weekView`, or `_forcePickerMode="timeline"` + `_forceWeekView=false` |
| Week view | `weekView === true` or `_forceWeekView=true` — 7 daily rows. Preview block rendered only on selected days. Not draggable in this view |
| Knob drag | `mousedown` on start/end knob sets internal `drag` state; window `mousemove` updates values with 15-min snap |

### PeriodModal force flags

- `_forcePickerMode`: `"clock"` \| `"timeline"` — locks picker mode
- `_forceClockStep`: `"hour"` \| `"minute"` — passed through to ClockPicker
- `_forceActiveTimeField`: `"start"` \| `"end"` \| `null` — opens a time field popover on mount
- `_forceWeekView`: boolean

### TopNav (`shell.jsx`)

| State | Trigger |
|---|---|
| User menu closed | default |
| User menu open | `forceMenuOpen` or click on trigger — rounded pill trigger + dropdown with `cc-pop` |

### DeleteAccountModal

| State | Trigger |
|---|---|
| Input empty / mismatched | `typed !== account.email` — Delete button disabled |
| Match | `typed === account.email` — Delete button enabled |

---

## Known deviations from client.md

Intentionally different from, or not yet captured in, the architecture doc:

1. **Control panel `1fr 1.15fr` split.** `overview-tab.jsx` renders the two
   panels with the control panel slightly wider. client.md only says
   "side-by-side" / "two-column grid". If Phase 6 prefers `1fr 1fr`, flip
   it; not load-bearing.

2. **Grow-domain references remain in `states.jsx`.** The one hardcoded
   string `activeName="Vegetative stage"` and the synthetic-room objects
   are kept per the instruction to not modify the states artifact. The
   live app (`index.html`) uses `data.jsx` which is fully neutralized.

3. **Seed room count and composition.** `data.jsx` ships 6 rooms including
   one temp-only (Storage Closet) and one hum-only (Office) to exercise
   capability-aware rendering. Implementors should not wire this shape
   into production; real rooms come from `GET /rooms`.

4. **Mock history generator.** `data.jsx` `generateHistory` is a sinusoidal
   mock with a simulated null gap midway on 24h / 7d windows. Replace with
   `GET /rooms/:id/climate/history?window=…` in Phase 6. The mock's `t`,
   `tempAvg`, `humAvg`, `tempTarget`, `humTarget`, `tempDb`, `humDb`,
   `heatDuty`, `humDuty` keys mirror the real API field names.

5. **`hmToMin` / `minToHm` keep 24-hour format internally.** Period data
   stored/sent as "HH:MM" in 24-hour format (consistent with backend and
   `models.SchedulePeriod`). All display passes through `fmtTime12`,
   `fmtMin12`, or `fmtTick12`.

6. **`fmtTick12` ignores the `tz` parameter in the mockup.** The signature
   accepts `tz` for parity with production, but the mockup uses the
   browser's local time via `Date` getters. Production code must route
   through `Intl.DateTimeFormat` with the user's `timeZone` from
   `account.timezone`.

7. **Live pulse animation.** `Readout` drives this internally via
   `useEffect` comparing against a `useRef` snapshot. On real data from
   SWR, this will fire on every successful poll that returns a new value —
   which is the desired behaviour. Null-to-non-null transitions also
   trigger the pulse; null-to-null does not.

8. **TopNav avatar is a heat→cool gradient circle** rather than the
   `--cc-primary` / `--cc-primary-fg` convention. Purely stylistic; flip
   if the gradient reads as out of place in production.

9. **History chart is a raw SVG**, not Recharts. The mockup's `Chart` is
   structured as a pure-function renderer over `data` — port the visual
   logic (null-as-gap, dashed target band, opacity-fill duty cycle,
   plan-based x-ticks) into Recharts primitives (`LineChart`, `ReferenceArea`,
   `XAxis tickFormatter` wired to `fmtTick12`).

10. **Period modal `endMin > startMin` only.** Midnight-crossing periods
    are rejected — matches the backend constraint in CLAUDE.md
    ("`end_time` must be strictly greater than `start_time`"). If the
    product ever allows crossing, the timeline picker and clock validation
    both need revisiting.

11. **Source badge color mapping.** `dashboard.jsx` `SOURCE_MAP` keys
    to these variants/colors — memorize them:
    - `manual_override` → variant `err`, dot `--cc-hold`
    - `schedule` → variant `info`, dot `--cc-info`
    - `grace_period` → variant `warn`, dot `--cc-grace`
    - `none` → default variant, dot `--cc-fg-4`

12. **No staleness banner.** Deliberately deferred — see client.md "No
    staleness warning in Phase 6". The `last_updated` timestamps are
    displayed as `last updated Xm ago` meta but no warn tint is rendered.

13. **User menu ordering has two dividers, not one.** `shell.jsx` renders
    Account settings / `<hr>` / Log out / `<hr>` / Delete account, which
    matches client.md. Delete account always appears as a danger item.

14. **`ModeBadge` uses `warn` variant for OFF.** Mode=OFF renders with a
    warning-tint badge. client.md doesn't prescribe — keep or drop. It
    reads slightly louder than OFF deserves; consider default variant if
    it looks too alarming in context.

15. **Tolerances modal uses `InputUnit` non-mono.** The `InputUnit`
    defaults to `mono=true`; `modals-simple.jsx` `TolerancesModal`
    explicitly keeps the default. Matches the mono convention for
    numeric inputs throughout.

---

## Notes for the implementor

- The mockup uses `window.CCData` / `window.*` globals because Babel-in-
  browser prototypes can't do ES modules. In production, convert every
  `window.ComponentName` to a normal import and drop the
  `Object.assign(window, { … })` exports.
- `app.jsx`'s modal dispatch table is a reasonable pattern for
  production too — a single `<ModalHost>` that reads a `useModal()`
  context and renders the appropriate form. Or use each page's own
  local state — no strict requirement.
- For time inputs in forms, the mockup's internal storage is "HH:MM"
  (24-hour). Keep that convention — it's what the backend accepts and
  what `models.SchedulePeriod.StartTime` / `EndTime` expect.
- The `_force…` props on several components exist purely so the states
  gallery can render specific interaction states side-by-side. Do not
  port those to production — drive state via normal `useState` or SWR.
- Recharts replacement for `Chart`: use `LineChart`, `Line` with
  `connectNulls={false}`, `ReferenceLine` (or `ReferenceArea`) for the
  target band, and custom `XAxis` ticks via `tickFormatter: fmtTick12`.
  Duty-cycle opacity fill can be a `Customized` component drawing rects,
  or a separate `Area` with variable opacity per segment.
