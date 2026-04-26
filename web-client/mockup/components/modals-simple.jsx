/* global React */
const { useState } = React;

// ============================================================
// Simple modals: rooms, devices, schedules, tolerances, account
// ============================================================

// ---------- Room modals ----------
const AddRoomModal = ({ open, onClose, onSave }) => {
  const [name, setName] = useState("");
  React.useEffect(() => { if (open) setName(""); }, [open]);
  return (
    <window.Modal open={open} onClose={onClose} title="Add room"
      footer={<><window.Button variant="ghost" onClick={onClose}>Cancel</window.Button>
               <window.Button onClick={() => { onSave?.(name); onClose(); }} disabled={!name.trim()}>Save</window.Button></>}>
      <window.Field label="Name"><window.Input value={name} onChange={e => setName(e.target.value)} placeholder="e.g. Living Room" autoFocus/></window.Field>
    </window.Modal>
  );
};

const EditRoomNameModal = ({ open, onClose, room, onSave }) => {
  const [name, setName] = useState(room?.name || "");
  React.useEffect(() => { if (open && room) setName(room.name); }, [open, room]);
  return (
    <window.Modal open={open} onClose={onClose} title="Edit room name"
      footer={<><window.Button variant="ghost" onClick={onClose}>Cancel</window.Button>
               <window.Button onClick={() => { onSave?.(name); onClose(); }} disabled={!name.trim()}>Save</window.Button></>}>
      <window.Field label="Name"><window.Input value={name} onChange={e => setName(e.target.value)} autoFocus/></window.Field>
    </window.Modal>
  );
};

const DeleteRoomModal = ({ open, onClose, room, onConfirm }) => (
  <window.Modal open={open} onClose={onClose} title="Delete room" subtitle="This action cannot be undone"
    footer={<><window.Button variant="ghost" onClick={onClose}>Cancel</window.Button>
             <window.Button variant="danger" onClick={() => { onConfirm?.(); onClose(); }}>Delete</window.Button></>}>
    <div className="cc-body" style={{ color: "var(--cc-fg-2)" }}>
      This will unassign all devices and delete all schedules for <strong style={{ color: "var(--cc-fg)" }}>{room?.name}</strong>. This cannot be undone.
    </div>
  </window.Modal>
);

// ---------- Device modals ----------
const RegisterDeviceModal = ({ open, onClose, rooms, onSave }) => {
  const [hwId, setHwId] = useState("");
  const [name, setName] = useState("");
  const [roomId, setRoomId] = useState("");
  React.useEffect(() => { if (open) { setHwId(""); setName(""); setRoomId(""); } }, [open]);
  return (
    <window.Modal open={open} onClose={onClose} title="Register device" subtitle="Associate a hardware controller with your account"
      footer={<><window.Button variant="ghost" onClick={onClose}>Cancel</window.Button>
               <window.Button onClick={() => { onSave?.({ hwId, name, roomId: roomId || null }); onClose(); }} disabled={!hwId.trim() || !name.trim()}>Register</window.Button></>}>
      <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
        <window.Field label="Hardware ID" hint="Found on the device label or QR code">
          <window.Input mono value={hwId} onChange={e => setHwId(e.target.value)} placeholder="8c:1f:64:a2:3d:00"/>
        </window.Field>
        <window.Field label="Display name">
          <window.Input value={name} onChange={e => setName(e.target.value)} placeholder="e.g. Living Room Sensor"/>
        </window.Field>
        <window.Field label="Room" hint="Optional — you can assign later from the Devices page">
          <window.Select value={roomId} onChange={e => setRoomId(e.target.value)}>
            <option value="">Unassigned</option>
            {rooms.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
          </window.Select>
        </window.Field>
      </div>
    </window.Modal>
  );
};

const EditDeviceModal = ({ open, onClose, device, rooms, onSave }) => {
  const [name, setName] = useState(device?.name || "");
  const [roomId, setRoomId] = useState(device?.roomId || "");
  React.useEffect(() => { if (open && device) { setName(device.name); setRoomId(device.roomId || ""); } }, [open, device]);
  return (
    <window.Modal open={open} onClose={onClose} title="Edit device"
      subtitle={device ? <span style={{ fontFamily: "var(--cc-font-mono)" }}>{device.hwId}</span> : null}
      footer={<><window.Button variant="ghost" onClick={onClose}>Cancel</window.Button>
               <window.Button onClick={() => { onSave?.({ name, roomId: roomId || null }); onClose(); }} disabled={!name.trim()}>Save</window.Button></>}>
      <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
        <window.Field label="Display name"><window.Input value={name} onChange={e => setName(e.target.value)}/></window.Field>
        <window.Field label="Room assignment">
          <window.Select value={roomId} onChange={e => setRoomId(e.target.value)}>
            <option value="">Unassigned</option>
            {rooms.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
          </window.Select>
        </window.Field>
      </div>
    </window.Modal>
  );
};

const DeleteDeviceModal = ({ open, onClose, device, onConfirm }) => (
  <window.Modal open={open} onClose={onClose} title="Delete device" subtitle="This action cannot be undone"
    footer={<><window.Button variant="ghost" onClick={onClose}>Cancel</window.Button>
             <window.Button variant="danger" onClick={() => { onConfirm?.(); onClose(); }}>Delete</window.Button></>}>
    <div className="cc-body" style={{ color: "var(--cc-fg-2)" }}>
      This will permanently remove <strong style={{ color: "var(--cc-fg)" }}>{device?.name}</strong> from the system.
    </div>
  </window.Modal>
);

// ---------- Schedule modals ----------
const AddScheduleModal = ({ open, onClose, onSave }) => {
  const [name, setName] = useState("");
  React.useEffect(() => { if (open) setName(""); }, [open]);
  return (
    <window.Modal open={open} onClose={onClose} title="Add schedule"
      footer={<><window.Button variant="ghost" onClick={onClose}>Cancel</window.Button>
               <window.Button onClick={() => { onSave?.(name); onClose(); }} disabled={!name.trim()}>Save</window.Button></>}>
      <window.Field label="Name"><window.Input value={name} onChange={e => setName(e.target.value)} placeholder="e.g. Weekday Morning" autoFocus/></window.Field>
    </window.Modal>
  );
};

const EditScheduleModal = ({ open, onClose, schedule, onSave }) => {
  const [name, setName] = useState(schedule?.name || "");
  React.useEffect(() => { if (open && schedule) setName(schedule.name); }, [open, schedule]);
  return (
    <window.Modal open={open} onClose={onClose} title="Edit schedule"
      footer={<><window.Button variant="ghost" onClick={onClose}>Cancel</window.Button>
               <window.Button onClick={() => { onSave?.(name); onClose(); }} disabled={!name.trim()}>Save</window.Button></>}>
      <window.Field label="Name"><window.Input value={name} onChange={e => setName(e.target.value)} autoFocus/></window.Field>
    </window.Modal>
  );
};

const DeleteScheduleModal = ({ open, onClose, schedule, onConfirm }) => (
  <window.Modal open={open} onClose={onClose} title="Delete schedule" subtitle="This action cannot be undone"
    footer={<><window.Button variant="ghost" onClick={onClose}>Cancel</window.Button>
             <window.Button variant="danger" onClick={() => { onConfirm?.(); onClose(); }}>Delete</window.Button></>}>
    <div className="cc-body" style={{ color: "var(--cc-fg-2)" }}>
      This will delete <strong style={{ color: "var(--cc-fg)" }}>{schedule?.name}</strong> and all its periods.
    </div>
  </window.Modal>
);

// ---------- Tolerances modal ----------
const TolerancesModal = ({ open, onClose, room, onSave }) => {
  const [tempDb, setTempDb] = useState(room?.desired?.tempDb || 0.5);
  const [humDb, setHumDb] = useState(room?.desired?.humDb || 3.0);
  React.useEffect(() => { if (open && room) { setTempDb(room.desired.tempDb ?? 0.5); setHumDb(room.desired.humDb ?? 3.0); } }, [open, room]);
  const tempTarget = room?.desired?.tempC ?? 21;
  const humTarget = room?.desired?.humPct ?? 50;

  return (
    <window.Modal open={open} onClose={onClose} title="Tolerances"
      subtitle="Wider tolerances save energy but allow more drift"
      footer={<><window.Button variant="ghost" onClick={onClose}>Cancel</window.Button>
               <window.Button onClick={() => { onSave?.({ tempDb, humDb }); onClose(); }}>Save</window.Button></>}>
      <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
        {room?.hasTemp && (
          <window.Field label="Temperature tolerance"
            hint={`Heater turns on below ${(tempTarget - tempDb).toFixed(1)}°C, off above ${(tempTarget + tempDb).toFixed(1)}°C`}>
            <window.InputUnit value={tempDb} onChange={e => setTempDb(parseFloat(e.target.value) || 0)} unit="°C" style={{ width: 140 }}/>
          </window.Field>
        )}
        {room?.hasHum && (
          <window.Field label="Humidity tolerance"
            hint={`Humidifier turns on below ${(humTarget - humDb).toFixed(1)}%, off above ${(humTarget + humDb).toFixed(1)}%`}>
            <window.InputUnit value={humDb} onChange={e => setHumDb(parseFloat(e.target.value) || 0)} unit="%" style={{ width: 140 }}/>
          </window.Field>
        )}
      </div>
    </window.Modal>
  );
};

// ---------- Account settings ----------
const TIMEZONES = [
  "America/Los_Angeles", "America/Denver", "America/Chicago", "America/New_York",
  "Europe/London", "Europe/Berlin", "Europe/Madrid", "Asia/Tokyo", "Asia/Singapore",
  "Australia/Sydney", "UTC",
];

const AccountSettingsModal = ({ open, onClose, account, onSave }) => {
  const [tz, setTz] = useState(account?.timezone || "UTC");
  React.useEffect(() => { if (open && account) setTz(account.timezone); }, [open, account]);
  return (
    <window.Modal open={open} onClose={onClose} title="Account settings"
      footer={<><window.Button variant="ghost" onClick={onClose}>Cancel</window.Button>
               <window.Button onClick={() => { onSave?.({ timezone: tz }); onClose(); }}>Save</window.Button></>}>
      <window.Field label="Timezone" hint="Used for schedule display and evaluation">
        <window.Select value={tz} onChange={e => setTz(e.target.value)}>
          {TIMEZONES.map(z => <option key={z} value={z}>{z}</option>)}
        </window.Select>
      </window.Field>
    </window.Modal>
  );
};

// ---------- Delete account ----------
const DeleteAccountModal = ({ open, onClose, account, onConfirm }) => {
  const [typed, setTyped] = useState("");
  React.useEffect(() => { if (open) setTyped(""); }, [open]);
  const matches = typed === account?.email;
  return (
    <window.Modal open={open} onClose={onClose} title="Delete account" subtitle="Permanent and irreversible"
      footer={<><window.Button variant="ghost" onClick={onClose}>Cancel</window.Button>
               <window.Button variant="danger" onClick={() => { onConfirm?.(); onClose(); }} disabled={!matches}>Delete account</window.Button></>}>
      <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
        <div className="cc-body" style={{ color: "var(--cc-fg-2)" }}>
          All your data will be permanently deleted. This cannot be undone.
        </div>
        <window.Field label="Type your email to confirm" hint={<><span style={{ fontFamily: "var(--cc-font-mono)" }}>{account?.email}</span></>}>
          <window.Input value={typed} onChange={e => setTyped(e.target.value)} placeholder={account?.email}/>
        </window.Field>
      </div>
    </window.Modal>
  );
};

Object.assign(window, {
  AddRoomModal, EditRoomNameModal, DeleteRoomModal,
  RegisterDeviceModal, EditDeviceModal, DeleteDeviceModal,
  AddScheduleModal, EditScheduleModal, DeleteScheduleModal,
  TolerancesModal, AccountSettingsModal, DeleteAccountModal,
});
