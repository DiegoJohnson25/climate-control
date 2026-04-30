import { useState } from 'react'
import { MapPin, X } from 'lucide-react'

// ---------------------------------------------------------------------------
// TimezonePrompt
// Shown on the dashboard when the user's timezone is UTC (the default set
// at registration). Prompts the user to set their timezone. Dismissed state
// is persisted to localStorage so it does not reappear after being closed.
//
// Props:
//   userTimezone  — string, the user's current timezone from GET /users/me
//   onSave        — async function(timezone: string) — called when user saves
// ---------------------------------------------------------------------------

const STORAGE_KEY = 'cc-timezone-prompt-dismissed'

// Common timezones for the selector. Full IANA list is available via
// Intl.supportedValuesOf('timeZone') but is too long for a simple dropdown.
const TIMEZONES = [
  'UTC',
  'America/St_Johns',
  'America/Halifax',
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Phoenix',
  'America/Los_Angeles',
  'America/Anchorage',
  'America/Honolulu',
  'Europe/London',
  'Europe/Paris',
  'Europe/Berlin',
  'Europe/Helsinki',
  'Europe/Moscow',
  'Asia/Dubai',
  'Asia/Karachi',
  'Asia/Kolkata',
  'Asia/Dhaka',
  'Asia/Bangkok',
  'Asia/Singapore',
  'Asia/Tokyo',
  'Asia/Seoul',
  'Australia/Perth',
  'Australia/Adelaide',
  'Australia/Sydney',
  'Pacific/Auckland',
]

export default function TimezonePrompt({ userTimezone, onSave }) {
  const [dismissed, setDismissed] = useState(
    () => localStorage.getItem(STORAGE_KEY) === 'true'
  )
  const [selected, setSelected] = useState(
    () => Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  )
  const [saving, setSaving] = useState(false)

  if (dismissed) return null
  if (userTimezone === null || userTimezone === undefined) return null
  if (userTimezone !== 'UTC') return null

  function handleDismiss() {
    localStorage.setItem(STORAGE_KEY, 'true')
    setDismissed(true)
  }

  async function handleSave() {
    setSaving(true)
    try {
      await onSave(selected)
      localStorage.setItem(STORAGE_KEY, 'true')
      setDismissed(true)
    } catch {
      // error handling delegated to parent via onSave
    } finally {
      setSaving(false)
    }
  }

  return (
    <div
      style={{
        background: 'var(--cc-surface)',
        border: '1px solid var(--cc-border)',
        borderLeft: '3px solid var(--cc-info)',
        borderRadius: 'var(--cc-radius-md)',
        padding: '14px 16px',
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        marginBottom: 24,
      }}
    >
      <MapPin
        size={16}
        style={{ color: 'var(--cc-info)', flexShrink: 0 }}
      />

      <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
        <span style={{ fontSize: 'var(--cc-fs-sm)', color: 'var(--cc-fg-2)' }}>
          Set your timezone for accurate schedule display
        </span>

        <select
          className="cc-input"
          value={selected}
          onChange={(e) => setSelected(e.target.value)}
          style={{ width: 'auto', minWidth: 200 }}
        >
          {TIMEZONES.map((tz) => (
            <option key={tz} value={tz}>
              {tz.replace(/_/g, ' ')}
            </option>
          ))}
        </select>

        <button
          className="cc-btn cc-btn--primary cc-btn--sm"
          onClick={handleSave}
          disabled={saving || selected === userTimezone}
        >
          {saving ? 'Saving…' : 'Save Timezone'}
        </button>
      </div>

      <button
        className="cc-iconbtn"
        onClick={handleDismiss}
        title="Dismiss"
        style={{ flexShrink: 0 }}
      >
        <X size={14} />
      </button>
    </div>
  )
}
