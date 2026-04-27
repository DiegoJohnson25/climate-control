/* global React */
const { useState } = React;

// Common sensor/actuator chips for Device tables
const CapChip = ({ kind }) => {
  const map = {
    temp: { color: "var(--cc-heat-fg)", bg: "var(--cc-heat-tint)", label: "temp" },
    hum:  { color: "var(--cc-cool-fg)", bg: "var(--cc-cool-tint)", label: "hum" },
    heat: { color: "var(--cc-heat-fg)", bg: "var(--cc-heat-tint)", label: "heat" },
  };
  const m = map[kind];
  return (
    <span style={{
      display: "inline-block", padding: "2px 8px", borderRadius: 999,
      fontFamily: "var(--cc-font-mono)", fontSize: 10, letterSpacing: "0.04em",
      fontWeight: 500, color: m.color, background: m.bg,
    }}>{m.label}</span>
  );
};

// ============================================================
// DevicesTab — room-scoped
// ============================================================
const DevicesTab = ({ room, devices, onRegister, onEditDevice, onDeleteDevice, onGoToGlobal }) => {
  const roomDevices = devices.filter(d => d.roomId === room.id);

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", marginBottom: 14 }}>
        <button onClick={onGoToGlobal} style={{
          background: "transparent", border: "none", cursor: "pointer", padding: 0,
          display: "inline-flex", alignItems: "center", gap: 5,
          fontSize: 12, color: "var(--cc-fg-3)",
        }}>
          Manage all devices {window.Icon.chevronRight(12)}
        </button>
        <div style={{ flex: 1 }}/>
        <window.Button onClick={onRegister} icon={window.Icon.plus(14)}>Register device</window.Button>
      </div>

      <div className="cc-card" style={{ padding: 0, overflow: "hidden" }}>
        <table className="cc-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>hw_id</th>
              <th>Sensors</th>
              <th>Actuators</th>
              <th style={{ textAlign: "right", width: 90 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {roomDevices.length === 0 && (
              <tr>
                <td colSpan={5} style={{ padding: 32, textAlign: "center", color: "var(--cc-fg-3)" }}>
                  No devices registered to this room yet.
                </td>
              </tr>
            )}
            {roomDevices.map(d => (
              <tr key={d.id}>
                <td style={{ fontWeight: 500 }}>{d.name}</td>
                <td><span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 12, color: "var(--cc-fg-2)" }}>{d.hwId}</span></td>
                <td><div style={{ display: "flex", gap: 4 }}>{d.sensors.map(s => <CapChip key={s} kind={s}/>)}{d.sensors.length === 0 && <span className="cc-meta">—</span>}</div></td>
                <td><div style={{ display: "flex", gap: 4 }}>{d.actuators.map(a => <CapChip key={a} kind={a}/>)}{d.actuators.length === 0 && <span className="cc-meta">—</span>}</div></td>
                <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                  <window.IconBtn onClick={() => onEditDevice?.(d)} title="Edit device">{window.Icon.pencil(14)}</window.IconBtn>
                  <window.IconBtn onClick={() => onDeleteDevice?.(d)} danger title="Delete device">{window.Icon.trash(14)}</window.IconBtn>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// ============================================================
// Global Devices page
// ============================================================
const DevicesPage = ({ devices, rooms, onRegister, onEditDevice, onDeleteDevice, onChangeRoom }) => {
  return (
    <div style={{ maxWidth: 1280, margin: "0 auto", padding: "32px 24px 48px" }}>
      <div style={{ display: "flex", alignItems: "baseline", justifyContent: "space-between", marginBottom: 24 }}>
        <div style={{ display: "flex", alignItems: "baseline", gap: 12 }}>
          <h1 className="cc-h1" style={{ fontFamily: "var(--cc-font-sans)" }}>Devices</h1>
          <span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 13, color: "var(--cc-fg-3)" }}>
            {devices.length} total
          </span>
        </div>
        <window.Button onClick={onRegister} icon={window.Icon.plus(14)}>Register device</window.Button>
      </div>

      <div className="cc-card" style={{ padding: 0, overflow: "hidden" }}>
        <table className="cc-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>hw_id</th>
              <th>Capabilities</th>
              <th style={{ width: 200 }}>Room</th>
              <th style={{ textAlign: "right", width: 90 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {devices.map(d => (
              <tr key={d.id}>
                <td style={{ fontWeight: 500 }}>{d.name}</td>
                <td><span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 12, color: "var(--cc-fg-2)" }}>{d.hwId}</span></td>
                <td>
                  <div style={{ display: "flex", gap: 4, flexWrap: "wrap" }}>
                    {d.sensors.map(s => <CapChip key={"s"+s} kind={s}/>)}
                    {d.actuators.map(a => <CapChip key={"a"+a} kind={a}/>)}
                    {d.sensors.length === 0 && d.actuators.length === 0 && <span className="cc-meta">—</span>}
                  </div>
                </td>
                <td>
                  <window.Select
                    value={d.roomId || ""}
                    onChange={e => onChangeRoom?.(d.id, e.target.value || null)}
                    style={{ width: "100%" }}
                  >
                    <option value="">Unassigned</option>
                    {rooms.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
                  </window.Select>
                </td>
                <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                  <window.IconBtn onClick={() => onEditDevice?.(d)} title="Edit device">{window.Icon.pencil(14)}</window.IconBtn>
                  <window.IconBtn onClick={() => onDeleteDevice?.(d)} danger title="Delete device">{window.Icon.trash(14)}</window.IconBtn>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

Object.assign(window, { DevicesTab, DevicesPage, CapChip });
