/* global React */
const { useState } = React;

// ============================================================
// Current-state card (left column of Overview)
// ============================================================
const CurrentStateCard = ({ room }) => {
  const I = window.Icon;
  const stateItems = [];
  if (room.hasTemp) stateItems.push({
    label: "Heater", on: room.heaterOn, color: "var(--cc-heat)",
    icon: I.flame(14),
  });
  if (room.hasHum) stateItems.push({
    label: "Humidifier", on: room.humOn, color: "var(--cc-cool)",
    icon: I.droplets(14),
  });

  return (
    <div className="cc-card" style={{ padding: 24 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 16 }}>
        <span className="cc-section-label">Current state</span>
        <div style={{ flex: 1 }}/>
        <span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 11, color: "var(--cc-fg-3)" }}>
          live
        </span>
        <span style={{
          width: 6, height: 6, borderRadius: "50%", background: "var(--cc-success)",
          boxShadow: "0 0 0 3px var(--cc-success-tint)",
        }}/>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 24, marginBottom: 22 }}>
        <div>
          <div className="cc-meta" style={{ display: "flex", alignItems: "center", gap: 5, marginBottom: 8 }}>
            {I.thermometer(12)} Temperature
          </div>
          {room.hasTemp ? (
            <window.Readout value={room.tempC} unit="°C" tone="heat"/>
          ) : (
            <div className="cc-readout" style={{ color: "var(--cc-fg-4)" }}>—<span style={{ fontSize: "0.45em", marginLeft: 2 }}>°C</span></div>
          )}
          <div style={{ fontFamily: "var(--cc-font-mono)", fontSize: 11, color: "var(--cc-fg-3)", marginTop: 6 }}>
            {room.hasTemp ? `last updated ${window.timeAgo(room.tempUpdated)}` : "no sensor"}
          </div>
        </div>
        <div>
          <div className="cc-meta" style={{ display: "flex", alignItems: "center", gap: 5, marginBottom: 8 }}>
            {I.droplets(12)} Humidity
          </div>
          {room.hasHum ? (
            <window.Readout value={room.humPct} unit="%" tone="cool"/>
          ) : (
            <div className="cc-readout" style={{ color: "var(--cc-fg-4)" }}>—<span style={{ fontSize: "0.45em", marginLeft: 2 }}>%</span></div>
          )}
          <div style={{ fontFamily: "var(--cc-font-mono)", fontSize: 11, color: "var(--cc-fg-3)", marginTop: 6 }}>
            {room.hasHum ? `last updated ${window.timeAgo(room.humUpdated)}` : "no sensor"}
          </div>
        </div>
      </div>

      <div style={{ borderTop: "1px solid var(--cc-divider)", paddingTop: 18, display: "flex", flexDirection: "column", gap: 12 }}>
        {stateItems.map(it => (
          <div key={it.label} style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <span style={{ color: it.on ? it.color : "var(--cc-fg-4)", display: "flex" }}>{it.icon}</span>
            <span style={{ fontSize: 13, color: "var(--cc-fg-2)", flex: 1 }}>{it.label}</span>
            <span style={{
              fontFamily: "var(--cc-font-mono)", fontSize: 11, letterSpacing: "0.04em",
              textTransform: "uppercase", fontWeight: 500,
              color: it.on ? it.color : "var(--cc-fg-4)",
            }}>{it.on ? "ON" : "OFF"}</span>
          </div>
        ))}

        <div style={{ display: "flex", alignItems: "center", gap: 10, paddingTop: 10, borderTop: "1px solid var(--cc-divider)" }}>
          <span style={{ fontSize: 13, color: "var(--cc-fg-2)", flex: 1 }}>Active source</span>
          <window.SourceBadge source={room.source}/>
        </div>

        {(room.hasTemp || room.hasHum) && (
          <div style={{ display: "flex", gap: 20, paddingTop: 12, borderTop: "1px solid var(--cc-divider)" }}>
            {room.hasTemp && (
              <div style={{ flex: 1 }}>
                <div className="cc-meta" style={{ marginBottom: 4 }}>Target temp</div>
                <div style={{ fontFamily: "var(--cc-font-mono)", fontSize: 14, fontWeight: 500, color: "var(--cc-fg)", fontVariantNumeric: "tabular-nums" }}>
                  {room.targets.tempC?.toFixed(1)}°C
                  <span style={{ fontSize: 11, color: "var(--cc-fg-3)", marginLeft: 6 }}>±{room.targets.tempDb}</span>
                </div>
              </div>
            )}
            {room.hasHum && (
              <div style={{ flex: 1 }}>
                <div className="cc-meta" style={{ marginBottom: 4 }}>Target humidity</div>
                <div style={{ fontFamily: "var(--cc-font-mono)", fontSize: 14, fontWeight: 500, color: "var(--cc-fg)", fontVariantNumeric: "tabular-nums" }}>
                  {room.targets.humPct}%
                  <span style={{ fontSize: 11, color: "var(--cc-fg-3)", marginLeft: 6 }}>±{room.targets.humDb}</span>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

// ============================================================
// Control Panel card (right column) — the interaction-heavy one
// ============================================================
const ControlPanel = ({ room, schedules, onApply, onRevert, onOpenTolerances,
                        // state overrides for state gallery
                        _forceState }) => {
  // local draft of desired-state — can diverge from room.desired until Apply
  const [draft, setDraft] = useState(_forceState?.draft || room.desired);
  const [mode, setMode] = useState(_forceState?.mode || room.mode);
  const [hold, setHold] = useState(_forceState?.hold || room.hold);

  const activeSched = schedules.find(s => s.id === room.activeScheduleId);
  const activeScheduleName = activeSched?.name || null;
  const holdActive = hold.on;

  const anyCapOn = (draft.tempOn && room.hasTemp) || (draft.humOn && room.hasHum);
  const holdDisabled = mode === "auto" && !anyCapOn;

  const CapRow = ({ cap }) => {
    // cap is "temp" or "hum"
    const has = cap === "temp" ? room.hasTemp : room.hasHum;
    const on  = cap === "temp" ? draft.tempOn : draft.humOn;
    const value = cap === "temp" ? draft.tempC : draft.humPct;
    const db    = cap === "temp" ? draft.tempDb : draft.humDb;
    const label = cap === "temp" ? "Temperature" : "Humidity";
    const unit  = cap === "temp" ? "°C" : "%";
    const icon  = cap === "temp" ? window.Icon.thermometer(14) : window.Icon.droplets(14);

    const toggle = () => {
      if (!has) return;
      if (cap === "temp") setDraft(d => ({ ...d, tempOn: !d.tempOn }));
      else setDraft(d => ({ ...d, humOn: !d.humOn }));
    };
    const setValue = (v) => {
      if (cap === "temp") setDraft(d => ({ ...d, tempC: v }));
      else setDraft(d => ({ ...d, humPct: v }));
    };

    if (!has) {
      return (
        <div className="cc-row cc-row--disabled" title={`No ${label.toLowerCase()} sensor or actuator in this room`}>
          <window.Tooltip text={`No ${label.toLowerCase()} sensor or actuator in this room`}>
            <window.ToggleDot on={false} disabled title=""/>
          </window.Tooltip>
          <span style={{ color: "var(--cc-fg-4)", display: "flex", alignItems: "center", gap: 6 }}>{icon} {label}</span>
          <div style={{ flex: 1 }}/>
          <span className="cc-meta">Not available</span>
        </div>
      );
    }

    return (
      <div className="cc-row">
        <window.ToggleDot on={on} onClick={toggle}/>
        <span style={{ color: "var(--cc-fg-2)", display: "flex", alignItems: "center", gap: 6, minWidth: 110 }}>{icon} {label}</span>
        {on ? (
          <div style={{ display: "flex", alignItems: "center", gap: 10, flex: 1 }}>
            <window.InputUnit
              mono value={value}
              onChange={e => setValue(parseFloat(e.target.value))}
              unit={unit}
              style={{ width: 96 }}
            />
            <window.DeadbandPill value={db.toFixed(1)} unit={unit} onClick={onOpenTolerances}/>
          </div>
        ) : (
          <span className="cc-meta" style={{ flex: 1, paddingLeft: 4 }}>Not regulating</span>
        )}
      </div>
    );
  };

  const holdDurationOptions = [
    { value: "30m", label: "30 min" },
    { value: "1h",  label: "1h" },
    { value: "2h",  label: "2h" },
    { value: "4h",  label: "4h" },
    { value: "inf", label: "Indefinite" },
  ];

  return (
    <div className="cc-card" style={{ padding: 24, display: "flex", flexDirection: "column", gap: 22 }}>
      <div style={{ display: "flex", alignItems: "center" }}>
        <span className="cc-section-label">Control panel</span>
        <div style={{ flex: 1 }}/>
      </div>

      {/* Schedule section */}
      <div>
        <div className="cc-section-label" style={{ marginBottom: 8 }}>Schedule</div>
        <div style={{
          display: "flex", alignItems: "center", gap: 10,
          padding: "10px 12px", border: "1px solid var(--cc-border)", borderRadius: 6,
          background: "var(--cc-surface-2)",
          opacity: holdActive ? 0.5 : 1,
        }}>
          <span className="cc-statusdot" style={{
            background: !activeScheduleName ? "var(--cc-fg-4)" : holdActive ? "var(--cc-fg-4)" : "var(--cc-info)",
          }}/>
          <span style={{ fontSize: 13, color: activeScheduleName ? "var(--cc-fg)" : "var(--cc-fg-4)", flex: 1 }}>
            {activeScheduleName || "None"}
          </span>
          {holdActive && <span className="cc-meta">Overridden by Hold</span>}
        </div>
      </div>

      {/* Mode section */}
      <div>
        <div className="cc-section-label" style={{ marginBottom: 8 }}>Mode</div>
        <window.Segmented
          value={mode}
          onChange={setMode}
          options={[{ value: "off", label: "OFF" }, { value: "auto", label: "AUTO" }]}
        />
      </div>

      {/* Capability rows */}
      {mode === "auto" && (
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <CapRow cap="temp"/>
          <CapRow cap="hum"/>
        </div>
      )}

      {/* Hold section */}
      <div>
        <div className="cc-section-label" style={{ marginBottom: 8 }}>Hold</div>
        <div style={{ display: "flex", alignItems: "center", gap: 14, flexWrap: "wrap" }}
             title={holdDisabled ? "Turn on at least one capability to use Hold" : ""}>
          <window.Segmented
            value={hold.on ? "on" : "off"}
            onChange={v => setHold(h => ({ ...h, on: v === "on" }))}
            options={[{ value: "off", label: "Off" }, { value: "on", label: "On" }]}
            disabled={holdDisabled}
          />
          {hold.on && !holdDisabled && (
            <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
              <span className="cc-meta" style={{ textTransform: "none", letterSpacing: 0 }}>for</span>
              {holdDurationOptions.map(o => (
                <window.Chip
                  key={o.value}
                  on={hold.duration === o.value}
                  onClick={() => setHold(h => ({ ...h, duration: o.value }))}
                >{o.label}</window.Chip>
              ))}
            </div>
          )}
          {holdDisabled && (
            <span className="cc-meta" style={{ textTransform: "none", letterSpacing: 0 }}>
              Enable a capability to hold
            </span>
          )}
        </div>
      </div>

      {/* Footer */}
      <div style={{ display: "flex", justifyContent: "flex-end", gap: 8, paddingTop: 4, borderTop: "1px solid var(--cc-divider)", marginTop: 4, paddingTop: 18 }}>
        <window.Button variant="ghost" onClick={() => { setDraft(room.desired); setMode(room.mode); setHold(room.hold); onRevert?.(); }}>
          Revert
        </window.Button>
        <window.Button onClick={() => onApply?.({ draft, mode, hold })}>Apply</window.Button>
      </div>
    </div>
  );
};

// ============================================================
// Overview tab
// ============================================================
const OverviewTab = ({ room, schedules, onOpenTolerances, onApply, onRevert }) => (
  <div style={{ display: "grid", gridTemplateColumns: "1fr 1.15fr", gap: 20, alignItems: "start" }}>
    <CurrentStateCard room={room}/>
    <ControlPanel room={room} schedules={schedules}
                  onOpenTolerances={onOpenTolerances}
                  onApply={onApply} onRevert={onRevert}/>
  </div>
);

Object.assign(window, { OverviewTab, CurrentStateCard, ControlPanel });
