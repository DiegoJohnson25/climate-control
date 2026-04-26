/* global React */
const { useState } = React;

const DAY_LABELS = ["M","T","W","T","F","S","S"];

// ============================================================
// Period row (inside expanded schedule card)
// ============================================================
const PeriodRow = ({ period, room, onEdit, onDelete, _forceConfirm }) => {
  const [confirming, setConfirming] = useState(_forceConfirm || false);

  if (confirming) {
    return (
      <tr style={{ background: "var(--cc-danger-tint)" }}>
        <td colSpan={6} style={{ padding: "10px 14px" }}>
          <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
            <span style={{ color: "var(--cc-danger-fg)", display: "flex" }}>{window.Icon.alert(14)}</span>
            <span style={{ fontSize: 13, color: "var(--cc-danger-fg)", flex: 1 }}>
              Delete this period? <span style={{ fontFamily: "var(--cc-font-mono)", color: "var(--cc-fg-3)", marginLeft: 6 }}>
                {window.fmtTime12(period.start)} – {window.fmtTime12(period.end)}
              </span>
            </span>
            <window.Button size="sm" variant="danger" onClick={() => { onDelete?.(); setConfirming(false); }}>Yes, delete</window.Button>
            <window.Button size="sm" variant="ghost" onClick={() => setConfirming(false)}>Cancel</window.Button>
          </div>
        </td>
      </tr>
    );
  }

  return (
    <tr>
      <td>
        <div style={{ display: "flex", gap: 3 }}>
          {DAY_LABELS.map((d, i) => (
            <window.DayChip key={i} label={d} on={!!period.days[i]} readonly/>
          ))}
        </div>
      </td>
      <td>
        <span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 13, fontVariantNumeric: "tabular-nums" }}>
          {window.fmtTime12(period.start)}
        </span>
      </td>
      <td>
        <span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 13, fontVariantNumeric: "tabular-nums" }}>
          {window.fmtTime12(period.end)}
        </span>
      </td>
      {room.hasTemp && (
        <td>
          {period.tempC != null ? (
            <span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 13, fontVariantNumeric: "tabular-nums", color: "var(--cc-heat-fg)" }}>
              {period.tempC.toFixed(1)}°C
            </span>
          ) : <span className="cc-meta">—</span>}
        </td>
      )}
      {room.hasHum && (
        <td>
          {period.humPct != null ? (
            <span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 13, fontVariantNumeric: "tabular-nums", color: "var(--cc-cool-fg)" }}>
              {period.humPct}%
            </span>
          ) : <span className="cc-meta">—</span>}
        </td>
      )}
      <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
        <window.IconBtn onClick={onEdit} title="Edit period">{window.Icon.pencil(14)}</window.IconBtn>
        <window.IconBtn onClick={() => setConfirming(true)} danger title="Delete period">{window.Icon.trash(14)}</window.IconBtn>
      </td>
    </tr>
  );
};

// ============================================================
// Schedule card — collapsed + expanded
// ============================================================
const ScheduleCard = ({ schedule, room, forceExpanded = false, onEditName, onDelete, onToggleActive, onAddPeriod, onEditPeriod, onDeletePeriod, _forceConfirmPeriodId }) => {
  const [expanded, setExpanded] = useState(forceExpanded);

  return (
    <div style={{
      background: "var(--cc-surface)",
      border: "1px solid var(--cc-border)",
      borderRadius: 8, overflow: "hidden",
    }}>
      {/* Collapsed header (always visible) */}
      <div style={{ display: "flex", alignItems: "center", gap: 12, padding: "12px 16px" }}>
        <button onClick={() => setExpanded(e => !e)}
                className="cc-iconbtn" title={expanded ? "Collapse" : "Expand"}>
          {expanded ? window.Icon.chevronDown(16) : window.Icon.chevronRight(16)}
        </button>
        <span style={{ fontSize: 14, fontWeight: 600, color: "var(--cc-fg)" }}>{schedule.name}</span>
        <span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 11, color: "var(--cc-fg-3)" }}>
          {schedule.periods.length} period{schedule.periods.length === 1 ? "" : "s"}
        </span>
        <window.Badge variant={schedule.active ? "ok" : undefined}>
          {schedule.active ? "Active" : "Inactive"}
        </window.Badge>
        <div style={{ flex: 1 }}/>
        <window.Button size="sm" variant={schedule.active ? "ghost" : "secondary"} onClick={onToggleActive}>
          {schedule.active ? "Deactivate" : "Activate"}
        </window.Button>
        <window.KebabMenu items={[
          { label: "Edit name", onClick: onEditName },
          { divider: true },
          { label: "Delete schedule", danger: true, onClick: onDelete },
        ]}/>
      </div>

      {/* Expanded body */}
      {expanded && (
        <div style={{ background: "var(--cc-surface-2)", borderTop: "1px solid var(--cc-border)", padding: "6px 0 12px" }}>
          <table className="cc-table">
            <thead>
              <tr>
                <th>Days</th>
                <th>Start</th>
                <th>End</th>
                {room.hasTemp && <th>Target temp</th>}
                {room.hasHum && <th>Target hum</th>}
                <th style={{ textAlign: "right", width: 90 }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {schedule.periods.map(p => (
                <PeriodRow
                  key={p.id}
                  period={p}
                  room={room}
                  onEdit={() => onEditPeriod?.(p)}
                  onDelete={() => onDeletePeriod?.(p.id)}
                  _forceConfirm={_forceConfirmPeriodId === p.id}
                />
              ))}
            </tbody>
          </table>

          <div style={{ padding: "8px 14px 0" }}>
            <window.Button variant="ghost" size="sm" icon={window.Icon.plus(14)} onClick={onAddPeriod}>
              Add period
            </window.Button>
          </div>
        </div>
      )}
    </div>
  );
};

// ============================================================
// SchedulesTab
// ============================================================
const SchedulesTab = ({ room, schedules, onAddSchedule, onEditSchedule, onDeleteSchedule, onToggleActive, onAddPeriod, onEditPeriod, onDeletePeriod }) => {
  const list = schedules.filter(s => s.roomId === room.id);

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 16 }}>
        <div>
          <span className="cc-section-label">{list.length} schedule{list.length === 1 ? "" : "s"}</span>
        </div>
        <window.Button onClick={onAddSchedule} icon={window.Icon.plus(14)}>Add schedule</window.Button>
      </div>

      {list.length === 0 ? (
        <EmptyScheduleState onAdd={onAddSchedule}/>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
          {list.map(s => (
            <ScheduleCard
              key={s.id}
              schedule={s}
              room={room}
              onEditName={() => onEditSchedule?.(s)}
              onDelete={() => onDeleteSchedule?.(s)}
              onToggleActive={() => onToggleActive?.(s.id)}
              onAddPeriod={() => onAddPeriod?.(s.id)}
              onEditPeriod={(p) => onEditPeriod?.(s.id, p)}
              onDeletePeriod={(pid) => onDeletePeriod?.(s.id, pid)}
            />
          ))}
        </div>
      )}
    </div>
  );
};

const EmptyScheduleState = ({ onAdd }) => (
  <div className="cc-card" style={{ padding: 40, textAlign: "center" }}>
    <div style={{
      width: 40, height: 40, borderRadius: 8, background: "var(--cc-surface-2)",
      display: "inline-flex", alignItems: "center", justifyContent: "center",
      color: "var(--cc-fg-3)", marginBottom: 14,
    }}>{window.Icon.calendar(20)}</div>
    <div style={{ fontSize: 15, fontWeight: 600, marginBottom: 6 }}>No schedules yet</div>
    <div className="cc-body" style={{ maxWidth: 360, margin: "0 auto 18px", color: "var(--cc-fg-3)" }}>
      Schedules let you set target temperature and humidity across the week. Create one to automate climate control for this room.
    </div>
    <window.Button onClick={onAdd} icon={window.Icon.plus(14)}>Add schedule</window.Button>
  </div>
);

Object.assign(window, { SchedulesTab, ScheduleCard, PeriodRow, EmptyScheduleState });
