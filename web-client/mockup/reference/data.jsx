/* global */
// Seed data for the whole mock. Dates are relative to "now" so timestamps feel live.
const NOW = Date.now();
const mins = (m) => new Date(NOW - m * 60 * 1000).toISOString();
const hrs  = (h) => new Date(NOW - h * 60 * 60 * 1000).toISOString();

const ACCOUNT = {
  email: "operator@local.dev",
  timezone: "America/Los_Angeles",
};

const ROOMS = [
  {
    id: "r-living", name: "Living Room",
    hasTemp: true, hasHum: true,
    mode: "auto",                       // auto | off
    source: "schedule",                 // schedule | manual_override | grace_period | none
    activeScheduleId: "s-weekday",
    tempC: 21.4, humPct: 48.1,
    tempUpdated: mins(2), humUpdated: mins(2),
    heaterOn: false, humOn: true,
    targets: { tempC: 21.0, humPct: 50, tempDb: 0.5, humDb: 3.0 },
    desired: { tempOn: true, humOn: true, tempC: 21.0, humPct: 50, tempDb: 0.5, humDb: 3.0 },
    hold: { on: false, duration: "1h" },
  },
  {
    id: "r-bedroom", name: "Bedroom",
    hasTemp: true, hasHum: true,
    mode: "auto", source: "manual_override",
    activeScheduleId: "s-overnight",
    tempC: 19.8, humPct: 44.0,
    tempUpdated: mins(1), humUpdated: mins(1),
    heaterOn: true, humOn: false,
    targets: { tempC: 19.5, humPct: 45, tempDb: 0.5, humDb: 4.0 },
    desired: { tempOn: true, humOn: true, tempC: 19.5, humPct: 45, tempDb: 0.5, humDb: 4.0 },
    hold: { on: true, duration: "2h" },
  },
  {
    id: "r-nursery", name: "Nursery",
    hasTemp: true, hasHum: true,
    mode: "auto", source: "grace_period",
    activeScheduleId: "s-nursery",
    tempC: 22.1, humPct: 52.5,
    tempUpdated: mins(3), humUpdated: mins(4),
    heaterOn: false, humOn: true,
    targets: { tempC: 22.0, humPct: 50, tempDb: 0.5, humDb: 2.5 },
    desired: { tempOn: true, humOn: true, tempC: 22.0, humPct: 50, tempDb: 0.5, humDb: 2.5 },
    hold: { on: false, duration: "1h" },
  },
  {
    id: "r-basement", name: "Basement",
    hasTemp: true, hasHum: true,
    mode: "auto", source: "schedule",
    activeScheduleId: "s-basement",
    tempC: 17.2, humPct: 58.4,
    tempUpdated: mins(1), humUpdated: mins(1),
    heaterOn: false, humOn: false,
    targets: { tempC: 18.0, humPct: 55, tempDb: 0.5, humDb: 2.0 },
    desired: { tempOn: true, humOn: true, tempC: 18.0, humPct: 55, tempDb: 0.5, humDb: 2.0 },
    hold: { on: false, duration: "1h" },
  },
  {
    id: "r-office", name: "Office",
    hasTemp: false, hasHum: true,        // no temp sensor/actuator
    mode: "auto", source: "schedule",
    activeScheduleId: "s-office",
    tempC: null, humPct: 41.0,
    tempUpdated: null, humUpdated: mins(5),
    heaterOn: null, humOn: false,
    targets: { tempC: null, humPct: 42, tempDb: null, humDb: 1.5 },
    desired: { tempOn: false, humOn: true, tempC: null, humPct: 42, tempDb: null, humDb: 1.5 },
    hold: { on: false, duration: "1h" },
  },
  {
    id: "r-storage", name: "Storage Closet",
    hasTemp: true, hasHum: false,        // no humidifier
    mode: "off", source: "none",
    activeScheduleId: null,
    tempC: 15.7, humPct: null,
    tempUpdated: mins(6), humUpdated: null,
    heaterOn: false, humOn: null,
    targets: { tempC: 16.0, humPct: null, tempDb: 0.5, humDb: null },
    desired: { tempOn: true, humOn: false, tempC: 16.0, humPct: null, tempDb: 0.5, humDb: null },
    hold: { on: false, duration: "1h" },
  },
];

const DEVICES = [
  { id: "d-001", name: "Living Room Sensor",    hwId: "8c:1f:64:a2:3d:01", sensors: ["temp","hum"], actuators: [],            roomId: "r-living"   },
  { id: "d-002", name: "Living Room Heater",    hwId: "8c:1f:64:a2:3d:02", sensors: [],             actuators: ["heat"],      roomId: "r-living"   },
  { id: "d-003", name: "Living Room Humidifier",hwId: "8c:1f:64:a2:3d:03", sensors: [],             actuators: ["hum"],       roomId: "r-living"   },
  { id: "d-004", name: "Bedroom Sensor",        hwId: "8c:1f:64:a2:3d:10", sensors: ["temp","hum"], actuators: [],            roomId: "r-bedroom"  },
  { id: "d-005", name: "Bedroom Heater",        hwId: "8c:1f:64:a2:3d:11", sensors: [],             actuators: ["heat"],      roomId: "r-bedroom"  },
  { id: "d-006", name: "Bedroom Humidifier",    hwId: "8c:1f:64:a2:3d:12", sensors: [],             actuators: ["hum"],       roomId: "r-bedroom"  },
  { id: "d-007", name: "Nursery Sensor",        hwId: "8c:1f:64:a2:3d:20", sensors: ["temp","hum"], actuators: [],            roomId: "r-nursery"  },
  { id: "d-008", name: "Nursery Humidifier",    hwId: "8c:1f:64:a2:3d:21", sensors: [],             actuators: ["hum"],       roomId: "r-nursery"  },
  { id: "d-009", name: "Basement Combo",        hwId: "8c:1f:64:a2:3d:30", sensors: ["temp","hum"], actuators: ["heat","hum"],roomId: "r-basement" },
  { id: "d-010", name: "Office Sensor",         hwId: "8c:1f:64:a2:3d:40", sensors: ["hum"],        actuators: [],            roomId: "r-office"   },
  { id: "d-011", name: "Office Humidifier",     hwId: "8c:1f:64:a2:3d:41", sensors: [],             actuators: ["hum"],       roomId: "r-office"   },
  { id: "d-012", name: "Storage Sensor",        hwId: "8c:1f:64:a2:3d:50", sensors: ["temp"],       actuators: [],            roomId: "r-storage"  },
  { id: "d-013", name: "Storage Heater",        hwId: "8c:1f:64:a2:3d:51", sensors: [],             actuators: ["heat"],      roomId: "r-storage"  },
  { id: "d-014", name: "Spare Sensor",          hwId: "8c:1f:64:a2:3d:99", sensors: ["temp","hum"], actuators: [],            roomId: null         },
];

const SCHEDULES = [
  {
    id: "s-weekday", name: "Weekday Routine", roomId: "r-living", active: true,
    periods: [
      { id: "p1", days: [1,1,1,1,1,0,0], start: "06:00", end: "20:00", tempC: 21.0, humPct: 50 },
      { id: "p2", days: [1,1,1,1,1,0,0], start: "20:00", end: "06:00", tempC: 19.0, humPct: 45 },
      { id: "p3", days: [0,0,0,0,0,1,1], start: "07:00", end: "22:00", tempC: 21.5, humPct: 50 },
    ],
  },
  {
    id: "s-overnight", name: "Overnight Comfort", roomId: "r-bedroom", active: true,
    periods: [
      { id: "p4", days: [1,1,1,1,1,1,1], start: "07:00", end: "22:00", tempC: 20.0, humPct: 45 },
      { id: "p5", days: [1,1,1,1,1,1,1], start: "22:00", end: "07:00", tempC: 19.0, humPct: 42 },
    ],
  },
  {
    id: "s-nursery", name: "Nursery Always On", roomId: "r-nursery", active: true,
    periods: [
      { id: "p6", days: [1,1,1,1,1,1,1], start: "00:00", end: "24:00", tempC: 22.0, humPct: 50 },
    ],
  },
  {
    id: "s-basement", name: "Basement Maintenance", roomId: "r-basement", active: true,
    periods: [
      { id: "p7", days: [1,1,1,1,1,1,1], start: "00:00", end: "24:00", tempC: 18.0, humPct: 55 },
    ],
  },
  {
    id: "s-office", name: "Work Hours", roomId: "r-office", active: true,
    periods: [
      { id: "p8", days: [1,1,1,1,1,0,0], start: "08:00", end: "18:00", tempC: null, humPct: 42 },
    ],
  },
  {
    id: "s-weekday-alt", name: "Morning Routine", roomId: "r-living", active: false,
    periods: [
      { id: "p9",  days: [1,1,1,1,1,0,0], start: "06:00", end: "22:00", tempC: 22.0, humPct: 50 },
      { id: "p10", days: [1,1,1,1,1,0,0], start: "22:00", end: "06:00", tempC: 18.0, humPct: 45 },
    ],
  },
];

// ---- Mock history series ----------------------------------------------
// Generates a sinusoidal temp + hum path with target step, deadband, and duty-cycle.
function generateHistory(windowKey, room) {
  const map = { "1h": 60, "6h": 360, "24h": 1440, "7d": 60*24*7 };
  const totalMin = map[windowKey];
  const step = windowKey === "7d" ? 60 : windowKey === "24h" ? 15 : windowKey === "6h" ? 5 : 1;
  const pts = [];
  const now = Date.now();
  const base = { tempC: room.targets.tempC ?? 21, humPct: room.targets.humPct ?? 50 };
  for (let m = totalMin; m >= 0; m -= step) {
    const t = now - m * 60 * 1000;
    const phase = (totalMin - m) / totalMin * Math.PI * (windowKey === "7d" ? 14 : 4);
    const tempAvg = base.tempC + Math.sin(phase) * 0.8 + (Math.random() - 0.5) * 0.15;
    const humAvg  = base.humPct + Math.cos(phase * 0.8) * 3 + (Math.random() - 0.5) * 0.7;
    // Simulate a brief null gap midway on 24h and 7d
    const isGap = (windowKey === "24h" || windowKey === "7d") && m > totalMin * 0.45 && m < totalMin * 0.48;
    pts.push({
      t,
      tempAvg: room.hasTemp && !isGap ? tempAvg : null,
      humAvg:  room.hasHum  && !isGap ? humAvg  : null,
      tempTarget: room.hasTemp ? base.tempC : null,
      tempDb: room.hasTemp ? room.targets.tempDb : null,
      humTarget: room.hasHum ? base.humPct : null,
      humDb: room.hasHum ? room.targets.humDb : null,
      heatDuty: room.hasTemp && !isGap ? Math.max(0, Math.sin(phase + 1.2)) * 0.7 : null,
      humDuty:  room.hasHum  && !isGap ? Math.max(0, Math.cos(phase - 0.6)) * 0.5 : null,
    });
  }
  return pts;
}

window.CCData = { ACCOUNT, ROOMS, DEVICES, SCHEDULES, generateHistory };
