// Formats a duration relative to now.
// Example: timeAgo("2024-01-01T12:00:00Z") → "2m ago"
export function timeAgo(iso) {
  if (!iso) return '—'
  const ms = Date.now() - new Date(iso).getTime()
  const s = Math.floor(ms / 1000)
  if (s < 60) return `${s}s ago`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h / 24)}d ago`
}

// Formats an "HH:MM" 24h string to 12h display.
// Examples: fmtTime12("14:30") → "2:30pm"
//           fmtTime12("09:00") → "9am"
//           fmtTime12("24:00") → "12am"
export function fmtTime12(hm) {
  if (!hm) return ''
  if (hm === '24:00') return '12am'
  const [hStr, mStr] = hm.split(':')
  const h24 = parseInt(hStr, 10)
  const m = parseInt(mStr, 10)
  const ampm = h24 >= 12 ? 'pm' : 'am'
  let h12 = h24 % 12
  if (h12 === 0) h12 = 12
  return m === 0 ? `${h12}${ampm}` : `${h12}:${String(m).padStart(2, '0')}${ampm}`
}

// Formats minutes-of-day to 12h display.
// Examples: fmtMin12(540) → "9am"
//           fmtMin12(870) → "2:30pm"
//           fmtMin12(1440) → "12am"
export function fmtMin12(minOfDay) {
  if (minOfDay == null) return ''
  if (minOfDay >= 24 * 60) return '12am'
  const h24 = Math.floor(minOfDay / 60)
  const m = minOfDay % 60
  const ampm = h24 >= 12 ? 'pm' : 'am'
  let h12 = h24 % 12
  if (h12 === 0) h12 = 12
  return m === 0 ? `${h12}${ampm}` : `${h12}:${String(m).padStart(2, '0')}${ampm}`
}

// Formats a timestamp for history chart x-axis tick labels.
// Uses the user's timezone via Intl.DateTimeFormat.
// Examples: fmtTick12(ts, "24h", "America/New_York") → "6am"
//           fmtTick12(ts, "7d",  "America/New_York") → "Mon 6am"
export function fmtTick12(ts, windowKey, tz) {
  const opts = {
    hour: 'numeric',
    minute: 'numeric',
    hour12: true,
    timeZone: tz || 'UTC',
  }
  const parts = new Intl.DateTimeFormat('en-US', opts).formatToParts(new Date(ts))
  const hour = parts.find(p => p.type === 'hour')?.value ?? '12'
  const minute = parts.find(p => p.type === 'minute')?.value ?? '00'
  const dayPeriod = parts.find(p => p.type === 'dayPeriod')?.value?.toLowerCase() ?? 'am'
  const time = minute === '00' ? `${hour}${dayPeriod}` : `${hour}:${minute}${dayPeriod}`
  if (windowKey === '7d') {
    const weekday = new Intl.DateTimeFormat('en-US', {
      weekday: 'short',
      timeZone: tz || 'UTC',
    }).format(new Date(ts))
    return `${weekday} ${time}`
  }
  return time
}
