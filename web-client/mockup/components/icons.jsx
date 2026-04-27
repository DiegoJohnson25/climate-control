/* global React */
// Lucide-style icons — minimal, stroke=1.5, currentColor
const svg = (d, size = 16) => (
  <svg viewBox="0 0 24 24" width={size} height={size} fill="none" stroke="currentColor"
       strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">{d}</svg>
);

window.Icon = {
  thermometer: (s) => svg(<path d="M14 4v10.54a4 4 0 1 1-4 0V4a2 2 0 0 1 4 0Z"/>, s),
  droplets: (s) => svg(<path d="M12 22a7 7 0 0 0 7-7c0-2-1-4-3-6l-4-5-4 5c-2 2-3 4-3 6a7 7 0 0 0 7 7Z"/>, s),
  flame: (s) => svg(<path d="M8.5 14.5A2.5 2.5 0 0 0 11 12c0-1.38-.5-2-1-3-1.072-2.143-.224-4.054 2-6 .5 2.5 2 4.9 4 6.5 2 1.6 3 3.5 3 5.5a7 7 0 1 1-14 0c0-1.153.433-2.294 1-3a2.5 2.5 0 0 0 2.5 2.5z"/>, s),
  power: (s) => svg(<><path d="M12 2v10"/><path d="M18.4 6.6a9 9 0 1 1-12.77.04"/></>, s),
  cpu: (s) => svg(<><rect x="4" y="4" width="16" height="16" rx="2"/><rect x="9" y="9" width="6" height="6"/><path d="M15 2v2M9 2v2M15 20v2M9 20v2M20 9h2M20 15h2M2 9h2M2 15h2"/></>, s),
  chevronLeft: (s) => svg(<path d="m15 18-6-6 6-6"/>, s),
  chevronRight: (s) => svg(<path d="m9 18 6-6-6-6"/>, s),
  chevronDown: (s) => svg(<path d="m6 9 6 6 6-6"/>, s),
  chevronUp: (s) => svg(<path d="m18 15-6-6-6 6"/>, s),
  plus: (s) => svg(<><path d="M12 5v14"/><path d="M5 12h14"/></>, s),
  x: (s) => svg(<><path d="M18 6 6 18"/><path d="M6 6l12 12"/></>, s),
  check: (s) => svg(<path d="M20 6 9 17l-5-5"/>, s),
  pencil: (s) => svg(<><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 1 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/></>, s),
  trash: (s) => svg(<><path d="M3 6h18"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/></>, s),
  kebab: (s) => svg(<><circle cx="12" cy="5" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="12" cy="19" r="1"/></>, s),
  clock: (s) => svg(<><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></>, s),
  calendar: (s) => svg(<><rect x="3" y="4" width="18" height="18" rx="2"/><path d="M16 2v4M8 2v4M3 10h18"/></>, s),
  grid: (s) => svg(<><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></>, s),
  rows: (s) => svg(<><rect x="3" y="4" width="18" height="6" rx="1"/><rect x="3" y="14" width="18" height="6" rx="1"/></>, s),
  settings: (s) => svg(<><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></>, s),
  logOut: (s) => svg(<><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><path d="m16 17 5-5-5-5"/><path d="M21 12H9"/></>, s),
  wifi: (s) => svg(<><path d="M5 12.55a11 11 0 0 1 14.08 0"/><path d="M1.42 9a16 16 0 0 1 21.16 0"/><path d="M8.53 16.11a6 6 0 0 1 6.95 0"/><circle cx="12" cy="20" r="1"/></>, s),
  info: (s) => svg(<><circle cx="12" cy="12" r="10"/><path d="M12 16v-4M12 8h.01"/></>, s),
  alert: (s) => svg(<><circle cx="12" cy="12" r="10"/><path d="M12 8v4M12 16h.01"/></>, s),
  lock: (s) => svg(<><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></>, s),
  pause: (s) => svg(<><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></>, s),
  play: (s) => svg(<polygon points="5 3 19 12 5 21 5 3"/>, s),
  dotsGrid: (s) => svg(<><circle cx="5" cy="5" r="1"/><circle cx="12" cy="5" r="1"/><circle cx="19" cy="5" r="1"/><circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="19" r="1"/><circle cx="12" cy="19" r="1"/><circle cx="19" cy="19" r="1"/></>, s),
};
