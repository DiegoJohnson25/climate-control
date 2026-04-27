/* global React */
const { useState, useRef, useEffect } = React;
const I = window.Icon;

// ---------- Button, Input, Select, Card, Field ---------------------------
const Button = ({ variant = "primary", size = "md", icon, iconRight, children, ...rest }) => {
  const cls = `cc-btn cc-btn--${variant}` + (size !== "md" ? ` cc-btn--${size}` : "");
  return <button className={cls} {...rest}>{icon}{children}{iconRight}</button>;
};

const IconBtn = ({ children, danger, active, title, ...rest }) => (
  <button
    className={"cc-iconbtn" + (danger ? " cc-iconbtn--danger" : "") + (active ? " cc-iconbtn--active" : "")}
    title={title} {...rest}
  >{children}</button>
);

const Input = ({ mono = false, ...rest }) => (
  <input className={"cc-input" + (mono ? " cc-input--mono" : "")} {...rest} />
);

const InputUnit = ({ value, onChange, unit, mono = true, style = {}, disabled, ...rest }) => (
  <span className="cc-input-unit" style={{ width: style.width || 120, ...style }}>
    <Input mono={mono} value={value} onChange={onChange} disabled={disabled} {...rest}/>
    <span className="unit">{unit}</span>
  </span>
);

const Select = ({ value, onChange, children, mono = false, style, ...rest }) => (
  <select value={value} onChange={onChange} {...rest}
    className={"cc-input" + (mono ? " cc-input--mono" : "")} style={{ paddingRight: 28, ...style }}>
    {children}
  </select>
);

const Card = ({ children, style, ...rest }) => (
  <div className="cc-card" style={style} {...rest}>{children}</div>
);

const Field = ({ label, hint, children, style }) => (
  <label style={{ display: "flex", flexDirection: "column", gap: 6, ...style }}>
    <span className="cc-label">{label}</span>
    {children}
    {hint && <span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 11, color: "var(--cc-fg-3)" }}>{hint}</span>}
  </label>
);

// ---------- Badge --------------------------------------------------------
const Badge = ({ variant, children, dot = false }) => (
  <span className={"cc-badge" + (variant ? ` cc-badge--${variant}` : "")}>
    {dot && <span className="cc-dot"/>}{children}
  </span>
);

// ---------- Readout ------------------------------------------------------
// Plays a 600ms `cc-pulse` fade (--cc-dur-fade, --cc-ease) whenever `value`
// changes to a non-null value. Matches the v1 "live reading" behaviour.
const Readout = ({ value, unit, tone, size = "lg", decimals = 1 }) => {
  const [live, setLive] = useState(false);
  const prevRef = useRef(value);
  useEffect(() => {
    if (value != null && prevRef.current !== value) {
      setLive(true);
      const id = setTimeout(() => setLive(false), 600);
      prevRef.current = value;
      return () => clearTimeout(id);
    }
    prevRef.current = value;
  }, [value]);

  const color = tone === "heat" ? "var(--cc-heat)" : tone === "cool" ? "var(--cc-cool)" : "var(--cc-fg)";
  const baseCls = size === "lg" ? "cc-readout" : "cc-readout-sm";
  const cls = baseCls + (live ? " cc-readout--live" : "");

  if (value == null) {
    return <div className={baseCls} style={{ color: "var(--cc-fg-4)" }}>—</div>;
  }
  const formatted = typeof value === "number" ? value.toFixed(decimals) : value;
  return (
    <div className={cls} style={{ color }}>
      {formatted}<span style={{ color: "var(--cc-fg-3)", fontWeight: 400, marginLeft: 2, fontSize: "0.45em" }}>{unit}</span>
    </div>
  );
};

// ---------- Segmented ----------------------------------------------------
const Segmented = ({ value, onChange, options, disabled, size }) => (
  <div className={"cc-seg" + (disabled ? " cc-seg--disabled" : "")}>
    {options.map(o => (
      <button key={o.value} className={value === o.value ? "is-on" : ""} onClick={() => !disabled && onChange(o.value)}>
        {o.label}
      </button>
    ))}
  </div>
);

// ---------- ToggleDot ----------------------------------------------------
const ToggleDot = ({ on, onClick, disabled, title }) => (
  <button
    className={"cc-togdot" + (on ? " cc-togdot--on" : "") + (disabled ? " cc-togdot--disabled" : "")}
    onClick={disabled ? undefined : onClick}
    title={title}
    aria-pressed={on}
  />
);

// ---------- Chip ---------------------------------------------------------
const Chip = ({ on, onClick, children, disabled, size }) => (
  <button className={"cc-chip" + (on ? " cc-chip--on" : "")} onClick={onClick} disabled={disabled}>
    {children}
  </button>
);

// ---------- Day chip -----------------------------------------------------
const DayChip = ({ label, on, onClick, readonly }) => (
  <button
    className={"cc-daychip" + (on ? " cc-daychip--on" : "") + (readonly ? " cc-daychip--read" : "")}
    onClick={readonly ? undefined : onClick}
    disabled={readonly}
  >{label}</button>
);

// ---------- Deadband pill ------------------------------------------------
const DeadbandPill = ({ value, unit, onClick }) => (
  <button className="cc-dbpill" onClick={onClick} title="Edit tolerances">
    ±{value}{unit}
  </button>
);

// ---------- Tooltip ------------------------------------------------------
const Tooltip = ({ text, children, forceShow = false }) => {
  const [show, setShow] = useState(forceShow);
  const [pos, setPos] = useState({ x: 0, y: 0 });
  const ref = useRef(null);
  useEffect(() => setShow(forceShow), [forceShow]);

  const onEnter = (e) => {
    const r = e.currentTarget.getBoundingClientRect();
    setPos({ x: r.left + r.width / 2, y: r.top });
    setShow(true);
  };
  const onLeave = () => !forceShow && setShow(false);

  return (
    <>
      <span ref={ref} onMouseEnter={onEnter} onMouseLeave={onLeave} style={{ display: "inline-flex" }}>
        {children}
      </span>
      {show && ReactDOM.createPortal(
        <div className="cc-tt" style={{ left: pos.x, top: pos.y - 8, transform: "translate(-50%, -100%)" }}>{text}</div>,
        document.body
      )}
    </>
  );
};

// ---------- KebabMenu ----------------------------------------------------
const KebabMenu = ({ items, align = "right" }) => {
  const [open, setOpen] = useState(false);
  const ref = useRef(null);
  useEffect(() => {
    const onDoc = (e) => ref.current && !ref.current.contains(e.target) && setOpen(false);
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, []);
  return (
    <div ref={ref} style={{ position: "relative", display: "inline-block" }}>
      <IconBtn onClick={() => setOpen(o => !o)} active={open} title="More actions">{I.kebab()}</IconBtn>
      {open && (
        <div className="cc-pop" style={{ top: "calc(100% + 4px)", [align]: 0 }}>
          {items.map((it, i) =>
            it.divider ? <hr key={i}/> :
            <button key={i} className={it.danger ? "danger" : ""} onClick={() => { setOpen(false); it.onClick?.(); }}>{it.label}</button>
          )}
        </div>
      )}
    </div>
  );
};

// ---------- Modal --------------------------------------------------------
const Modal = ({ open, onClose, title, subtitle, children, footer, width = 460 }) => {
  useEffect(() => {
    if (!open) return;
    const onKey = (e) => e.key === "Escape" && onClose?.();
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onClose]);
  if (!open) return null;
  return ReactDOM.createPortal(
    <div className="cc-modal-bg" onClick={onClose}>
      <div className="cc-modal" style={{ width }} onClick={e => e.stopPropagation()}>
        <div className="cc-modal-head">
          <div style={{ flex: 1 }}>
            <div style={{ fontSize: 15, fontWeight: 600, letterSpacing: "-0.01em" }}>{title}</div>
            {subtitle && <div className="cc-meta" style={{ marginTop: 4 }}>{subtitle}</div>}
          </div>
          <IconBtn onClick={onClose} title="Close">{I.x()}</IconBtn>
        </div>
        <div className="cc-modal-body">{children}</div>
        {footer && <div className="cc-modal-foot">{footer}</div>}
      </div>
    </div>,
    document.body
  );
};

// ---------- time helpers -------------------------------------------------
const timeAgo = (iso) => {
  if (!iso) return "—";
  const ms = Date.now() - new Date(iso).getTime();
  const s = Math.floor(ms / 1000);
  if (s < 60) return `${s}s ago`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
};

// fmtTime12("14:30") → "2:30pm", fmtTime12("09:00") → "9am", fmtTime12("24:00") → "12am"
const fmtTime12 = (hm) => {
  if (!hm) return "";
  if (hm === "24:00") return "12am";
  const [hStr, mStr] = hm.split(":");
  const h24 = parseInt(hStr, 10);
  const m = parseInt(mStr, 10);
  const ampm = h24 >= 12 ? "pm" : "am";
  let h12 = h24 % 12; if (h12 === 0) h12 = 12;
  return m === 0 ? `${h12}${ampm}` : `${h12}:${String(m).padStart(2, "0")}${ampm}`;
};

// fmtMin12(540) → "9am", fmtMin12(870) → "2:30pm"
const fmtMin12 = (minOfDay) => {
  if (minOfDay == null) return "";
  if (minOfDay >= 24 * 60) return "12am";
  const h24 = Math.floor(minOfDay / 60);
  const m = minOfDay % 60;
  const ampm = h24 >= 12 ? "pm" : "am";
  let h12 = h24 % 12; if (h12 === 0) h12 = 12;
  return m === 0 ? `${h12}${ampm}` : `${h12}:${String(m).padStart(2, "0")}${ampm}`;
};

// fmtTick12(ts, "24h") → "6am" | "Mon 6am" (for 7d). User local time.
// `tz` is accepted for parity with the real API but not applied in the mockup —
// production code should route through Intl.DateTimeFormat with timeZone.
const fmtTick12 = (t, windowKey, _tz) => {
  const d = new Date(t);
  const h24 = d.getHours();
  const m = d.getMinutes();
  const ampm = h24 >= 12 ? "pm" : "am";
  let h12 = h24 % 12; if (h12 === 0) h12 = 12;
  const time = m === 0 ? `${h12}${ampm}` : `${h12}:${String(m).padStart(2, "0")}${ampm}`;
  if (windowKey === "7d") {
    const weekday = d.toLocaleDateString(undefined, { weekday: "short" });
    return `${weekday} ${time}`;
  }
  return time;
};

Object.assign(window, {
  Button, IconBtn, Input, InputUnit, Select, Card, Field, Badge, Readout,
  Segmented, ToggleDot, Chip, DayChip, DeadbandPill, Tooltip, KebabMenu, Modal,
  timeAgo, fmtTime12, fmtMin12, fmtTick12,
});
