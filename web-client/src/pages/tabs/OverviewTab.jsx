import { useState, useEffect, useRef } from 'react'
import { Thermometer, Droplets, Flame } from 'lucide-react'
import { useClimate } from '@/hooks/useClimate'
import { useSchedules } from '@/hooks/useSchedules'
import { useDesiredState } from '@/hooks/useDesiredState'
import { updateDesiredState } from '@/api/rooms'
import TolerancesModal from '@/components/TolerancesModal'
import { getToken } from '@/api/auth'

// ---------------------------------------------------------------------------
// OverviewTab
// Current state card + fully wired control panel for a room. Shown on the
// Overview tab of RoomDetailPage.
//
// Props:
//   roomId       — string, from useParams in the parent page
//   capabilities — { temperature: bool, humidity: bool } from room object
//   room         — full room object (passed through to ControlPanel)
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
              color: climate?.target_temp != null ? 'var(--cc-heat)' : 'var(--cc-fg-4)',
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
// ControlPanel — fully wired control panel backed by desired state.
// ---------------------------------------------------------------------------

function ControlPanel({ roomId, capabilities, room, mutateRoom }) {
  const { schedules } = useSchedules(roomId)
  const activeSchedule = schedules.find((s) => s.is_active) ?? null

  const { desiredState, mutate: mutateDesiredState } = useDesiredState(roomId)

  const [draft, setDraft] = useState({
    controlType: 'schedule',
    mode: 'OFF',
    tempTarget: null,
    humTarget: null,
    tempEnabled: false,
    humEnabled: false,
  })
  const [applying, setApplying] = useState(false)
  const [applyError, setApplyError] = useState(null)
  const [tempTargetError, setTempTargetError] = useState(null)
  const [humTargetError,  setHumTargetError]  = useState(null)
  const [tolerancesOpen, setTolerancesOpen] = useState(false)
  const resetCount = useRef(0)

  useEffect(() => {
    if (!desiredState) return
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setDraft({
      controlType: desiredState.manual_active ? 'manual' : 'schedule',
      mode: desiredState.mode,
      tempTarget: desiredState.target_temp ?? null,
      humTarget:  desiredState.target_hum  ?? null,
      tempEnabled: desiredState.target_temp != null,
      humEnabled: desiredState.target_hum != null,
    })
    setTempTargetError(null)
    setHumTargetError(null)
    resetCount.current += 1
  }, [desiredState])

  const { controlType, mode, tempTarget, humTarget, tempEnabled, humEnabled } = draft

  const isDirty = !!desiredState && (
    controlType !== (desiredState.manual_active ? 'manual' : 'schedule') ||
    mode !== desiredState.mode ||
    tempEnabled !== (desiredState.target_temp != null) ||
    humEnabled !== (desiredState.target_hum != null) ||
    (tempEnabled && tempTarget !== desiredState.target_temp) ||
    (humEnabled  && humTarget  !== desiredState.target_hum)
  )

  const isManual = controlType === 'manual'
  const isAuto   = mode === 'AUTO'

  const tempHardDisabled = !capabilities.temperature || !isManual || !isAuto
  const humHardDisabled  = !capabilities.humidity    || !isManual || !isAuto

  const tempContentDim = tempHardDisabled || !tempEnabled
  const humContentDim  = humHardDisabled  || !humEnabled

  const tempTogdotClass = tempHardDisabled ? 'cc-togdot--disabled'
                        : tempEnabled      ? 'cc-togdot--on'
                        : ''
  const humTogdotClass  = humHardDisabled  ? 'cc-togdot--disabled'
                        : humEnabled       ? 'cc-togdot--on'
                        : ''

  const tempTooltip = !capabilities.temperature ? 'Assign a temperature sensor and heater to enable'
                    : !isManual                  ? 'Set control type to Manual to edit'
                    : !isAuto                    ? 'Set mode to AUTO to edit'
                    : null
  const humTooltip  = !capabilities.humidity    ? 'Assign a humidity sensor and humidifier to enable'
                    : !isManual                  ? 'Set control type to Manual to edit'
                    : !isAuto                    ? 'Set mode to AUTO to edit'
                    : null

  const modeTooltip = !isManual ? 'Set control type to Manual to edit' : null

  async function handleApply() {
    if (tempTargetError || humTargetError) {
      setApplyError('Fix validation errors before applying.')
      return
    }
    if (isManual && isAuto && tempEnabled && tempTarget == null) {
      setApplyError('Enter a temperature target.')
      return
    }
    if (isManual && isAuto && humEnabled && humTarget == null) {
      setApplyError('Enter a humidity target.')
      return
    }
    setApplying(true)
    setApplyError(null)
    const payload = {
      mode,
      manual_active: isManual,
      target_temp: tempEnabled ? tempTarget : null,
      target_hum:  humEnabled  ? humTarget  : null,
      manual_override_until: isManual ? 'indefinite' : null,
    }
    try {
      await updateDesiredState(roomId, payload)
      mutateDesiredState()
    } catch (err) {
      setApplyError(err.status === 422 ? err.message : 'Something went wrong.')
    } finally {
      setApplying(false)
    }
  }

  function handleRevert() {
    if (!desiredState) return
    setDraft({
      controlType: desiredState.manual_active ? 'manual' : 'schedule',
      mode: desiredState.mode,
      tempTarget: desiredState.target_temp ?? null,
      humTarget:  desiredState.target_hum  ?? null,
      tempEnabled: desiredState.target_temp != null,
      humEnabled: desiredState.target_hum != null,
    })
    setTempTargetError(null)
    setHumTargetError(null)
    resetCount.current += 1
  }

  return (
    <div className="cc-card" style={{ padding: 20 }}>
      {/* Control type row */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 16,
        }}
      >
        <span style={{ fontSize: 13, fontWeight: 'var(--cc-fw-medium)', color: 'var(--cc-fg)' }}>
          Control type
        </span>
        <div className="cc-seg">
          <button
            className={controlType === 'schedule' ? 'is-on' : ''}
            onClick={() => setDraft(d => ({ ...d, controlType: 'schedule' }))}
          >
            Schedule
          </button>
          <button
            className={controlType === 'manual' ? 'is-on' : ''}
            onClick={() => setDraft(d => ({ ...d, controlType: 'manual' }))}
          >
            Manual
          </button>
        </div>
      </div>

      {/* Schedule section */}
      <div style={{ marginBottom: 16 }}>
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
      <div style={{ borderTop: '1px solid var(--cc-divider)', marginBottom: 16 }} />

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

        {/* Mode row — outside opacity wrapper so its tooltip isn't clipped by the stacking context */}
        <div
          className="cc-tooltip cc-tooltip--right"
          data-tooltip={modeTooltip || undefined}
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 16,
          }}
        >
          <span style={{ fontSize: 13, color: 'var(--cc-fg-2)', opacity: isManual ? 1 : 0.5 }}>Mode</span>
          <div className={`cc-seg${!isManual ? ' cc-seg--disabled' : ''}`} style={{ opacity: isManual ? 1 : 0.5 }}>
            <button
              className={mode === 'OFF' ? 'is-on' : ''}
              onClick={() => setDraft(d => ({ ...d, mode: 'OFF' }))}
              disabled={!isManual}
            >
              OFF
            </button>
            <button
              className={mode === 'AUTO' ? 'is-on' : ''}
              onClick={() => setDraft(d => ({ ...d, mode: 'AUTO' }))}
              disabled={!isManual}
            >
              AUTO
            </button>
          </div>
        </div>

        {/* Capability rows — no opacity wrapper; tempContentDim/humContentDim already factor in !isManual */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4, marginBottom: 16 }}>
            {/* Temperature row */}
            <div
              className="cc-row cc-tooltip"
              data-tooltip={tempTooltip || undefined}
            >
              <button
                className={`cc-togdot ${tempTogdotClass}`}
                onClick={() => !tempHardDisabled && setDraft(d => ({ ...d, tempEnabled: !d.tempEnabled }))}
                disabled={tempHardDisabled}
              />
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, flex: 1, minWidth: 0,
                            opacity: tempContentDim ? 0.5 : 1 }}>
                <Thermometer size={14} style={{ color: 'var(--cc-fg-3)' }} />
                <span style={{ fontSize: 13, color: 'var(--cc-fg-2)' }}>Temperature</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8,
                            opacity: tempContentDim ? 0.5 : 1 }}>
                <input
                  key={`temp-target-${resetCount.current}`}
                  className="cc-input cc-input--mono"
                  disabled={tempContentDim}
                  defaultValue={tempTarget != null ? tempTarget.toFixed(1) : ''}
                  onBlur={(e) => {
                    const raw = e.target.value
                    const val = parseFloat(raw)
                    if (!raw.trim() || isNaN(val)) {
                      setTempTargetError('Enter a valid number')
                      return
                    }
                    if (val < 5.0 || val > 40.0) {
                      setTempTargetError('Must be between 5.0°C and 40.0°C')
                      return
                    }
                    setDraft(d => ({ ...d, tempTarget: val }))
                    setTempTargetError(null)
                  }}
                  style={{ width: 80, textAlign: 'right',
                           borderColor: tempTargetError ? 'var(--cc-danger)' : undefined }}
                />
                <span style={{ fontSize: 12, color: 'var(--cc-fg-3)' }}>°C</span>
                <span
                  className="cc-dbpill"
                  style={{ cursor: 'pointer' }}
                  onClick={() => setTolerancesOpen(true)}
                >
                  {room?.deadband_temp != null ? `±${room.deadband_temp.toFixed(1)}` : '±—'}
                </span>
              </div>
            </div>
            {tempTargetError && (
              <span style={{ fontSize: 11, color: 'var(--cc-danger)', marginTop: 2, paddingLeft: 28 }}>
                {tempTargetError}
              </span>
            )}

            {/* Humidity row */}
            <div
              className="cc-row cc-tooltip"
              data-tooltip={humTooltip || undefined}
            >
              <button
                className={`cc-togdot ${humTogdotClass}`}
                onClick={() => !humHardDisabled && setDraft(d => ({ ...d, humEnabled: !d.humEnabled }))}
                disabled={humHardDisabled}
              />
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, flex: 1, minWidth: 0,
                            opacity: humContentDim ? 0.5 : 1 }}>
                <Droplets size={14} style={{ color: 'var(--cc-fg-3)' }} />
                <span style={{ fontSize: 13, color: 'var(--cc-fg-2)' }}>Humidity</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8,
                            opacity: humContentDim ? 0.5 : 1 }}>
                <input
                  key={`hum-target-${resetCount.current}`}
                  className="cc-input cc-input--mono"
                  disabled={humContentDim}
                  defaultValue={humTarget != null ? humTarget.toFixed(1) : ''}
                  onBlur={(e) => {
                    const raw = e.target.value
                    const val = parseFloat(raw)
                    if (!raw.trim() || isNaN(val)) {
                      setHumTargetError('Enter a valid number')
                      return
                    }
                    if (val < 10.0 || val > 90.0) {
                      setHumTargetError('Must be between 10.0% and 90.0%')
                      return
                    }
                    setDraft(d => ({ ...d, humTarget: val }))
                    setHumTargetError(null)
                  }}
                  style={{ width: 80, textAlign: 'right',
                           borderColor: humTargetError ? 'var(--cc-danger)' : undefined }}
                />
                <span style={{ fontSize: 12, color: 'var(--cc-fg-3)' }}>%</span>
                <span
                  className="cc-dbpill"
                  style={{ cursor: 'pointer' }}
                  onClick={() => setTolerancesOpen(true)}
                >
                  {room?.deadband_hum != null ? `±${room.deadband_hum.toFixed(1)}` : '±—'}
                </span>
              </div>
            </div>
            {humTargetError && (
              <span style={{ fontSize: 11, color: 'var(--cc-danger)', marginTop: 2, paddingLeft: 28 }}>
                {humTargetError}
              </span>
            )}
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
        {applyError && (
          <span className="cc-meta" style={{ color: 'var(--cc-danger)', fontSize: 11 }}>
            {applyError}
          </span>
        )}
        <div style={{ display: 'flex', gap: 8 }}>
          <button className="cc-btn cc-btn--ghost" onClick={handleRevert}
                  disabled={!isDirty || applying}>
            Revert
          </button>
          <button className={`cc-btn ${isDirty ? 'cc-btn--primary' : 'cc-btn--ghost'}`}
                  onClick={handleApply} disabled={!isDirty || applying}>
            {applying ? 'Applying…' : 'Apply'}
          </button>
        </div>
      </div>

      <TolerancesModal
        open={tolerancesOpen}
        onClose={() => setTolerancesOpen(false)}
        room={room}
        capabilities={capabilities}
        tempTarget={isAuto && tempEnabled && tempTarget != null ? tempTarget : null}
        humTarget={isAuto && humEnabled && humTarget != null ? humTarget : null}
        showHints={true}
        onSave={async ({ deadband_temp, deadband_hum }) => {
          const res = await fetch(`/api/v1/rooms/${roomId}`, {
            method: 'PUT',
            credentials: 'include',
            headers: {
              'Content-Type': 'application/json',
              Authorization: `Bearer ${getToken()}`,
            },
            body: JSON.stringify({
              name: room.name,
              deadband_temp,
              deadband_hum,
            }),
          })
          if (!res.ok) throw new Error(res.status)
          mutateRoom()
          setTolerancesOpen(false)
        }}
      />
    </div>
  )
}

// ---------------------------------------------------------------------------
// OverviewTab
// ---------------------------------------------------------------------------

export default function OverviewTab({ roomId, capabilities, room, mutateRoom }) {
  const { climate } = useClimate(roomId)

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))',
        gap: 20,
        alignItems: 'stretch',
      }}
    >
      <CurrentStateCard climate={climate} />
      <ControlPanel roomId={roomId} capabilities={capabilities} room={room} mutateRoom={mutateRoom} />
    </div>
  )
}
