/* global React */
const { useState } = React;

// Room detail container — tab strip, header with edit/kebab, active tab
const RoomDetail = ({
  room, rooms, schedules, devices, tz,
  initialTab = "overview",
  onBack, onEditName, onDeleteRoom,
  onOpenTolerances, onApplyControl, onRevertControl,
  onRegisterDevice, onEditDevice, onDeleteDevice, onGoToGlobalDevices,
  onAddSchedule, onEditSchedule, onDeleteSchedule, onToggleSchedule,
  onAddPeriod, onEditPeriod, onDeletePeriod,
  _forceTab,
}) => {
  const [tab, setTab] = useState(_forceTab || initialTab);

  const TABS = [
    { key: "overview",  label: "Overview" },
    { key: "history",   label: "History" },
    { key: "schedules", label: "Schedules" },
    { key: "devices",   label: "Devices" },
  ];

  return (
    <div style={{ maxWidth: 1280, margin: "0 auto", padding: "24px 24px 48px" }}>
      {/* Back link */}
      <button onClick={onBack} style={{
        background: "transparent", border: "none", cursor: "pointer", padding: 0,
        display: "inline-flex", alignItems: "center", gap: 5,
        fontSize: 12, color: "var(--cc-fg-3)", marginBottom: 16,
      }}>
        {window.Icon.chevronLeft(12)} Dashboard
      </button>

      {/* Header */}
      <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 22 }}>
        <h1 className="cc-h1" style={{ fontFamily: "var(--cc-font-sans)" }}>{room.name}</h1>
        <window.IconBtn onClick={onEditName} title="Edit name">{window.Icon.pencil(14)}</window.IconBtn>
        <div style={{ flex: 1 }}/>
        <window.ModeBadge mode={room.mode}/>
        <window.SourceBadge source={room.source}/>
        <window.KebabMenu items={[
          { label: "Edit name", onClick: onEditName },
          { divider: true },
          { label: "Delete room", danger: true, onClick: onDeleteRoom },
        ]}/>
      </div>

      {/* Tab strip */}
      <div style={{ borderBottom: "1px solid var(--cc-border)", display: "flex", gap: 2, marginBottom: 22 }}>
        {TABS.map(t => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            style={{
              background: "transparent", border: "none", cursor: "pointer",
              padding: "10px 14px", position: "relative",
              fontSize: 13, fontWeight: 500,
              color: tab === t.key ? "var(--cc-fg)" : "var(--cc-fg-3)",
            }}
          >
            {t.label}
            {tab === t.key && <div style={{
              position: "absolute", left: 10, right: 10, bottom: -1, height: 2,
              background: "var(--cc-fg)",
            }}/>}
          </button>
        ))}
      </div>

      {/* Active tab */}
      {tab === "overview" && (
        <window.OverviewTab
          room={room}
          schedules={schedules}
          onOpenTolerances={onOpenTolerances}
          onApply={onApplyControl}
          onRevert={onRevertControl}
        />
      )}
      {tab === "history" && <window.HistoryTab room={room} tz={tz}/>}
      {tab === "schedules" && (
        <window.SchedulesTab
          room={room}
          schedules={schedules}
          onAddSchedule={onAddSchedule}
          onEditSchedule={onEditSchedule}
          onDeleteSchedule={onDeleteSchedule}
          onToggleActive={onToggleSchedule}
          onAddPeriod={onAddPeriod}
          onEditPeriod={onEditPeriod}
          onDeletePeriod={onDeletePeriod}
        />
      )}
      {tab === "devices" && (
        <window.DevicesTab
          room={room}
          devices={devices}
          onRegister={onRegisterDevice}
          onEditDevice={onEditDevice}
          onDeleteDevice={onDeleteDevice}
          onGoToGlobal={onGoToGlobalDevices}
        />
      )}
    </div>
  );
};

Object.assign(window, { RoomDetail });
