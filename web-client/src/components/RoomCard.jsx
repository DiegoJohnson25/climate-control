import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Flame, Droplets } from 'lucide-react'
import { useClimate } from '@/hooks/useClimate'

// ---------------------------------------------------------------------------
// RoomCard
// Dashboard grid card for a single room. Fetches its own climate data via
// useClimate. Clicking navigates to /rooms/:id.
//
// Props:
//   room — room object from GET /api/v1/rooms
// ---------------------------------------------------------------------------

const SOURCE_CONFIG = {
  manual_override: {
    label: 'Hold active',
    style: {
      background: 'var(--cc-hold-tint)',
      color: 'var(--cc-hold-fg)',
      borderColor: 'rgba(161, 98, 7, 0.30)',
    },
  },
  schedule: {
    label: 'Schedule',
    style: {
      background: 'rgba(20, 19, 15, 0.06)',
      color: 'var(--cc-fg)',
      borderColor: 'var(--cc-border-strong)',
    },
  },
  grace_period: {
    label: 'Grace period',
    style: {
      background: 'var(--cc-grace-tint)',
      color: 'var(--cc-grace-fg)',
      borderColor: 'rgba(100, 116, 139, 0.30)',
    },
  },
}

function ModeBadge({ mode }) {
  return (
    <span className={`cc-badge${mode === 'AUTO' ? ' cc-badge--ok' : ''}`}>
      {mode ?? '—'}
    </span>
  )
}

function ActuatorBadge({ on, icon: Icon, labelOn, labelOff, onClassName }) {
  return (
    <span className={`cc-badge${on ? ` ${onClassName}` : ''}`}>
      <Icon size={11} />
      {on ? labelOn : labelOff}
    </span>
  )
}

function ControlSourceBadge({ source }) {
  const cfg = SOURCE_CONFIG[source]
  if (!cfg) return null
  return (
    <span className="cc-badge" style={cfg.style}>
      {cfg.label}
    </span>
  )
}

export default function RoomCard({ room }) {
  const navigate = useNavigate()
  const { climate } = useClimate(room.id)
  const [hovered, setHovered] = useState(false)

  const temp = climate?.avg_temp != null ? Number(climate.avg_temp).toFixed(1) : null
  const hum  = climate?.avg_hum  != null ? Number(climate.avg_hum).toFixed(1)  : null

  return (
    <div
      onClick={() => navigate(`/rooms/${room.id}`)}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        background: 'var(--cc-surface)',
        border: `1px solid ${hovered ? 'var(--cc-border-strong)' : 'var(--cc-border)'}`,
        borderRadius: 8,
        boxShadow: hovered ? 'var(--cc-shadow-md)' : 'var(--cc-shadow-sm)',
        transform: hovered ? 'translateY(-1px)' : 'translateY(0)',
        transition: 'border-color 150ms var(--cc-ease), box-shadow 150ms var(--cc-ease), transform 150ms var(--cc-ease)',
        cursor: 'pointer',
      }}
    >
      {/* Top section */}
      <div style={{ flex: 1, padding: '16px 18px 14px' }}>
        {/* Row 1: name + mode badge */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 10,
            marginBottom: 14,
          }}
        >
          <span
            style={{
              fontSize: 15,
              fontWeight: 'var(--cc-fw-semibold)',
              letterSpacing: 'var(--cc-tracking-tight)',
              color: 'var(--cc-fg)',
            }}
          >
            {room.name}
          </span>
          <ModeBadge mode={climate?.mode ?? null} />
        </div>

        {/* Row 2: temperature + humidity readouts */}
        <div style={{ display: 'flex', gap: 22, alignItems: 'baseline' }}>
          <div>
            <span
              className="cc-readout-sm"
              style={{ color: temp !== null ? 'var(--cc-heat)' : 'var(--cc-fg-4)' }}
            >
              {temp ?? '—'}
            </span>
            <span style={{ fontSize: 12, marginLeft: 2, color: 'var(--cc-fg-3)' }}>
              °C
            </span>
          </div>

          <div>
            <span
              className="cc-readout-sm"
              style={{ color: hum !== null ? 'var(--cc-cool)' : 'var(--cc-fg-4)' }}
            >
              {hum ?? '—'}
            </span>
            <span style={{ fontSize: 12, marginLeft: 2, color: 'var(--cc-fg-3)' }}>
              %
            </span>
          </div>
        </div>
      </div>

      {/* Bottom section — actuator + source badges */}
      <div
        style={{
          marginTop: 'auto',
          borderTop: '1px solid var(--cc-divider)',
          padding: '12px 18px',
          display: 'flex',
          flexWrap: 'wrap',
          gap: 6,
        }}
      >
        {climate !== null && climate.heater_cmd !== null && climate.heater_cmd !== undefined && (
          <ActuatorBadge
            on={climate.heater_cmd}
            icon={Flame}
            labelOn="Heater on"
            labelOff="Heater off"
            onClassName="cc-badge--heat"
          />
        )}
        {climate !== null && climate.humidifier_cmd !== null && climate.humidifier_cmd !== undefined && (
          <ActuatorBadge
            on={climate.humidifier_cmd}
            icon={Droplets}
            labelOn="Humidifier on"
            labelOff="Humidifier off"
            onClassName="cc-badge--cool"
          />
        )}
        {climate !== null && (
          <ControlSourceBadge source={climate.control_source} />
        )}
      </div>
    </div>
  )
}
