import { useState } from 'react'
import { Thermometer, Droplets, Flame } from 'lucide-react'
import { useClimate } from '@/hooks/useClimate'
import { useSchedules } from '@/hooks/useSchedules'

// ---------------------------------------------------------------------------
// OverviewTab
// Read-only current state + control panel shell for a room. Shown on the
// Overview tab of RoomDetailPage. The control panel card is a read-only
// placeholder shell until Phase 6c.
//
// Props:
//   roomId — string, from useParams in the parent page
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// Returns dot/label style based on climate staleness.
function liveStatus(climate) {
  if (!climate) {
    return { dotColor: 'var(--cc-fg-4)', dotShadow: 'none', labelColor: 'var(--cc-fg-4)', label: 'No data' }
  }
  const ageMs = Date.now() - new Date(climate.time).getTime()
  if (ageMs < 5 * 60 * 1000) {
    return {
      dotColor: 'var(--cc-success)',
      dotShadow: '0 0 0 3px var(--cc-success-tint)',
      labelColor: 'var(--cc-fg-3)',
      label: 'Live',
    }
  }
  return { dotColor: 'var(--cc-warning)', dotShadow: 'none', labelColor: 'var(--cc-warning-fg)', label: 'Stale' }
}

const ACTIVE_SOURCES = new Set(['manual_override', 'schedule', 'grace_period'])

// ---------------------------------------------------------------------------
// Shared sub-components
// ---------------------------------------------------------------------------

function ModeBadge({ mode }) {
  return (
    <span className={`cc-badge${mode === 'AUTO' ? ' cc-badge--ok' : ''}`}>
      {mode ?? '—'}
    </span>
  )
}

// Mono label displayed on the right of actuator rows.
const monoStyle = {
  fontFamily: 'var(--cc-font-mono)',
  fontSize: 11,
  letterSpacing: 'var(--cc-tracking-wide)',
  textTransform: 'uppercase',
  fontWeight: 'var(--cc-fw-medium)',
}

// ---------------------------------------------------------------------------
// CurrentStateCard
// ---------------------------------------------------------------------------

function CurrentStateCard({ climate }) {
  const status = liveStatus(climate)

  const heaterCmd  = climate?.heater_cmd     ?? null
  const humCmd     = climate?.humidifier_cmd ?? null
  const source     = climate?.control_source ?? null
  const showSource = climate !== null && ACTIVE_SOURCES.has(source)

  return (
    <div className="cc-card" style={{ padding: 24 }}>
      {/* Header row */}
      <div style={{ display: 'flex', alignItems: 'center', marginBottom: 20 }}>
        <span className="cc-section-label">Current state</span>
        <div style={{ flex: 1 }} />
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <div
            style={{
              width: 7,
              height: 7,
              borderRadius: '50%',
              flexShrink: 0,
              background: status.dotColor,
              boxShadow: status.dotShadow,
            }}
          />
          <span style={{ fontFamily: 'var(--cc-font-mono)', fontSize: 11, color: status.labelColor }}>
            {status.label}
          </span>
        </div>
      </div>

      {/* Readout grid */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr',
          gap: 20,
          marginBottom: 20,
        }}
      >
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 5, marginBottom: 8 }}>
            <Thermometer size={12} style={{ color: 'var(--cc-fg-3)' }} />
            <span className="cc-meta">Temperature</span>
          </div>
          <span
            className="cc-readout"
            style={{ color: climate?.avg_temp != null ? 'var(--cc-heat)' : 'var(--cc-fg-4)' }}
          >
            {climate?.avg_temp != null ? Number(climate.avg_temp).toFixed(1) : '—'}
            <span style={{ fontSize: '0.45em', marginLeft: 3, color: 'var(--cc-fg-3)' }}>°C</span>
          </span>
        </div>

        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 5, marginBottom: 8 }}>
            <Droplets size={12} style={{ color: 'var(--cc-fg-3)' }} />
            <span className="cc-meta">Humidity</span>
          </div>
          <span
            className="cc-readout"
            style={{ color: climate?.avg_hum != null ? 'var(--cc-cool)' : 'var(--cc-fg-4)' }}
          >
            {climate?.avg_hum != null ? Number(climate.avg_hum).toFixed(1) : '—'}
            <span style={{ fontSize: '0.45em', marginLeft: 3, color: 'var(--cc-fg-3)' }}>%</span>
          </span>
        </div>
      </div>

      {/* Actuator section — always rendered */}
      <div
        style={{
          borderTop: '1px solid var(--cc-divider)',
          paddingTop: 18,
          marginBottom: 18,
          display: 'flex',
          flexDirection: 'column',
          gap: 12,
        }}
      >
        {/* Heater row */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <Flame
            size={14}
            style={{ color: heaterCmd === true ? 'var(--cc-heat)' : 'var(--cc-fg-4)' }}
          />
          <span style={{ fontSize: 13, color: 'var(--cc-fg-2)', flex: 1 }}>Heater</span>
          <span
            style={{
              ...monoStyle,
              color: heaterCmd === true ? 'var(--cc-heat)' : 'var(--cc-fg-4)',
            }}
          >
            {heaterCmd === null ? '—' : heaterCmd ? 'ON' : 'OFF'}
          </span>
        </div>

        {/* Humidifier row */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <Droplets
            size={14}
            style={{ color: humCmd === true ? 'var(--cc-cool)' : 'var(--cc-fg-4)' }}
          />
          <span style={{ fontSize: 13, color: 'var(--cc-fg-2)', flex: 1 }}>Humidifier</span>
          <span
            style={{
              ...monoStyle,
              color: humCmd === true ? 'var(--cc-cool)' : 'var(--cc-fg-4)',
            }}
          >
            {humCmd === null ? '—' : humCmd ? 'ON' : 'OFF'}
          </span>
        </div>
      </div>

      {/* Source and mode row — always rendered */}
      <div
        style={{
          borderTop: '1px solid var(--cc-divider)',
          paddingTop: 14,
          marginBottom: 18,
          display: 'flex',
          alignItems: 'center',
          gap: 8,
        }}
      >
        <span style={{ fontSize: 13, color: 'var(--cc-fg-2)', flex: 1 }}>Active source</span>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          {source === 'manual_override' && (
            <span
              className="cc-badge"
              style={{
                background: 'var(--cc-hold-tint)',
                color: 'var(--cc-hold-fg)',
                borderColor: 'rgba(161, 98, 7, 0.30)',
              }}
            >
              Manual
            </span>
          )}
          {source === 'schedule' && (
            <span
              className="cc-badge"
              style={{
                background: 'rgba(20, 19, 15, 0.06)',
                color: 'var(--cc-fg)',
                borderColor: 'var(--cc-border-strong)',
              }}
            >
              Schedule
            </span>
          )}
          {source === 'grace_period' && (
            <span
              className="cc-badge"
              style={{
                background: 'var(--cc-grace-tint)',
                color: 'var(--cc-grace-fg)',
                borderColor: 'rgba(100, 116, 139, 0.30)',
              }}
            >
              Grace period
            </span>
          )}
          {!showSource && (
            <span className="cc-badge">None</span>
          )}
          {showSource && climate?.mode && (
            <ModeBadge mode={climate.mode} />
          )}
        </div>
      </div>

      {/* Targets section — always rendered */}
      <div
        style={{
          borderTop: '1px solid var(--cc-divider)',
          paddingTop: 16,
          display: 'grid',
          gridTemplateColumns: '1fr 1fr',
          gap: 16,
        }}
      >
        <div>
          <div className="cc-meta" style={{ marginBottom: 4 }}>Target temperature</div>
          <div
            style={{
              fontFamily: 'var(--cc-font-mono)',
              fontSize: 14,
              fontWeight: 'var(--cc-fw-medium)',
              fontVariantNumeric: 'tabular-nums',
              color: climate?.target_temp != null ? 'var(--cc-fg)' : 'var(--cc-fg-4)',
            }}
          >
            {climate?.target_temp != null ? (
              <>
                {climate.target_temp.toFixed(1)}°C
                {climate.deadband_temp != null && (
                  <span style={{ fontSize: 11, color: 'var(--cc-fg-3)', marginLeft: 6 }}>
                    ±{climate.deadband_temp.toFixed(1)}
                  </span>
                )}
              </>
            ) : '—'}
          </div>
        </div>

        <div>
          <div className="cc-meta" style={{ marginBottom: 4 }}>Target humidity</div>
          <div
            style={{
              fontFamily: 'var(--cc-font-mono)',
              fontSize: 14,
              fontWeight: 'var(--cc-fw-medium)',
              fontVariantNumeric: 'tabular-nums',
              color: climate?.target_hum != null ? 'var(--cc-cool)' : 'var(--cc-fg-4)',
            }}
          >
            {climate?.target_hum != null ? (
              <>
                {climate.target_hum.toFixed(1)}%
                {climate.deadband_hum != null && (
                  <span style={{ fontSize: 11, color: 'var(--cc-fg-3)', marginLeft: 6 }}>
                    ±{climate.deadband_hum.toFixed(1)}
                  </span>
                )}
              </>
            ) : '—'}
          </div>
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// ControlPanelShell — interactive shell with local draft state.
// Replaced with real desired-state data in Phase 6c after schema migration.
// ---------------------------------------------------------------------------

function ControlPanelShell({ climate, roomId }) {
  const { schedules } = useSchedules(roomId)
  const activeSchedule = schedules.find((s) => s.is_active) ?? null

  const [controlType, setControlType] = useState('schedule')
  const [mode, setMode] = useState('OFF')
  const [tempTarget] = useState(22.0)
  const [humTarget] = useState(50.0)

  const isManual = controlType === 'manual'
  const isAuto   = mode === 'AUTO'

  const manualRowDisabled = !isManual || !isAuto

  return (
    <div className="cc-card" style={{ padding: 24 }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', marginBottom: 20 }}>
        <span className="cc-section-label">Control panel</span>
      </div>

      {/* Control type row */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 20,
        }}
      >
        <span style={{ fontSize: 13, fontWeight: 'var(--cc-fw-medium)', color: 'var(--cc-fg)' }}>
          Control type
        </span>
        <div className="cc-seg">
          <button
            className={controlType === 'schedule' ? 'is-on' : ''}
            onClick={() => setControlType('schedule')}
          >
            Schedule
          </button>
          <button
            className={controlType === 'manual' ? 'is-on' : ''}
            onClick={() => setControlType('manual')}
          >
            Manual
          </button>
        </div>
      </div>

      {/* Schedule section */}
      <div style={{ marginBottom: 20 }}>
        <div className="cc-section-label" style={{ marginBottom: 8 }}>Schedule</div>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            padding: '10px 12px',
            border: '1px solid var(--cc-border)',
            borderRadius: 6,
            background: 'var(--cc-surface-2)',
            opacity: isManual ? 0.5 : 1,
          }}
        >
          <span
            className="cc-statusdot"
            style={{ background: activeSchedule && !isManual ? 'var(--cc-info)' : 'var(--cc-fg-4)' }}
          />
          <span
            style={{
              fontSize: 13,
              flex: 1,
              color: activeSchedule
                ? (isManual ? 'var(--cc-fg-3)' : 'var(--cc-fg)')
                : 'var(--cc-fg-4)',
            }}
          >
            {activeSchedule ? activeSchedule.name : 'No active schedule'}
          </span>
          {isManual && (
            <span className="cc-meta">Overridden by manual</span>
          )}
        </div>
      </div>

      {/* Divider */}
      <div style={{ borderTop: '1px solid var(--cc-divider)', marginBottom: 20 }} />

      {/* Manual settings section */}
      <div>
        {/* Section header */}
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
          <span className="cc-section-label">Manual settings</span>
          {!isManual && (
            <span className="cc-meta" style={{ marginLeft: 8, color: 'var(--cc-fg-4)' }}>
              Schedule active
            </span>
          )}
        </div>

        <div style={{ opacity: isManual ? 1 : 0.5 }}>
          {/* Mode row */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              marginBottom: 16,
            }}
          >
            <span style={{ fontSize: 13, color: 'var(--cc-fg-2)' }}>Mode</span>
            <div className={`cc-seg${!isManual ? ' cc-seg--disabled' : ''}`}>
              <button
                className={mode === 'OFF' ? 'is-on' : ''}
                onClick={() => setMode('OFF')}
                disabled={!isManual}
              >
                OFF
              </button>
              <button
                className={mode === 'AUTO' ? 'is-on' : ''}
                onClick={() => setMode('AUTO')}
                disabled={!isManual}
              >
                AUTO
              </button>
            </div>
          </div>

          {/* Capability rows */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12, marginBottom: 20 }}>
            {/* Temperature row */}
            <div className={`cc-row${manualRowDisabled ? ' cc-row--disabled' : ''}`}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, flex: 1, minWidth: 0 }}>
                <button
                  className={`cc-togdot${isManual && isAuto ? ' cc-togdot--on' : ' cc-togdot--disabled'}`}
                />
                <Thermometer size={14} style={{ color: 'var(--cc-fg-3)' }} />
                <span style={{ fontSize: 13, color: 'var(--cc-fg-2)' }}>Temperature</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <input
                  className="cc-input cc-input--mono"
                  disabled={manualRowDisabled}
                  value={isAuto ? tempTarget.toFixed(1) : '—'}
                  style={{ width: 80, textAlign: 'right' }}
                  readOnly
                />
                <span style={{ fontSize: 12, color: 'var(--cc-fg-3)' }}>°C</span>
                <span className="cc-dbpill" style={{ cursor: 'default' }}>
                  {climate?.deadband_temp != null ? `±${climate.deadband_temp.toFixed(1)}` : '±—'}
                </span>
              </div>
            </div>

            {/* Humidity row */}
            <div className={`cc-row${manualRowDisabled ? ' cc-row--disabled' : ''}`}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, flex: 1, minWidth: 0 }}>
                <button
                  className={`cc-togdot${isManual && isAuto ? ' cc-togdot--on' : ' cc-togdot--disabled'}`}
                />
                <Droplets size={14} style={{ color: 'var(--cc-fg-3)' }} />
                <span style={{ fontSize: 13, color: 'var(--cc-fg-2)' }}>Humidity</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <input
                  className="cc-input cc-input--mono"
                  disabled={manualRowDisabled}
                  value={isAuto ? humTarget.toFixed(1) : '—'}
                  style={{ width: 80, textAlign: 'right' }}
                  readOnly
                />
                <span style={{ fontSize: 12, color: 'var(--cc-fg-3)' }}>%</span>
                <span className="cc-dbpill" style={{ cursor: 'default' }}>
                  {climate?.deadband_hum != null ? `±${climate.deadband_hum.toFixed(1)}` : '±—'}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Divider */}
      <div style={{ borderTop: '1px solid var(--cc-divider)', marginTop: 4, marginBottom: 16 }} />

      {/* Footer */}
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'flex-end',
          gap: 8,
        }}
      >
        <div style={{ display: 'flex', gap: 8 }}>
          <button className="cc-btn cc-btn--ghost" disabled>Revert</button>
          <button className="cc-btn cc-btn--primary" disabled>Apply</button>
        </div>
        <span className="cc-meta" style={{ fontSize: 11, color: 'var(--cc-fg-4)' }}>
          Control panel coming soon
        </span>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// OverviewTab
// ---------------------------------------------------------------------------

export default function OverviewTab({ roomId }) {
  const { climate } = useClimate(roomId)

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))',
        gap: 20,
        alignItems: 'start',
      }}
    >
      <CurrentStateCard climate={climate} />
      <ControlPanelShell climate={climate} roomId={roomId} />
    </div>
  )
}
