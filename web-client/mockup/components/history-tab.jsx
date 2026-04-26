/* global React */
const { useState, useMemo } = React;

// ============================================================
// HistoryTab — two stacked charts, shared x-axis + window selector
// ============================================================
const WINDOWS = [
  { value: "1h",  label: "1h",  tickMin: 15 },
  { value: "6h",  label: "6h",  tickMin: 60 },
  { value: "24h", label: "24h", tickMin: 360 },
  { value: "7d",  label: "7d",  tickMin: 1440 },
];

const generateTimeTicks = (windowKey, minT, maxT, tz) => {
  const def = WINDOWS.find(w => w.value === windowKey);
  const spanMs = maxT - minT;
  const stepMs = def.tickMin * 60 * 1000;
  const ticks = [];
  // Round first tick to a clean minute boundary
  const first = Math.ceil(minT / stepMs) * stepMs;
  for (let t = first; t <= maxT; t += stepMs) ticks.push(t);
  return ticks.map(t => ({ t, label: window.fmtTick12(t, windowKey, tz) }));
};

// ---- Chart primitive -----------------------------------------------------
const Chart = ({
  data, height = 220, color, dutyColor,
  valueKey, targetKey, dbKey, dutyKey,
  unit, minDomain, maxDomain,
  showTargetBand, showDuty,
  windowKey, tz, tickSource,
  showXAxis = true,
}) => {
  const width = 780;                  // virtual px; SVG is responsive
  const padL = 48, padR = 16, padT = 10, padB = showXAxis ? 28 : 10;
  const innerW = width - padL - padR;
  const innerH = height - padT - padB;

  if (!data || data.length === 0) return <div style={{ height }}/>;

  const minT = data[0].t, maxT = data[data.length - 1].t;

  // y domain
  let vals = data.map(d => d[valueKey]).filter(v => v != null);
  if (targetKey && showTargetBand) {
    data.forEach(d => {
      if (d[targetKey] != null && d[dbKey] != null) {
        vals.push(d[targetKey] + d[dbKey]);
        vals.push(d[targetKey] - d[dbKey]);
      }
    });
  }
  const minV = Math.min(...vals) - 0.5;
  const maxV = Math.max(...vals) + 0.5;
  const lo = minDomain != null ? Math.min(minDomain, minV) : minV;
  const hi = maxDomain != null ? Math.max(maxDomain, maxV) : maxV;

  const x = (t) => padL + ((t - minT) / (maxT - minT)) * innerW;
  const y = (v) => padT + (1 - (v - lo) / (hi - lo)) * innerH;

  // build path with null-as-gap
  const buildPath = (key) => {
    let d = "";
    let penUp = true;
    data.forEach((p) => {
      const v = p[key];
      if (v == null) { penUp = true; return; }
      const cmd = penUp ? "M" : "L";
      d += `${cmd}${x(p.t).toFixed(1)},${y(v).toFixed(1)} `;
      penUp = false;
    });
    return d;
  };

  const valuePath = buildPath(valueKey);

  // target ± db (dashed)
  const targetHiPath = (targetKey && showTargetBand) ? (() => {
    let d = ""; let penUp = true;
    data.forEach(p => {
      if (p[targetKey] == null || p[dbKey] == null) { penUp = true; return; }
      const v = p[targetKey] + p[dbKey];
      d += (penUp ? "M" : "L") + `${x(p.t).toFixed(1)},${y(v).toFixed(1)} `;
      penUp = false;
    });
    return d;
  })() : "";
  const targetLoPath = (targetKey && showTargetBand) ? (() => {
    let d = ""; let penUp = true;
    data.forEach(p => {
      if (p[targetKey] == null || p[dbKey] == null) { penUp = true; return; }
      const v = p[targetKey] - p[dbKey];
      d += (penUp ? "M" : "L") + `${x(p.t).toFixed(1)},${y(v).toFixed(1)} `;
      penUp = false;
    });
    return d;
  })() : "";

  // duty-cycle background: fill columns per sample
  const dutyBars = (dutyKey && showDuty) ? data.map((p, i) => {
    if (p[dutyKey] == null || p[dutyKey] === 0) return null;
    const x0 = x(p.t);
    const next = data[i + 1];
    const x1 = next ? x(next.t) : x0 + (innerW / data.length);
    return (
      <rect key={i} x={x0} y={padT} width={Math.max(0.5, x1 - x0)} height={innerH}
            fill={dutyColor} opacity={p[dutyKey]}/>
    );
  }) : null;

  const ticks = tickSource || generateTimeTicks(windowKey, minT, maxT, tz);

  // y ticks: min / mid / max
  const yTicks = [lo, (lo + hi) / 2, hi];

  return (
    <svg width="100%" viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none"
         style={{ display: "block", overflow: "visible" }}>
      {/* Duty fill */}
      {dutyBars}

      {/* Grid (horizontal y-tick lines) */}
      {yTicks.map((v, i) => (
        <line key={i} x1={padL} x2={padL + innerW} y1={y(v)} y2={y(v)}
              stroke="var(--cc-divider)" strokeWidth="1"/>
      ))}
      {/* Y tick labels */}
      {yTicks.map((v, i) => (
        <text key={"yt"+i} x={padL - 8} y={y(v)} fontSize="10" textAnchor="end" dominantBaseline="middle"
              fill="var(--cc-fg-3)" fontFamily="var(--cc-font-mono)">
          {v.toFixed(1)}{unit}
        </text>
      ))}

      {/* Target band (dashed) */}
      {targetHiPath && (
        <>
          <path d={targetHiPath} fill="none" stroke={color} strokeWidth="1" strokeDasharray="3 4" opacity={0.55}/>
          <path d={targetLoPath} fill="none" stroke={color} strokeWidth="1" strokeDasharray="3 4" opacity={0.55}/>
        </>
      )}

      {/* Value line */}
      <path d={valuePath} fill="none" stroke={color} strokeWidth="1.6" strokeLinejoin="round"/>

      {/* X ticks */}
      {showXAxis && ticks.map(({ t, label }, i) => (
        <g key={i}>
          <line x1={x(t)} x2={x(t)} y1={padT + innerH} y2={padT + innerH + 4}
                stroke="var(--cc-fg-3)" strokeWidth="1"/>
          <text x={x(t)} y={padT + innerH + 16} fontSize="10" textAnchor="middle"
                fill="var(--cc-fg-3)" fontFamily="var(--cc-font-mono)">{label}</text>
        </g>
      ))}

      {/* border */}
      <line x1={padL} x2={padL + innerW} y1={padT + innerH} y2={padT + innerH} stroke="var(--cc-border-strong)" strokeWidth="1"/>
    </svg>
  );
};

// ---- one chart panel -----------------------------------------------------
const ChartPanel = ({ title, icon, unit, data, color, dutyColor,
                      valueKey, targetKey, dbKey, dutyKey,
                      windowKey, tz, showXAxis }) => {
  const [showBand, setShowBand] = useState(true);
  const [showDuty, setShowDuty] = useState(true);
  return (
    <div className="cc-card" style={{ padding: 20 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 12 }}>
        <span style={{ color, display: "flex" }}>{icon}</span>
        <span style={{ fontSize: 13, fontWeight: 600, letterSpacing: "-0.005em" }}>{title}</span>
        <div style={{ flex: 1 }}/>
        <div style={{ display: "flex", gap: 6 }}>
          <TinyToggle on={showBand} onClick={() => setShowBand(b => !b)} label="Target band"/>
          <TinyToggle on={showDuty} onClick={() => setShowDuty(d => !d)} label="Duty cycle"/>
        </div>
      </div>
      <Chart
        data={data} color={color} dutyColor={dutyColor}
        valueKey={valueKey} targetKey={targetKey} dbKey={dbKey} dutyKey={dutyKey}
        unit={unit} windowKey={windowKey} tz={tz}
        showTargetBand={showBand} showDuty={showDuty}
        showXAxis={showXAxis}
      />
    </div>
  );
};

const TinyToggle = ({ on, onClick, label }) => (
  <button onClick={onClick} style={{
    display: "inline-flex", alignItems: "center", gap: 6,
    padding: "3px 8px", borderRadius: 4,
    fontFamily: "var(--cc-font-mono)", fontSize: 11, letterSpacing: "0.02em",
    border: "1px solid " + (on ? "var(--cc-border-strong)" : "var(--cc-border)"),
    background: on ? "var(--cc-surface-2)" : "transparent",
    color: on ? "var(--cc-fg)" : "var(--cc-fg-3)",
    cursor: "pointer",
  }}>
    <span style={{
      width: 7, height: 7, borderRadius: "50%",
      background: on ? "var(--cc-success)" : "var(--cc-fg-4)",
    }}/>
    {label}
  </button>
);

// ---- History tab wrapper -------------------------------------------------
const HistoryTab = ({ room, tz }) => {
  const [windowKey, setWindowKey] = useState("24h");
  const data = useMemo(() => window.CCData.generateHistory(windowKey, room), [windowKey, room.id]);

  return (
    <div>
      <div style={{ display: "flex", alignItems: "flex-start", marginBottom: 16, gap: 14 }}>
        <window.Segmented
          value={windowKey}
          onChange={setWindowKey}
          options={WINDOWS.map(w => ({ value: w.value, label: w.label }))}
        />
        <div style={{ flex: 1 }}/>
        <div style={{ display: "flex", alignItems: "center", gap: 8, height: 30 }}>
          <span className="cc-meta">Timezone</span>
          <span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 12, color: "var(--cc-fg-2)" }}>{tz}</span>
        </div>
      </div>

      <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
        {room.hasTemp && (
          <ChartPanel
            title="Temperature" icon={window.Icon.thermometer(14)} unit="°C"
            data={data} color="var(--cc-heat)" dutyColor="var(--cc-heat)"
            valueKey="tempAvg" targetKey="tempTarget" dbKey="tempDb" dutyKey="heatDuty"
            windowKey={windowKey} tz={tz}
            showXAxis={!room.hasHum}
          />
        )}
        {room.hasHum && (
          <ChartPanel
            title="Humidity" icon={window.Icon.droplets(14)} unit="%"
            data={data} color="var(--cc-cool)" dutyColor="var(--cc-cool)"
            valueKey="humAvg" targetKey="humTarget" dbKey="humDb" dutyKey="humDuty"
            windowKey={windowKey} tz={tz}
            showXAxis={true}
          />
        )}
      </div>
    </div>
  );
};

Object.assign(window, { HistoryTab });
