/* global React */
const { useState } = React;

const App = () => {
  const [page, setPage] = useState("dashboard");       // login | dashboard | room | devices
  const [currentRoomId, setCurrentRoomId] = useState(null);
  const [initialTab, setInitialTab] = useState("overview");

  // Live mutable state (shallow clone of seed data)
  const [rooms, setRooms] = useState(() => [...window.CCData.ROOMS]);
  const [devices, setDevices] = useState(() => [...window.CCData.DEVICES]);
  const [schedules, setSchedules] = useState(() => [...window.CCData.SCHEDULES]);
  const [account, setAccount] = useState(() => ({ ...window.CCData.ACCOUNT }));

  // Modal state — single active modal at a time
  const [modal, setModal] = useState(null); // { kind, ctx? }
  const open = (kind, ctx) => setModal({ kind, ctx });
  const close = () => setModal(null);

  const currentRoom = rooms.find(r => r.id === currentRoomId);

  // --- Actions -----------------------------------------------
  const actions = {
    addRoom: (name) => {
      const id = "r-" + Math.random().toString(36).slice(2, 7);
      setRooms(rs => [...rs, {
        id, name, hasTemp: true, hasHum: true, mode: "off", source: "none",
        activeScheduleId: null, tempC: null, humPct: null,
        tempUpdated: null, humUpdated: null, heaterOn: false, humOn: false,
        targets: { tempC: 22, humPct: 55, tempDb: 0.5, humDb: 3 },
        desired: { tempOn: true, humOn: true, tempC: 22, humPct: 55, tempDb: 0.5, humDb: 3 },
        hold: { on: false, duration: "1h" },
      }]);
    },
    editRoomName: (id, name) => setRooms(rs => rs.map(r => r.id === id ? { ...r, name } : r)),
    deleteRoom: (id) => {
      setRooms(rs => rs.filter(r => r.id !== id));
      setDevices(ds => ds.map(d => d.roomId === id ? { ...d, roomId: null } : d));
      setSchedules(ss => ss.filter(s => s.roomId !== id));
      setPage("dashboard");
    },
    registerDevice: ({ hwId, name, roomId }) => {
      const id = "d-" + Math.random().toString(36).slice(2, 7);
      setDevices(ds => [...ds, { id, hwId, name, roomId, sensors: [], actuators: [] }]);
    },
    editDevice: (id, { name, roomId }) => setDevices(ds => ds.map(d => d.id === id ? { ...d, name, roomId } : d)),
    deleteDevice: (id) => setDevices(ds => ds.filter(d => d.id !== id)),
    changeDeviceRoom: (id, roomId) => setDevices(ds => ds.map(d => d.id === id ? { ...d, roomId } : d)),

    addSchedule: (roomId, name) => {
      const id = "s-" + Math.random().toString(36).slice(2, 7);
      setSchedules(ss => [...ss, { id, name, roomId, active: false, periods: [] }]);
    },
    editSchedule: (id, name) => setSchedules(ss => ss.map(s => s.id === id ? { ...s, name } : s)),
    deleteSchedule: (id) => setSchedules(ss => ss.filter(s => s.id !== id)),
    toggleSchedule: (id) => setSchedules(ss => ss.map(s => s.id === id ? { ...s, active: !s.active } : s)),

    addPeriod: (schedId, period) => setSchedules(ss => ss.map(s => s.id === schedId ? {
      ...s, periods: [...s.periods, { ...period, id: "p-" + Math.random().toString(36).slice(2, 7) }]
    } : s)),
    editPeriod: (schedId, period) => setSchedules(ss => ss.map(s => s.id === schedId ? {
      ...s, periods: s.periods.map(p => p.id === period.id ? { ...p, ...period } : p)
    } : s)),
    deletePeriod: (schedId, periodId) => setSchedules(ss => ss.map(s => s.id === schedId ? {
      ...s, periods: s.periods.filter(p => p.id !== periodId)
    } : s)),

    applyControl: (roomId, { draft, mode, hold }) => setRooms(rs => rs.map(r => r.id === roomId ? {
      ...r, desired: draft, mode, hold,
      source: hold.on ? "manual_override" : (r.activeScheduleId ? "schedule" : "none"),
      targets: { ...r.targets, tempC: draft.tempC, humPct: draft.humPct, tempDb: draft.tempDb, humDb: draft.humDb },
    } : r)),
    saveTolerances: (roomId, { tempDb, humDb }) => setRooms(rs => rs.map(r => r.id === roomId ? {
      ...r, desired: { ...r.desired, tempDb, humDb }, targets: { ...r.targets, tempDb, humDb },
    } : r)),
    saveAccount: ({ timezone }) => setAccount(a => ({ ...a, timezone })),
  };

  // --- Navigation --------------------------------------------
  const openRoom = (id, tab = "overview") => {
    setCurrentRoomId(id);
    setInitialTab(tab);
    setPage("room");
  };

  // --- Render ------------------------------------------------
  if (page === "login") {
    return <div className="cc"><window.Login onSignIn={() => setPage("dashboard")}/></div>;
  }

  return (
    <div className="cc">
      <window.TopNav
        page={page === "room" ? "dashboard" : page}
        onNav={(p) => { if (p === "dashboard") setPage("dashboard"); if (p === "devices") setPage("devices"); }}
        account={account}
        onOpenAccount={() => open("account")}
        onOpenDeleteAccount={() => open("deleteAccount")}
        onLogout={() => setPage("login")}
      />

      {page === "dashboard" && (
        <window.Dashboard
          rooms={rooms}
          onOpenRoom={openRoom}
          onAddRoom={() => open("addRoom")}
        />
      )}

      {page === "room" && currentRoom && (
        <window.RoomDetail
          room={currentRoom} rooms={rooms} schedules={schedules} devices={devices}
          tz={account.timezone}
          initialTab={initialTab}
          onBack={() => setPage("dashboard")}
          onEditName={() => open("editRoomName", currentRoom)}
          onDeleteRoom={() => open("deleteRoom", currentRoom)}
          onOpenTolerances={() => open("tolerances", currentRoom)}
          onApplyControl={(payload) => actions.applyControl(currentRoom.id, payload)}
          onRevertControl={() => {}}
          onRegisterDevice={() => open("registerDevice")}
          onEditDevice={(d) => open("editDevice", d)}
          onDeleteDevice={(d) => open("deleteDevice", d)}
          onGoToGlobalDevices={() => setPage("devices")}
          onAddSchedule={() => open("addSchedule", { roomId: currentRoom.id })}
          onEditSchedule={(s) => open("editSchedule", s)}
          onDeleteSchedule={(s) => open("deleteSchedule", s)}
          onToggleSchedule={(id) => actions.toggleSchedule(id)}
          onAddPeriod={(schedId) => open("period", { mode: "add", schedId })}
          onEditPeriod={(schedId, period) => open("period", { mode: "edit", schedId, period })}
          onDeletePeriod={(schedId, pid) => actions.deletePeriod(schedId, pid)}
        />
      )}

      {page === "devices" && (
        <window.DevicesPage
          devices={devices} rooms={rooms}
          onRegister={() => open("registerDevice")}
          onEditDevice={(d) => open("editDevice", d)}
          onDeleteDevice={(d) => open("deleteDevice", d)}
          onChangeRoom={(id, rid) => actions.changeDeviceRoom(id, rid)}
        />
      )}

      {/* --- Modals --- */}
      <window.AddRoomModal
        open={modal?.kind === "addRoom"} onClose={close}
        onSave={(name) => actions.addRoom(name)}
      />
      <window.EditRoomNameModal
        open={modal?.kind === "editRoomName"} onClose={close} room={modal?.ctx}
        onSave={(name) => actions.editRoomName(modal.ctx.id, name)}
      />
      <window.DeleteRoomModal
        open={modal?.kind === "deleteRoom"} onClose={close} room={modal?.ctx}
        onConfirm={() => actions.deleteRoom(modal.ctx.id)}
      />

      <window.RegisterDeviceModal
        open={modal?.kind === "registerDevice"} onClose={close} rooms={rooms}
        onSave={(payload) => actions.registerDevice(payload)}
      />
      <window.EditDeviceModal
        open={modal?.kind === "editDevice"} onClose={close} device={modal?.ctx} rooms={rooms}
        onSave={(payload) => actions.editDevice(modal.ctx.id, payload)}
      />
      <window.DeleteDeviceModal
        open={modal?.kind === "deleteDevice"} onClose={close} device={modal?.ctx}
        onConfirm={() => actions.deleteDevice(modal.ctx.id)}
      />

      <window.AddScheduleModal
        open={modal?.kind === "addSchedule"} onClose={close}
        onSave={(name) => actions.addSchedule(modal.ctx.roomId, name)}
      />
      <window.EditScheduleModal
        open={modal?.kind === "editSchedule"} onClose={close} schedule={modal?.ctx}
        onSave={(name) => actions.editSchedule(modal.ctx.id, name)}
      />
      <window.DeleteScheduleModal
        open={modal?.kind === "deleteSchedule"} onClose={close} schedule={modal?.ctx}
        onConfirm={() => actions.deleteSchedule(modal.ctx.id)}
      />

      <window.TolerancesModal
        open={modal?.kind === "tolerances"} onClose={close} room={modal?.ctx}
        onSave={(payload) => actions.saveTolerances(modal.ctx.id, payload)}
      />

      <window.AccountSettingsModal
        open={modal?.kind === "account"} onClose={close} account={account}
        onSave={(payload) => actions.saveAccount(payload)}
      />
      <window.DeleteAccountModal
        open={modal?.kind === "deleteAccount"} onClose={close} account={account}
        onConfirm={() => { setPage("login"); }}
      />

      <window.PeriodModal
        open={modal?.kind === "period" && !!currentRoom} onClose={close}
        mode={modal?.ctx?.mode} period={modal?.ctx?.period}
        room={currentRoom || rooms[0]}
        schedule={schedules.find(s => s.id === modal?.ctx?.schedId)}
        allPeriods={schedules.find(s => s.id === modal?.ctx?.schedId)?.periods || []}
        onSave={(p) => {
          if (modal.ctx.mode === "add") actions.addPeriod(modal.ctx.schedId, p);
          else actions.editPeriod(modal.ctx.schedId, p);
        }}
      />
    </div>
  );
};

const root = ReactDOM.createRoot(document.getElementById("root"));
root.render(<App/>);
