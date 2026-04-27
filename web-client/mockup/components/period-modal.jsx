/* global React */
const { useState, useEffect, useMemo } = React;

// ============================================================
// Period Modal — Add/Edit period, with Clock + Timeline picker modes
// ============================================================

const DAY_LETTERS = ["M","T","W","T","F","S","S"];
const DAY_NAMES = ["Mon","Tue","Wed","Thu","Fri","Sat","Sun"];

// Parse "HH:MM" → minutes-of-day (internal format, stays 24h)
const hmToMin = (hm) => {
  if (!hm) return 0;
  if (hm === "24:00") return 24 * 60;
  const [h, m] = hm.split(":").map(Number);
  return h * 60 + m;
};
// minutes-of-day → "HH:MM" (internal format, stays 24h)
const minToHm = (min) => {
  min = Math.max(0, Math.min(24 * 60, Math.round(min)));
  if (min === 24 * 60) return "24:00";
  const h = Math.floor(min / 60), m = min % 60;
  return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}`;
};
const min12 = (min) => {
  // returns { h12, m, period }
  const total = min % (24 * 60);
  const h24 = Math.floor(total / 60);
  const m = total % 60;
  const period = h24 >= 12 ? "PM" : "AM";
  let h12 = h24 % 12; if (h12 === 0) h12 = 12;
  return { h12, m, period };
};
const hm12to24 = (h12, m, period) => {
  let h24 = h12 % 12;
  if (period === "PM") h24 += 12;
  return h24 * 60 + m;
};

// ============================================================
// Clock picker (hour → minute, AM/PM toggle)
// ============================================================
const ClockPicker = ({ value /* minutes */, onChange, onConfirm, onCancel, _forceStep }) => {
  const { h12: initH, m: initM, period: initP } = min12(value ?? 9 * 60);
  const [h12, setH12] = useState(initH);
  const [m, setM] = useState(initM);
  const [period, setPeriod] = useState(initP);
  const [step, setStep] = useState(_forceStep || "hour");  // hour | minute

  useEffect(() => setStep(_forceStep || step), [_forceStep]);

  const SIZE = 220;
  const R = SIZE / 2 - 18;
  const C = SIZE / 2;

  // positions on clock face for hours 1-12 or minutes 0,5,10...55
  const hourPos = (n) => {
    const angle = (n / 12) * 2 * Math.PI - Math.PI / 2;
    return { x: C + Math.cos(angle) * R, y: C + Math.sin(angle) * R };
  };
  const minPos = (n) => {
    const angle = (n / 60) * 2 * Math.PI - Math.PI / 2;
    return { x: C + Math.cos(angle) * R, y: C + Math.sin(angle) * R };
  };

  const selectedAngle = step === "hour"
    ? (h12 / 12) * 2 * Math.PI - Math.PI / 2
    : (m / 60) * 2 * Math.PI - Math.PI / 2;
  const handEnd = { x: C + Math.cos(selectedAngle) * (R - 10), y: C + Math.sin(selectedAngle) * (R - 10) };

  const commitMinutes = (newMm, newPeriod) => {
    const finalM = newMm != null ? newMm : m;
    const finalP = newPeriod != null ? newPeriod : period;
    const total = hm12to24(h12, finalM, finalP);
    onChange?.(total);
  };

  const selectHour = (n) => {
    setH12(n);
    const total = hm12to24(n, m, period);
    onChange?.(total);
    setStep("minute"); // auto-advance
  };
  const selectMinute = (n) => {
    setM(n);
    commitMinutes(n);
  };

  return (
    <div style={{
      background: "var(--cc-surface)", border: "1px solid var(--cc-border-strong)",
      borderRadius: 10, padding: 18, width: 280,
      boxShadow: "var(--cc-shadow-lg)",
    }}>
      {/* Time display / step indicators */}
      <div style={{ display: "flex", alignItems: "baseline", justifyContent: "center", gap: 4, marginBottom: 14 }}>
        <button onClick={() => setStep("hour")}
          style={{
            background: "none", border: "none", padding: "2px 6px", borderRadius: 4,
            fontFamily: "var(--cc-font-mono)", fontSize: 32, fontWeight: 500, cursor: "pointer",
            color: step === "hour" ? "var(--cc-fg)" : "var(--cc-fg-3)",
            backgroundColor: step === "hour" ? "var(--cc-surface-2)" : "transparent",
            fontVariantNumeric: "tabular-nums",
          }}>
          {String(h12).padStart(2, "0")}
        </button>
        <span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 32, fontWeight: 500, color: "var(--cc-fg-3)" }}>:</span>
        <button onClick={() => setStep("minute")}
          style={{
            background: "none", border: "none", padding: "2px 6px", borderRadius: 4,
            fontFamily: "var(--cc-font-mono)", fontSize: 32, fontWeight: 500, cursor: "pointer",
            color: step === "minute" ? "var(--cc-fg)" : "var(--cc-fg-3)",
            backgroundColor: step === "minute" ? "var(--cc-surface-2)" : "transparent",
            fontVariantNumeric: "tabular-nums",
          }}>
          {String(m).padStart(2, "0")}
        </button>
        <div style={{ display: "flex", flexDirection: "column", gap: 2, marginLeft: 10 }}>
          <button onClick={() => { setPeriod("AM"); commitMinutes(null, "AM"); }}
            style={{ fontSize: 11, fontFamily: "var(--cc-font-mono)", letterSpacing: "0.04em",
                     padding: "2px 8px", borderRadius: 3, border: "1px solid " + (period === "AM" ? "var(--cc-fg)" : "var(--cc-border-strong)"),
                     background: period === "AM" ? "var(--cc-fg)" : "var(--cc-surface)",
                     color: period === "AM" ? "var(--cc-fg-invert)" : "var(--cc-fg-2)",
                     cursor: "pointer" }}>AM</button>
          <button onClick={() => { setPeriod("PM"); commitMinutes(null, "PM"); }}
            style={{ fontSize: 11, fontFamily: "var(--cc-font-mono)", letterSpacing: "0.04em",
                     padding: "2px 8px", borderRadius: 3, border: "1px solid " + (period === "PM" ? "var(--cc-fg)" : "var(--cc-border-strong)"),
                     background: period === "PM" ? "var(--cc-fg)" : "var(--cc-surface)",
                     color: period === "PM" ? "var(--cc-fg-invert)" : "var(--cc-fg-2)",
                     cursor: "pointer" }}>PM</button>
        </div>
      </div>

      {/* Clock face */}
      <div style={{ display: "flex", justifyContent: "center", marginBottom: 14 }}>
        <svg width={SIZE} height={SIZE}>
          <circle cx={C} cy={C} r={R + 10} fill="var(--cc-surface-2)"/>
          {/* Hand */}
          <line x1={C} y1={C} x2={handEnd.x} y2={handEnd.y} stroke="var(--cc-primary)" strokeWidth="2"/>
          <circle cx={C} cy={C} r="3" fill="var(--cc-primary)"/>
          <circle cx={handEnd.x} cy={handEnd.y} r="14" fill="var(--cc-primary)" opacity="0.12"/>
          <circle cx={handEnd.x} cy={handEnd.y} r="3" fill="var(--cc-primary)"/>

          {step === "hour" ? (
            [1,2,3,4,5,6,7,8,9,10,11,12].map(n => {
              const p = hourPos(n);
              const sel = n === h12;
              return (
                <g key={n} onClick={() => selectHour(n)} style={{ cursor: "pointer" }}>
                  <circle cx={p.x} cy={p.y} r="14" fill="transparent"/>
                  <text x={p.x} y={p.y} fontFamily="var(--cc-font-mono)" fontSize="13"
                        textAnchor="middle" dominantBaseline="central"
                        fill={sel ? "var(--cc-primary-fg)" : "var(--cc-fg-2)"}
                        fontWeight={sel ? 600 : 400}>{n}</text>
                </g>
              );
            })
          ) : (
            Array.from({length: 12}, (_, i) => i * 5).map(n => {
              const p = minPos(n);
              const major = n % 15 === 0;
              const sel = n === m;
              return (
                <g key={n} onClick={() => selectMinute(n)} style={{ cursor: "pointer" }}>
                  <circle cx={p.x} cy={p.y} r="14" fill="transparent"/>
                  {major ? (
                    <text x={p.x} y={p.y} fontFamily="var(--cc-font-mono)" fontSize="13"
                          textAnchor="middle" dominantBaseline="central"
                          fill={sel ? "var(--cc-primary-fg)" : "var(--cc-fg-2)"}
                          fontWeight={sel ? 600 : 400}>{String(n).padStart(2,"0")}</text>
                  ) : (
                    <circle cx={p.x} cy={p.y} r="2" fill={sel ? "var(--cc-primary-fg)" : "var(--cc-fg-4)"}/>
                  )}
                </g>
              );
            })
          )}
        </svg>
      </div>

      {/* Step indicator dots */}
      <div style={{ display: "flex", justifyContent: "center", gap: 6, marginBottom: 12 }}>
        <span style={{ width: 6, height: 6, borderRadius: "50%", background: step === "hour" ? "var(--cc-primary)" : "var(--cc-border-strong)" }}/>
        <span style={{ width: 6, height: 6, borderRadius: "50%", background: step === "minute" ? "var(--cc-primary)" : "var(--cc-border-strong)" }}/>
      </div>

      <div style={{ display: "flex", justifyContent: "flex-end", gap: 6 }}>
        <window.Button variant="ghost" size="sm" onClick={onCancel}>Cancel</window.Button>
        <window.Button size="sm" onClick={onConfirm}>Confirm</window.Button>
      </div>
    </div>
  );
};

// ============================================================
// Timeline picker — 24h band with two draggable knobs + existing period blocks
// ============================================================
const DayTimeline = ({
  startMin, endMin, onChange,
  existingBlocks, // [{ start, end, label }]
  height = 52, editable = true, showHours = true, label,
}) => {
  const ref = React.useRef(null);
  const [drag, setDrag] = useState(null);

  const pctToMin = (pct) => Math.round((pct * 24 * 60) / 15) * 15; // snap to 15min
  const fromClientX = (clientX) => {
    const r = ref.current.getBoundingClientRect();
    return Math.max(0, Math.min(1, (clientX - r.left) / r.width));
  };

  const startPct = (startMin / (24 * 60)) * 100;
  const endPct   = (endMin / (24 * 60)) * 100;

  useEffect(() => {
    if (!drag) return;
    const onMove = (e) => {
      const pct = fromClientX(e.clientX);
      const min = pctToMin(pct);
      if (drag === "start") onChange?.({ start: Math.min(min, endMin - 15), end: endMin });
      else if (drag === "end") onChange?.({ start: startMin, end: Math.max(min, startMin + 15) });
    };
    const onUp = () => setDrag(null);
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
    return () => { window.removeEventListener("mousemove", onMove); window.removeEventListener("mouseup", onUp); };
  }, [drag, startMin, endMin, onChange]);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
      {label && <div style={{ fontFamily: "var(--cc-font-mono)", fontSize: 10, color: "var(--cc-fg-3)", letterSpacing: "0.04em", textTransform: "uppercase" }}>{label}</div>}
      <div ref={ref} style={{
        position: "relative", height, width: "100%",
        background: "var(--cc-surface-2)",
        border: "1px solid var(--cc-border)", borderRadius: 5,
        overflow: "hidden",
      }}>
        {/* Hour ticks */}
        {Array.from({length: 25}, (_, i) => (
          <div key={i} style={{
            position: "absolute", left: `${(i/24)*100}%`, top: 0, bottom: 0, width: 1,
            background: i % 6 === 0 ? "var(--cc-border-strong)" : "var(--cc-divider)",
          }}/>
        ))}

        {/* Existing periods (read-only) */}
        {existingBlocks?.map((b, i) => (
          <div key={i} style={{
            position: "absolute",
            left: `${(b.start / (24*60))*100}%`,
            width: `${((b.end - b.start) / (24*60))*100}%`,
            top: 4, bottom: 4,
            background: "rgba(120, 118, 111, 0.22)",
            border: "1px solid rgba(120, 118, 111, 0.45)",
            borderRadius: 3,
            display: "flex", alignItems: "center", justifyContent: "center",
            fontFamily: "var(--cc-font-mono)", fontSize: 10,
            color: "var(--cc-fg-3)", overflow: "hidden",
          }}>
            {b.label}
          </div>
        ))}

        {/* Active selection band */}
        <div style={{
          position: "absolute",
          left: `${startPct}%`, width: `${endPct - startPct}%`,
          top: 3, bottom: 3,
          background: "var(--cc-heat-tint)", border: "1.5px solid var(--cc-heat)",
          borderRadius: 4,
          display: "flex", alignItems: "center", justifyContent: "center",
          fontFamily: "var(--cc-font-mono)", fontSize: 10,
          color: "var(--cc-heat-fg)",
        }}>
          {window.fmtMin12(startMin)} – {window.fmtMin12(endMin)}
        </div>

        {/* Start knob */}
        {editable && (
          <div onMouseDown={() => setDrag("start")}
               style={{
                 position: "absolute", left: `calc(${startPct}% - 7px)`, top: 0, bottom: 0,
                 width: 14, cursor: "ew-resize",
                 display: "flex", alignItems: "center", justifyContent: "center",
               }}>
            <div style={{ width: 4, height: "65%", background: "var(--cc-heat)", borderRadius: 2 }}/>
          </div>
        )}
        {/* End knob */}
        {editable && (
          <div onMouseDown={() => setDrag("end")}
               style={{
                 position: "absolute", left: `calc(${endPct}% - 7px)`, top: 0, bottom: 0,
                 width: 14, cursor: "ew-resize",
                 display: "flex", alignItems: "center", justifyContent: "center",
               }}>
            <div style={{ width: 4, height: "65%", background: "var(--cc-heat)", borderRadius: 2 }}/>
          </div>
        )}
      </div>
      {showHours && (
        <div style={{ display: "flex", justifyContent: "space-between",
                      fontFamily: "var(--cc-font-mono)", fontSize: 9, color: "var(--cc-fg-4)", marginTop: 2 }}>
          {[0,3,6,9,12,15,18,21,24].map(h => <span key={h}>{window.fmtMin12(h * 60)}</span>)}
        </div>
      )}
    </div>
  );
};

// ============================================================
// Period modal proper
// ============================================================
const PeriodModal = ({
  open, onClose, mode = "add", room, schedule, period,
  allPeriods = [], onSave,
  _forcePickerMode, _forceClockStep, _forceActiveTimeField, _forceWeekView,
}) => {
  // Form state
  const [days, setDays] = useState(period?.days || [1,1,1,1,1,0,0]);
  const [startMin, setStartMin] = useState(hmToMin(period?.start || "08:00"));
  const [endMin, setEndMin] = useState(hmToMin(period?.end || "20:00"));
  const [tempC, setTempC] = useState(period?.tempC ?? 21.0);
  const [humPct, setHumPct] = useState(period?.humPct ?? 50);

  const [pickerMode, setPickerMode] = useState(_forcePickerMode || "clock"); // clock | timeline
  const [activeTimeField, setActiveTimeField] = useState(_forceActiveTimeField || null); // "start" | "end" | null
  const [weekView, setWeekView] = useState(_forceWeekView || false);

  // reset when opening
  useEffect(() => {
    if (!open) return;
    if (period) {
      setDays(period.days);
      setStartMin(hmToMin(period.start));
      setEndMin(hmToMin(period.end));
      setTempC(period.tempC ?? 21.0);
      setHumPct(period.humPct ?? 50);
    } else {
      setDays([1,1,1,1,1,0,0]); setStartMin(hmToMin("08:00")); setEndMin(hmToMin("20:00"));
      setTempC(21.0); setHumPct(50);
    }
    setPickerMode(_forcePickerMode || "clock");
    setActiveTimeField(_forceActiveTimeField || null);
    setWeekView(_forceWeekView || false);
  }, [open, period?.id]);

  // Blocks existing in same schedule, filtered to selected days (Option B).
  // Block labels use 12-hour AM/PM display format.
  const existingBlocksForDay = (dayIdx) => {
    return allPeriods
      .filter(p => p.id !== period?.id)
      .filter(p => p.days[dayIdx])
      .map(p => ({
        start: hmToMin(p.start), end: hmToMin(p.end),
        label: `${window.fmtTime12(p.start)}–${window.fmtTime12(p.end)}`,
      }));
  };
  const anySelectedDay = days.findIndex(d => d === 1);
  const defaultBandBlocks = anySelectedDay >= 0 ? existingBlocksForDay(anySelectedDay) : [];

  const atLeastOneTarget = (room.hasTemp && tempC != null) || (room.hasHum && humPct != null);
  const atLeastOneDay = days.some(d => d === 1);
  const valid = atLeastOneDay && atLeastOneTarget && endMin > startMin;

  const handleSave = () => {
    onSave?.({
      id: period?.id,
      days,
      start: minToHm(startMin),
      end: minToHm(endMin),
      tempC: room.hasTemp ? tempC : null,
      humPct: room.hasHum ? humPct : null,
    });
    onClose?.();
  };

  // ----- Header with picker-mode toggle -----
  const HeaderToggle = (
    <div style={{ display: "inline-flex", border: "1px solid var(--cc-border-strong)",
                  borderRadius: 5, padding: 2, background: "var(--cc-surface-2)" }}>
      <button onClick={() => { setPickerMode("clock"); setActiveTimeField(null); setWeekView(false); }}
        style={{
          background: pickerMode === "clock" ? "var(--cc-surface)" : "transparent",
          border: "none", padding: "3px 9px", borderRadius: 3, cursor: "pointer",
          fontSize: 11, fontFamily: "var(--cc-font-mono)", color: "var(--cc-fg)",
          boxShadow: pickerMode === "clock" ? "0 1px 2px rgba(0,0,0,.05)" : "none",
          display: "inline-flex", alignItems: "center", gap: 4,
        }}>{window.Icon.clock(12)} Clock</button>
      <button onClick={() => { setPickerMode("timeline"); setActiveTimeField(null); }}
        style={{
          background: pickerMode === "timeline" ? "var(--cc-surface)" : "transparent",
          border: "none", padding: "3px 9px", borderRadius: 3, cursor: "pointer",
          fontSize: 11, fontFamily: "var(--cc-font-mono)", color: "var(--cc-fg)",
          boxShadow: pickerMode === "timeline" ? "0 1px 2px rgba(0,0,0,.05)" : "none",
          display: "inline-flex", alignItems: "center", gap: 4,
        }}>{window.Icon.rows(12)} Timeline</button>
    </div>
  );

  const timeFieldBtn = (which, value, label) => {
    const isActive = activeTimeField === which;
    return (
      <div style={{ position: "relative" }}>
        <window.Field label={label}>
          <button
            onClick={() => setActiveTimeField(isActive ? null : which)}
            style={{
              height: 32, padding: "0 12px", width: "100%",
              background: "var(--cc-surface)",
              border: "1px solid " + (isActive ? "var(--cc-ring)" : "var(--cc-border-strong)"),
              borderRadius: 4, textAlign: "left", cursor: "pointer",
              fontFamily: "var(--cc-font-mono)", fontSize: 13, fontVariantNumeric: "tabular-nums",
              display: "flex", alignItems: "center", gap: 6,
              boxShadow: isActive ? "0 0 0 3px rgba(8,145,178,0.18)" : "none",
            }}>
            {window.Icon.clock(13)} {window.fmtMin12(value)}
          </button>
        </window.Field>
        {isActive && pickerMode === "clock" && (
          <div style={{ position: "absolute", top: "calc(100% + 6px)", left: 0, zIndex: 30 }}>
            <ClockPicker
              value={value}
              onChange={(v) => which === "start" ? setStartMin(v) : setEndMin(v)}
              onConfirm={() => setActiveTimeField(null)}
              onCancel={() => setActiveTimeField(null)}
              _forceStep={_forceClockStep}
            />
          </div>
        )}
      </div>
    );
  };

  return (
    <window.Modal
      open={open} onClose={onClose} width={640}
      title={
        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
          <span>{mode === "add" ? "Add period" : "Edit period"}</span>
          {HeaderToggle}
        </div>
      }
      subtitle={schedule ? <>In <strong style={{ color: "var(--cc-fg-2)" }}>{schedule.name}</strong></> : null}
      footer={<><window.Button variant="ghost" onClick={onClose}>Cancel</window.Button>
               <window.Button onClick={handleSave} disabled={!valid}>Save</window.Button></>}
    >
      <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
        {/* Days picker */}
        <window.Field label="Days" hint={atLeastOneDay ? null : <span style={{ color: "var(--cc-danger-fg)" }}>Select at least one day</span>}>
          <div style={{ display: "flex", gap: 6 }}>
            {DAY_LETTERS.map((d, i) => (
              <button key={i} onClick={() => { const n = [...days]; n[i] = n[i] ? 0 : 1; setDays(n); }}
                className={"cc-daychip" + (days[i] ? " cc-daychip--on" : "")}
                style={{ width: 30, height: 30, fontSize: 12 }}>{d}</button>
            ))}
          </div>
        </window.Field>

        {/* TIME PICKER AREA ---------------------------------------- */}
        {pickerMode === "clock" ? (
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 14 }}>
            {timeFieldBtn("start", startMin, "Start time")}
            {timeFieldBtn("end",   endMin,   "End time")}
          </div>
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
            <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <span className="cc-label">Time range</span>
              <div style={{ flex: 1 }}/>
              <span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 12, color: "var(--cc-fg-2)", fontVariantNumeric: "tabular-nums" }}>
                {window.fmtMin12(startMin)} → {window.fmtMin12(endMin)}
              </span>
              <button onClick={() => setWeekView(w => !w)} style={{
                display: "inline-flex", alignItems: "center", gap: 4,
                padding: "3px 8px", border: "1px solid var(--cc-border-strong)",
                background: weekView ? "var(--cc-surface-2)" : "var(--cc-surface)",
                borderRadius: 4, fontSize: 11, fontFamily: "var(--cc-font-mono)",
                color: "var(--cc-fg-2)", cursor: "pointer",
              }}>
                {weekView ? window.Icon.rows(11) : window.Icon.grid(11)}
                {weekView ? "Single day" : "Week view"}
              </button>
            </div>

            {!weekView ? (
              <DayTimeline
                startMin={startMin} endMin={endMin}
                onChange={({ start, end }) => { setStartMin(start); setEndMin(end); }}
                existingBlocks={defaultBandBlocks}
              />
            ) : (
              <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
                {DAY_NAMES.map((dn, i) => {
                  const isSelectedDay = !!days[i];
                  const otherBlocks = existingBlocksForDay(i);
                  return (
                    <div key={i} style={{ display: "grid", gridTemplateColumns: "32px 1fr", gap: 8, alignItems: "center" }}>
                      <div style={{
                        fontFamily: "var(--cc-font-mono)", fontSize: 11,
                        color: isSelectedDay ? "var(--cc-fg)" : "var(--cc-fg-4)",
                        fontWeight: isSelectedDay ? 600 : 400,
                      }}>{dn}</div>
                      <div style={{ position: "relative", height: 28,
                                    background: "var(--cc-surface-2)",
                                    border: "1px solid var(--cc-border)", borderRadius: 4,
                                    opacity: isSelectedDay ? 1 : 0.55, overflow: "hidden" }}>
                        {/* ticks */}
                        {[0,6,12,18,24].map(h => (
                          <div key={h} style={{ position: "absolute", left: `${(h/24)*100}%`, top: 0, bottom: 0, width: 1,
                                                 background: "var(--cc-divider)" }}/>
                        ))}
                        {/* existing */}
                        {otherBlocks.map((b, j) => (
                          <div key={j} style={{
                            position: "absolute", left: `${(b.start/(24*60))*100}%`,
                            width: `${((b.end - b.start)/(24*60))*100}%`,
                            top: 3, bottom: 3,
                            background: "rgba(120, 118, 111, 0.22)",
                            border: "1px solid rgba(120, 118, 111, 0.45)",
                            borderRadius: 2,
                          }}/>
                        ))}
                        {/* this period as preview block on selected days */}
                        {isSelectedDay && (
                          <div style={{
                            position: "absolute", left: `${(startMin/(24*60))*100}%`,
                            width: `${((endMin - startMin)/(24*60))*100}%`,
                            top: 2, bottom: 2,
                            background: "var(--cc-heat-tint)", border: "1.5px solid var(--cc-heat)",
                            borderRadius: 3,
                          }}/>
                        )}
                      </div>
                    </div>
                  );
                })}
                <div style={{ fontFamily: "var(--cc-font-mono)", fontSize: 10, color: "var(--cc-fg-3)", textAlign: "right", marginTop: 2 }}>
                  Drag the handles in single-day view to adjust the time range.
                </div>
              </div>
            )}
          </div>
        )}

        {/* Targets */}
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 14 }}>
          {room.hasTemp && (
            <window.Field label="Target temperature">
              <window.InputUnit mono value={tempC} onChange={e => setTempC(parseFloat(e.target.value) || 0)} unit="°C"/>
            </window.Field>
          )}
          {room.hasHum && (
            <window.Field label="Target humidity">
              <window.InputUnit mono value={humPct} onChange={e => setHumPct(parseFloat(e.target.value) || 0)} unit="%"/>
            </window.Field>
          )}
        </div>

        {!atLeastOneTarget && (
          <div className="cc-meta" style={{ color: "var(--cc-danger-fg)" }}>At least one target must be set.</div>
        )}
      </div>
    </window.Modal>
  );
};

Object.assign(window, { PeriodModal, ClockPicker, DayTimeline });
