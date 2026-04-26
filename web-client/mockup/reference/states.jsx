/* global React */
const { useState } = React;

// ============================================================
// Interaction States Gallery — every state called out by the brief
// ============================================================

const StateCard = ({ title, desc, bg = "var(--cc-surface)", children, width = 420, pad = 16 }) => (
  <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
    <div>
      <div style={{ fontSize: 13, fontWeight: 600, color: "var(--cc-fg)" }}>{title}</div>
      {desc && <div className="cc-meta" style={{ marginTop: 2, textTransform: "none", letterSpacing: 0, fontSize: 12 }}>{desc}</div>}
    </div>
    <div style={{
      background: bg, border: "1px solid var(--cc-border)", borderRadius: 8,
      padding: pad, minWidth: width,
    }}>
      {children}
    </div>
  </div>
);

const SectionHeader = ({ name, count, kicker }) => (
  <div style={{ padding: "40px 0 14px", borderBottom: "1px solid var(--cc-divider)", marginBottom: 26 }}>
    <div style={{ fontFamily: "var(--cc-font-mono)", fontSize: 11, color: "var(--cc-fg-3)", letterSpacing: "0.1em", textTransform: "uppercase" }}>
      {kicker}
    </div>
    <div style={{ display: "flex", alignItems: "baseline", gap: 10, marginTop: 6 }}>
      <h2 className="cc-h2" style={{ fontFamily: "var(--cc-font-sans)" }}>{name}</h2>
      <span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 12, color: "var(--cc-fg-3)" }}>
        {count} state{count === 1 ? "" : "s"}
      </span>
    </div>
  </div>
);

const Row = ({ children, cols }) => (
  <div style={{ display: "grid", gridTemplateColumns: cols || "repeat(auto-fit, minmax(420px, 1fr))", gap: 20, marginBottom: 22 }}>
    {children}
  </div>
);

// ---- helpers to build synthetic rooms for gallery demos ----
const roomA = window.CCData.ROOMS[0]; // full capability, schedule active
const roomHoldActive = { ...roomA, source: "manual_override", hold: { on: true, duration: "2h" },
                         desired: { ...roomA.desired } };
const roomNoSchedule = { ...roomA, activeScheduleId: null, source: "none" };
const roomNoHum = { ...roomA, hasHum: false, humPct: null, humOn: null,
                    desired: { ...roomA.desired, humOn: false, humPct: null, humDb: null } };
const roomOff = { ...roomA, mode: "off", source: "none",
                  desired: { ...roomA.desired, tempOn: true, humOn: true } };

const StatesGallery = () => {
  return (
    <div className="cc" style={{ minHeight: "100vh", background: "var(--cc-bg)" }}>
      <window.TopNav
        page="none"
        onNav={() => {}}
        account={window.CCData.ACCOUNT}
        onOpenAccount={() => {}}
        onLogout={() => {}}
      />

      <div style={{ maxWidth: 1280, margin: "0 auto", padding: "32px 24px 64px" }}>
        <div style={{ marginBottom: 24 }}>
          <div style={{ fontFamily: "var(--cc-font-mono)", fontSize: 11, color: "var(--cc-fg-3)", letterSpacing: "0.1em", textTransform: "uppercase" }}>
            Climate Control · Web Dashboard
          </div>
          <h1 className="cc-h1" style={{ fontFamily: "var(--cc-font-sans)", marginTop: 6 }}>Interaction states</h1>
          <p className="cc-body" style={{ marginTop: 6, color: "var(--cc-fg-3)", maxWidth: 640 }}>
            Every state called out in the design brief, rendered side-by-side so they can be
            reviewed without clicking through the live app. Organised by component.
          </p>
          <div style={{ marginTop: 16, display: "flex", gap: 10 }}>
            <a href="index.html" style={{ textDecoration: "none" }}>
              <window.Button variant="secondary" iconRight={window.Icon.chevronRight(14)}>Open live dashboard</window.Button>
            </a>
          </div>
        </div>

        {/* ============ DASHBOARD ROOMCARD ============ */}
        <SectionHeader kicker="Dashboard" name="Room card" count={2}/>
        <Row cols="repeat(auto-fit, minmax(320px, 400px))">
          <StateCard title="Default" desc="Resting border and shadow." bg="var(--cc-bg)" pad={20}>
            <window.RoomCard room={roomA} onOpen={() => {}}/>
          </StateCard>
          <StateCard title="Hover" desc="Lifted border + shadow + 1px translate." bg="var(--cc-bg)" pad={20}>
            <window.RoomCard room={roomA} onOpen={() => {}} forceHover/>
          </StateCard>
        </Row>

        {/* ============ CONTROL PANEL — SCHEDULE SECTION ============ */}
        <SectionHeader kicker="Control panel" name="Schedule section" count={3}/>
        <Row>
          <StateCard title="Active schedule" desc="Name visible, info-colored status dot.">
            <ScheduleSectionDemo activeName="Vegetative stage" holdActive={false}/>
          </StateCard>
          <StateCard title="Hold active" desc="Name and dot grey out — Hold subordinates schedule.">
            <ScheduleSectionDemo activeName="Vegetative stage" holdActive={true}/>
          </StateCard>
          <StateCard title="No schedule" desc={`"None" shown in muted color.`}>
            <ScheduleSectionDemo activeName={null} holdActive={false}/>
          </StateCard>
        </Row>

        {/* ============ CONTROL PANEL — MODE ============ */}
        <SectionHeader kicker="Control panel" name="Mode" count={2}/>
        <Row>
          <StateCard title="AUTO selected" desc="Capability rows visible below.">
            <ModeSectionDemo mode="auto"/>
          </StateCard>
          <StateCard title="OFF selected" desc="Capability rows collapsed — not relevant when holding OFF.">
            <ModeSectionDemo mode="off"/>
          </StateCard>
        </Row>

        {/* ============ CONTROL PANEL — CAPABILITY ROW ============ */}
        <SectionHeader kicker="Control panel" name="Capability row" count={3}/>
        <Row cols="repeat(auto-fit, minmax(420px, 1fr))">
          <StateCard title="Toggle ON + value + deadband" desc="Input, unit suffix, clickable deadband pill.">
            <CapRowDemo state="on"/>
          </StateCard>
          <StateCard title="Toggle OFF — not regulating" desc="Input hidden, muted label. Dot remains interactive.">
            <CapRowDemo state="off"/>
          </StateCard>
          <StateCard title="Capability absent + tooltip" desc="Row greyed; hovering the dot shows why.">
            <CapRowDemo state="absent"/>
          </StateCard>
        </Row>

        {/* ============ CONTROL PANEL — HOLD ============ */}
        <SectionHeader kicker="Control panel" name="Hold" count={3}/>
        <Row>
          <StateCard title="Off" desc="Just the toggle.">
            <HoldDemo state="off"/>
          </StateCard>
          <StateCard title="On — duration selector visible" desc="Inline chips for duration.">
            <HoldDemo state="on"/>
          </StateCard>
          <StateCard title="Disabled" desc="AUTO + no capabilities active → nothing to hold.">
            <HoldDemo state="disabled"/>
          </StateCard>
        </Row>

        {/* ============ DEADBAND PILL ============ */}
        <SectionHeader kicker="Control panel" name="Deadband pill" count={2}/>
        <Row cols="repeat(auto-fit, minmax(300px, 360px))">
          <StateCard title="Default" pad={20}>
            <div style={{ display: "flex", gap: 20, alignItems: "center" }}>
              <window.DeadbandPill value="0.5" unit="°C" onClick={() => {}}/>
              <window.DeadbandPill value="3.0" unit="%" onClick={() => {}}/>
            </div>
          </StateCard>
          <StateCard title="Hover" desc="Filled background signals clickability." pad={20}>
            <div style={{ display: "flex", gap: 20, alignItems: "center" }}>
              <DeadbandHover value="0.5" unit="°C"/>
              <DeadbandHover value="3.0" unit="%"/>
            </div>
          </StateCard>
        </Row>

        {/* ============ SCHEDULE CARD ============ */}
        <SectionHeader kicker="Schedules tab" name="Schedule card" count={2}/>
        <Row cols="1fr">
          <StateCard title="Collapsed" desc="Header only — chevron, name, period count, active badge, actions." width={900}>
            <window.ScheduleCard
              schedule={window.CCData.SCHEDULES[0]}
              room={roomA}
            />
          </StateCard>
          <StateCard title="Expanded" desc="Period table revealed below, surface-2 background." width={900}>
            <window.ScheduleCard
              schedule={window.CCData.SCHEDULES[0]}
              room={roomA}
              forceExpanded
            />
          </StateCard>
        </Row>

        {/* ============ PERIOD DELETE INLINE ============ */}
        <SectionHeader kicker="Schedules tab" name="Period delete confirm" count={2}/>
        <Row cols="1fr">
          <StateCard title="Normal row" width={900}>
            <window.ScheduleCard
              schedule={window.CCData.SCHEDULES[0]}
              room={roomA}
              forceExpanded
            />
          </StateCard>
          <StateCard title="Inline confirm" desc={`"Delete this period? Yes / Cancel" replaces the row — no modal.`} width={900}>
            <window.ScheduleCard
              schedule={window.CCData.SCHEDULES[0]}
              room={roomA}
              forceExpanded
              _forceConfirmPeriodId="p1"
            />
          </StateCard>
        </Row>

        {/* ============ TIME PICKERS ============ */}
        <SectionHeader kicker="Period modal · time pickers" name="Clock picker" count={2}/>
        <Row cols="repeat(auto-fit, minmax(320px, 420px))">
          <StateCard title="Hour selection" desc="1–12 on circular face. Tapping a number auto-advances to minutes." pad={20}>
            <div style={{ display: "flex", justifyContent: "center" }}>
              <window.ClockPicker value={9 * 60 + 0} onChange={() => {}} _forceStep="hour" onConfirm={()=>{}} onCancel={()=>{}}/>
            </div>
          </StateCard>
          <StateCard title="Minute selection" desc="Majors at 0/15/30/45; dots between. Confirm writes HH:MM back to field." pad={20}>
            <div style={{ display: "flex", justifyContent: "center" }}>
              <window.ClockPicker value={9 * 60 + 45} onChange={() => {}} _forceStep="minute" onConfirm={()=>{}} onCancel={()=>{}}/>
            </div>
          </StateCard>
        </Row>

        <SectionHeader kicker="Period modal · time pickers" name="Timeline picker" count={2}/>
        <Row cols="1fr">
          <StateCard title="Default band view" desc="24h horizontal band, two draggable knobs, filtered existing periods for selected days." width={900}>
            <TimelineDemo mode="single"/>
          </StateCard>
          <StateCard title="Week view expanded" desc="7 daily rows, existing blocks per day, preview block on selected days." width={900}>
            <TimelineDemo mode="week"/>
          </StateCard>
        </Row>

        {/* ============ Full control panel variants ============ */}
        <SectionHeader kicker="Composed" name="Control panel" count={3}/>
        <Row cols="1fr 1fr">
          <StateCard title="AUTO, full capability, schedule active" width={520}>
            <window.ControlPanel room={roomA} schedules={window.CCData.SCHEDULES} onOpenTolerances={()=>{}}/>
          </StateCard>
          <StateCard title="Hold active → schedule subordinated" width={520}>
            <window.ControlPanel room={roomHoldActive} schedules={window.CCData.SCHEDULES} onOpenTolerances={()=>{}}
              _forceState={{ draft: roomHoldActive.desired, mode: "auto", hold: { on: true, duration: "2h" } }}/>
          </StateCard>
          <StateCard title="OFF mode — capability rows collapsed" width={520}>
            <window.ControlPanel room={roomOff} schedules={window.CCData.SCHEDULES} onOpenTolerances={()=>{}}
              _forceState={{ draft: roomOff.desired, mode: "off", hold: roomOff.hold }}/>
          </StateCard>
          <StateCard title="Humidity not available — row greyed" width={520}>
            <window.ControlPanel room={roomNoHum} schedules={window.CCData.SCHEDULES} onOpenTolerances={()=>{}}/>
          </StateCard>
        </Row>

        <div style={{ marginTop: 48, padding: "20px 0", borderTop: "1px solid var(--cc-divider)",
                       display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <span className="cc-meta">End of states reference.</span>
          <a href="index.html" style={{ textDecoration: "none" }}>
            <window.Button variant="secondary" iconRight={window.Icon.chevronRight(14)}>Open live dashboard</window.Button>
          </a>
        </div>
      </div>
    </div>
  );
};

// ---- Small demo snippets for the gallery -----------------------------------
const ScheduleSectionDemo = ({ activeName, holdActive }) => (
  <div>
    <div className="cc-section-label" style={{ marginBottom: 8 }}>Schedule</div>
    <div style={{
      display: "flex", alignItems: "center", gap: 10,
      padding: "10px 12px", border: "1px solid var(--cc-border)", borderRadius: 6,
      background: "var(--cc-surface-2)", opacity: holdActive ? 0.5 : 1,
    }}>
      <span className="cc-statusdot" style={{ background: !activeName ? "var(--cc-fg-4)" : holdActive ? "var(--cc-fg-4)" : "var(--cc-info)" }}/>
      <span style={{ fontSize: 13, color: activeName ? "var(--cc-fg)" : "var(--cc-fg-4)", flex: 1 }}>
        {activeName || "None"}
      </span>
      {holdActive && <span className="cc-meta">Overridden by Hold</span>}
    </div>
  </div>
);

const ModeSectionDemo = ({ mode }) => {
  const [m, setM] = useState(mode);
  React.useEffect(() => setM(mode), [mode]);
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <div>
        <div className="cc-section-label" style={{ marginBottom: 8 }}>Mode</div>
        <window.Segmented value={m} onChange={setM}
          options={[{ value: "off", label: "OFF" }, { value: "auto", label: "AUTO" }]}/>
      </div>
      {m === "auto" ? (
        <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
          <div className="cc-row">
            <window.ToggleDot on onClick={()=>{}}/>
            <span style={{ color: "var(--cc-fg-2)", display: "flex", alignItems: "center", gap: 6, minWidth: 110 }}>
              {window.Icon.thermometer(14)} Temperature
            </span>
            <window.InputUnit value={24.0} onChange={()=>{}} unit="°C" style={{ width: 96 }}/>
            <window.DeadbandPill value="0.5" unit="°C" onClick={()=>{}}/>
          </div>
          <div className="cc-row">
            <window.ToggleDot on onClick={()=>{}}/>
            <span style={{ color: "var(--cc-fg-2)", display: "flex", alignItems: "center", gap: 6, minWidth: 110 }}>
              {window.Icon.droplets(14)} Humidity
            </span>
            <window.InputUnit value={65} onChange={()=>{}} unit="%" style={{ width: 96 }}/>
            <window.DeadbandPill value="3.0" unit="%" onClick={()=>{}}/>
          </div>
        </div>
      ) : (
        <div style={{
          padding: "10px 12px", border: "1px dashed var(--cc-border-strong)",
          borderRadius: 6, fontSize: 12, color: "var(--cc-fg-3)",
          fontFamily: "var(--cc-font-mono)", letterSpacing: "0.02em",
          background: "var(--cc-surface-2)",
        }}>Capability rows hidden — room is holding OFF.</div>
      )}
    </div>
  );
};

const CapRowDemo = ({ state }) => {
  if (state === "on") {
    return (
      <div className="cc-row">
        <window.ToggleDot on onClick={()=>{}}/>
        <span style={{ color: "var(--cc-fg-2)", display: "flex", alignItems: "center", gap: 6, minWidth: 110 }}>
          {window.Icon.thermometer(14)} Temperature
        </span>
        <window.InputUnit value={24.0} onChange={()=>{}} unit="°C" style={{ width: 96 }}/>
        <window.DeadbandPill value="0.5" unit="°C" onClick={()=>{}}/>
      </div>
    );
  }
  if (state === "off") {
    return (
      <div className="cc-row">
        <window.ToggleDot on={false} onClick={()=>{}}/>
        <span style={{ color: "var(--cc-fg-2)", display: "flex", alignItems: "center", gap: 6, minWidth: 110 }}>
          {window.Icon.thermometer(14)} Temperature
        </span>
        <span className="cc-meta" style={{ flex: 1, paddingLeft: 4 }}>Not regulating</span>
      </div>
    );
  }
  // absent
  return (
    <div>
      <div className="cc-row cc-row--disabled">
        <window.Tooltip text="No humidity sensor or actuator in this room" forceShow>
          <window.ToggleDot on={false} disabled/>
        </window.Tooltip>
        <span style={{ color: "var(--cc-fg-4)", display: "flex", alignItems: "center", gap: 6, minWidth: 110 }}>
          {window.Icon.droplets(14)} Humidity
        </span>
        <div style={{ flex: 1 }}/>
        <span className="cc-meta">Not available</span>
      </div>
      <div style={{ marginTop: 32, fontFamily: "var(--cc-font-mono)", fontSize: 10, color: "var(--cc-fg-3)" }}>
        ↑ tooltip rendered eagerly for reference
      </div>
    </div>
  );
};

const HoldDemo = ({ state }) => {
  const on = state === "on";
  const disabled = state === "disabled";
  return (
    <div>
      <div className="cc-section-label" style={{ marginBottom: 8 }}>Hold</div>
      <div style={{ display: "flex", alignItems: "center", gap: 14, flexWrap: "wrap" }}>
        <window.Segmented value={on ? "on" : "off"} onChange={()=>{}}
          options={[{ value: "off", label: "Off" }, { value: "on", label: "On" }]}
          disabled={disabled}/>
        {on && (
          <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
            <span className="cc-meta" style={{ textTransform: "none", letterSpacing: 0 }}>for</span>
            <window.Chip on={false} onClick={()=>{}}>30 min</window.Chip>
            <window.Chip on={true} onClick={()=>{}}>1h</window.Chip>
            <window.Chip on={false} onClick={()=>{}}>2h</window.Chip>
            <window.Chip on={false} onClick={()=>{}}>4h</window.Chip>
            <window.Chip on={false} onClick={()=>{}}>Indefinite</window.Chip>
          </div>
        )}
        {disabled && (
          <span className="cc-meta" style={{ textTransform: "none", letterSpacing: 0 }}>Enable a capability to hold</span>
        )}
      </div>
    </div>
  );
};

const DeadbandHover = ({ value, unit }) => (
  <button className="cc-dbpill" style={{
    color: "var(--cc-fg)", borderColor: "var(--cc-fg-3)", background: "var(--cc-surface-2)",
  }}>±{value}{unit}</button>
);

const TimelineDemo = ({ mode }) => {
  const [startMin, setStartMin] = useState(8 * 60);
  const [endMin, setEndMin] = useState(20 * 60);
  const existingOnMon = [{ start: 20 * 60, end: 24 * 60, label: "20:00–24:00" },
                        { start: 0, end: 6 * 60, label: "00:00–06:00" }];
  if (mode === "single") {
    return (
      <div>
        <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 10 }}>
          <span className="cc-label">Time range</span>
          <div style={{ flex: 1 }}/>
          <span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 12, color: "var(--cc-fg-2)" }}>
            08:00 → 20:00
          </span>
        </div>
        <window.DayTimeline
          startMin={startMin} endMin={endMin}
          onChange={({ start, end }) => { setStartMin(start); setEndMin(end); }}
          existingBlocks={existingOnMon}
        />
      </div>
    );
  }
  // week view
  const DAY_NAMES = ["Mon","Tue","Wed","Thu","Fri","Sat","Sun"];
  const selectedDays = [1,1,1,1,1,0,0];
  const existingPerDay = {
    0: [{ start: 20*60, end: 24*60 }, { start: 0, end: 6*60 }],
    1: [{ start: 20*60, end: 24*60 }, { start: 0, end: 6*60 }],
    2: [{ start: 20*60, end: 24*60 }, { start: 0, end: 6*60 }],
    3: [{ start: 20*60, end: 24*60 }, { start: 0, end: 6*60 }],
    4: [{ start: 20*60, end: 24*60 }, { start: 0, end: 6*60 }],
    5: [{ start: 7*60, end: 19*60 }],
    6: [{ start: 7*60, end: 19*60 }],
  };
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
      {DAY_NAMES.map((dn, i) => {
        const sel = !!selectedDays[i];
        return (
          <div key={i} style={{ display: "grid", gridTemplateColumns: "36px 1fr", gap: 8, alignItems: "center" }}>
            <div style={{ fontFamily: "var(--cc-font-mono)", fontSize: 11,
                           color: sel ? "var(--cc-fg)" : "var(--cc-fg-4)",
                           fontWeight: sel ? 600 : 400 }}>{dn}</div>
            <div style={{ position: "relative", height: 26, background: "var(--cc-surface-2)",
                           border: "1px solid var(--cc-border)", borderRadius: 4,
                           opacity: sel ? 1 : 0.55, overflow: "hidden" }}>
              {[0,6,12,18,24].map(h => (
                <div key={h} style={{ position: "absolute", left: `${(h/24)*100}%`, top: 0, bottom: 0, width: 1, background: "var(--cc-divider)" }}/>
              ))}
              {(existingPerDay[i] || []).map((b, j) => (
                <div key={j} style={{
                  position: "absolute", left: `${(b.start/(24*60))*100}%`,
                  width: `${((b.end-b.start)/(24*60))*100}%`,
                  top: 3, bottom: 3, background: "rgba(120, 118, 111, 0.22)",
                  border: "1px solid rgba(120, 118, 111, 0.45)", borderRadius: 2,
                }}/>
              ))}
              {sel && (
                <div style={{
                  position: "absolute", left: `${(startMin/(24*60))*100}%`,
                  width: `${((endMin - startMin)/(24*60))*100}%`,
                  top: 2, bottom: 2, background: "var(--cc-heat-tint)",
                  border: "1.5px solid var(--cc-heat)", borderRadius: 3,
                }}/>
              )}
            </div>
          </div>
        );
      })}
      <div style={{ display: "flex", justifyContent: "space-between",
                     fontFamily: "var(--cc-font-mono)", fontSize: 9, color: "var(--cc-fg-4)",
                     marginLeft: 44, marginTop: 2 }}>
        {[0,3,6,9,12,15,18,21,24].map(h => <span key={h}>{String(h).padStart(2,"0")}</span>)}
      </div>
    </div>
  );
};

const root = ReactDOM.createRoot(document.getElementById("root"));
root.render(<StatesGallery/>);
