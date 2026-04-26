/* global React */
const { useState } = React;

// ============================================================
// Source badge — maps backend source to display
// ============================================================
const SOURCE_MAP = {
  manual_override: { label: "Hold active",   variant: "err",  dotColor: "var(--cc-hold)" },
  schedule:        { label: "Schedule",      variant: "info", dotColor: "var(--cc-info)" },
  grace_period:    { label: "Grace period",  variant: "warn", dotColor: "var(--cc-grace)" },
  none:            { label: "Idle",          variant: undefined, dotColor: "var(--cc-fg-4)" },
};
const SourceBadge = ({ source }) => {
  const m = SOURCE_MAP[source] || SOURCE_MAP.none;
  return <window.Badge variant={m.variant} dot>{m.label}</window.Badge>;
};
const ModeBadge = ({ mode }) => (
  <window.Badge variant={mode === "auto" ? undefined : "warn"}>
    {mode === "auto" ? "AUTO" : "OFF"}
  </window.Badge>
);

// ============================================================
// RoomCard
// ============================================================
const RoomCard = ({ room, onOpen, forceHover = false }) => {
  const [hover, setHover] = useState(forceHover);
  const isHover = hover || forceHover;

  const I = window.Icon;
  const capabilityBadges = [];
  if (room.hasTemp) {
    capabilityBadges.push(
      <window.Badge key="heat" variant={room.heaterOn ? "heat" : undefined}>
        {I.flame(11)} Heater {room.heaterOn ? "on" : "off"}
      </window.Badge>
    );
  }
  if (room.hasHum) {
    capabilityBadges.push(
      <window.Badge key="hum" variant={room.humOn ? "cool" : undefined}>
        {I.droplets(11)} Humidifier {room.humOn ? "on" : "off"}
      </window.Badge>
    );
  }
  capabilityBadges.push(<SourceBadge key="src" source={room.source}/>);

  return (
    <button
      onClick={onOpen}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={{
        display: "block", width: "100%", textAlign: "left", padding: 0,
        background: "var(--cc-surface)",
        border: "1px solid " + (isHover ? "var(--cc-border-strong)" : "var(--cc-border)"),
        borderRadius: 8,
        boxShadow: isHover ? "var(--cc-shadow-md)" : "var(--cc-shadow-sm)",
        cursor: "pointer",
        transition: "border-color .15s, box-shadow .15s, transform .15s",
        transform: isHover ? "translateY(-1px)" : "translateY(0)",
        fontFamily: "inherit",
      }}
    >
      <div style={{ padding: "16px 18px 14px" }}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 10, marginBottom: 14 }}>
          <div style={{ fontSize: 15, fontWeight: 600, letterSpacing: "-0.01em" }}>{room.name}</div>
          <ModeBadge mode={room.mode}/>
        </div>

        <div style={{ display: "flex", gap: 22, alignItems: "baseline" }}>
          {room.hasTemp ? (
            <window.Readout value={room.tempC} unit="°C" tone="heat" size="sm"/>
          ) : (
            <div className="cc-readout-sm" style={{ color: "var(--cc-fg-4)" }}>—<span style={{ fontSize: "0.45em", marginLeft: 2, color: "var(--cc-fg-4)" }}>°C</span></div>
          )}
          {room.hasHum ? (
            <window.Readout value={room.humPct} unit="%" tone="cool" size="sm"/>
          ) : (
            <div className="cc-readout-sm" style={{ color: "var(--cc-fg-4)" }}>—<span style={{ fontSize: "0.45em", marginLeft: 2, color: "var(--cc-fg-4)" }}>%</span></div>
          )}
        </div>
      </div>

      <div style={{ borderTop: "1px solid var(--cc-divider)", padding: "12px 18px", display: "flex", flexWrap: "wrap", gap: 6 }}>
        {capabilityBadges}
      </div>
    </button>
  );
};

// ============================================================
// Dashboard
// ============================================================
const Dashboard = ({ rooms, onOpenRoom, onAddRoom }) => (
  <div style={{ maxWidth: 1280, margin: "0 auto", padding: "32px 24px 48px" }}>
    <div style={{ display: "flex", alignItems: "baseline", justifyContent: "space-between", marginBottom: 24 }}>
      <div style={{ display: "flex", alignItems: "baseline", gap: 12 }}>
        <h1 className="cc-h1" style={{ fontFamily: "var(--cc-font-sans)" }}>Rooms</h1>
        <span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 13, color: "var(--cc-fg-3)" }}>
          {rooms.length} total
        </span>
      </div>
      <window.Button onClick={onAddRoom} icon={window.Icon.plus(14)}>Add room</window.Button>
    </div>

    <div style={{
      display: "grid", gap: 16,
      gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))",
    }}>
      {rooms.map(r => <RoomCard key={r.id} room={r} onOpen={() => onOpenRoom(r.id)}/>)}
    </div>
  </div>
);

Object.assign(window, { Dashboard, RoomCard, SourceBadge, ModeBadge });
