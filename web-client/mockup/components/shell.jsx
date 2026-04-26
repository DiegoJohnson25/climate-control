/* global React */
const { useState, useRef, useEffect } = React;

// ============================================================
// TopNav — 56px sticky header
// ============================================================
const TopNav = ({ page, onNav, account, onOpenAccount, onOpenDeleteAccount, onLogout, forceMenuOpen = false }) => {
  const [menuOpen, setMenuOpen] = useState(forceMenuOpen);
  const menuRef = useRef(null);
  useEffect(() => setMenuOpen(forceMenuOpen), [forceMenuOpen]);
  useEffect(() => {
    if (!menuOpen || forceMenuOpen) return;
    const onDoc = (e) => menuRef.current && !menuRef.current.contains(e.target) && setMenuOpen(false);
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [menuOpen, forceMenuOpen]);

  const NavLink = ({ name, label }) => {
    const active = page === name;
    return (
      <button
        onClick={() => onNav(name)}
        style={{
          background: "transparent", border: "none", cursor: "pointer",
          padding: "0 2px", height: 56, position: "relative",
          fontSize: 13, fontWeight: 500, letterSpacing: "-0.005em",
          color: active ? "var(--cc-fg)" : "var(--cc-fg-3)",
        }}
      >
        {label}
        {active && <div style={{
          position: "absolute", left: 0, right: 0, bottom: 0, height: 2,
          background: "var(--cc-fg)",
        }}/>}
      </button>
    );
  };

  return (
    <div style={{
      position: "sticky", top: 0, zIndex: 30, height: 56,
      borderBottom: "1px solid var(--cc-border)", background: "var(--cc-surface)",
    }}>
      <div style={{
        maxWidth: 1280, margin: "0 auto", height: "100%", padding: "0 24px",
        display: "flex", alignItems: "center", gap: 28,
      }}>
        <button onClick={() => onNav("dashboard")}
          style={{ background: "none", border: "none", cursor: "pointer", padding: 0, display: "flex", alignItems: "center", gap: 8 }}>
          <div style={{
            width: 22, height: 22, borderRadius: 5, background: "var(--cc-primary)",
            display: "flex", alignItems: "center", justifyContent: "center",
            color: "var(--cc-primary-fg)",
          }}>{window.Icon.thermometer(13)}</div>
          <span style={{ fontSize: 14, fontWeight: 600, letterSpacing: "-0.01em" }}>Climate Control</span>
        </button>

        <div style={{ display: "flex", gap: 24, marginLeft: 8 }}>
          <NavLink name="dashboard" label="Dashboard"/>
          <NavLink name="devices" label="Devices"/>
        </div>

        <div style={{ flex: 1 }}/>

        <div ref={menuRef} style={{ position: "relative" }}>
          <button
            onClick={() => setMenuOpen(o => !o)}
            style={{
              display: "flex", alignItems: "center", gap: 8,
              background: menuOpen ? "var(--cc-surface-2)" : "transparent",
              border: "1px solid " + (menuOpen ? "var(--cc-border-strong)" : "transparent"),
              padding: "4px 10px 4px 4px", borderRadius: 999, cursor: "pointer",
            }}
          >
            <div style={{
              width: 26, height: 26, borderRadius: "50%",
              background: "linear-gradient(135deg, #D97706 0%, #0891B2 100%)",
              color: "#fff", display: "flex", alignItems: "center", justifyContent: "center",
              fontSize: 11, fontWeight: 600, fontFamily: "var(--cc-font-mono)",
            }}>{account.email[0].toUpperCase()}</div>
            <span style={{ fontFamily: "var(--cc-font-mono)", fontSize: 12, color: "var(--cc-fg-2)" }}>{account.email}</span>
            {window.Icon.chevronDown(14)}
          </button>
          {menuOpen && (
            <div className="cc-pop" style={{ position: "absolute", top: "calc(100% + 6px)", right: 0, width: 240 }}>
              <div style={{ padding: "8px 10px 10px" }}>
                <div className="cc-meta">Signed in as</div>
                <div style={{ fontFamily: "var(--cc-font-mono)", fontSize: 12, color: "var(--cc-fg)", marginTop: 2 }}>{account.email}</div>
              </div>
              <hr/>
              <button onClick={() => { setMenuOpen(false); onOpenAccount?.(); }}>Account settings</button>
              <hr/>
              <button onClick={() => { setMenuOpen(false); onLogout?.(); }}>Log out</button>
              <hr/>
              <button className="danger" onClick={() => { setMenuOpen(false); onOpenDeleteAccount?.(); }}>Delete account</button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

// ============================================================
// Login — centered card, nothing else above the card. A mono
// dev-hint footer below is retained as a neutral seed-credential
// indicator for development.
// ============================================================
const Login = ({ onSignIn }) => {
  const [email, setEmail] = useState("operator@local.dev");
  const [pw, setPw] = useState("••••••••••");

  return (
    <div style={{
      minHeight: "100vh", background: "var(--cc-bg)",
      display: "flex", alignItems: "center", justifyContent: "center", padding: 24,
    }}>
      <div style={{ width: 360 }}>
        <div className="cc-card" style={{ padding: 28 }}>
          <div style={{ marginBottom: 20 }}>
            <div style={{ fontSize: 17, fontWeight: 600, letterSpacing: "-0.01em" }}>Sign in</div>
          </div>

          <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
            <window.Field label="Email">
              <window.Input value={email} onChange={e => setEmail(e.target.value)} type="email"/>
            </window.Field>
            <window.Field label="Password">
              <window.Input value={pw} onChange={e => setPw(e.target.value)} type="password"/>
            </window.Field>
            <window.Button onClick={onSignIn} style={{ marginTop: 6, height: 36 }}>Sign in</window.Button>
          </div>
        </div>

        <div style={{
          marginTop: 16, padding: "10px 14px",
          fontFamily: "var(--cc-font-mono)", fontSize: 11, color: "var(--cc-fg-3)",
          textAlign: "center", lineHeight: 1.5,
        }}>
          dev build · seed credentials active · <span style={{ color: "var(--cc-warning-fg)" }}>hide in production</span>
        </div>
      </div>
    </div>
  );
};

Object.assign(window, { TopNav, Login });
