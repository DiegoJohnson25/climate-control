import { useState, useEffect } from 'react'

function validateTempDb(val) {
  if (val === null || val === undefined || isNaN(val)) return 'Must be between 0.1°C and 10.0°C'
  if (val < 0.1 || val > 10.0) return 'Must be between 0.1°C and 10.0°C'
  return null
}

function validateHumDb(val) {
  if (val === null || val === undefined || isNaN(val)) return 'Must be between 0.5% and 20.0%'
  if (val < 0.5 || val > 20.0) return 'Must be between 0.5% and 20.0%'
  return null
}

export default function TolerancesModal({
  open,
  onClose,
  room,
  capabilities,
  tempTarget,
  humTarget,
  showHints,
  onSave,
}) {
  const [state, setState] = useState({
    tempDb: 0.5,
    humDb: 2.0,
    tempDbError: null,
    humDbError: null,
    saving: false,
    saveError: null,
  })
  const [resetCount, setResetCount] = useState(0)

  useEffect(() => {
    if (!open || !room) return
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setState({
      tempDb: room.deadband_temp ?? 0.5,
      humDb: room.deadband_hum ?? 2.0,
      tempDbError: null,
      humDbError: null,
      saving: false,
      saveError: null,
    })
    setResetCount(c => c + 1)
  }, [open, room])

  if (!open) return null
  if (!capabilities?.temperature && !capabilities?.humidity) return null

  const handleSave = async () => {
    const tErr = validateTempDb(capabilities.temperature ? state.tempDb : 0.5)
    const hErr = validateHumDb(capabilities.humidity ? state.humDb : 2.0)

    let hasError = false
    if (capabilities.temperature && tErr) {
      setState(s => ({ ...s, tempDbError: tErr }))
      hasError = true
    }
    if (capabilities.humidity && hErr) {
      setState(s => ({ ...s, humDbError: hErr }))
      hasError = true
    }
    if (hasError) return

    setState(s => ({ ...s, saving: true, saveError: null }))
    try {
      await onSave({ deadband_temp: state.tempDb, deadband_hum: state.humDb })
    } catch {
      setState(s => ({ ...s, saveError: 'Something went wrong.' }))
    } finally {
      setState(s => ({ ...s, saving: false }))
    }
  }

  const hasErrors = Boolean(state.tempDbError) || Boolean(state.humDbError)

  return (
    <div className="cc-modal-bg" onClick={onClose}>
      <div className="cc-modal" onClick={(e) => e.stopPropagation()}>
        <div className="cc-modal-head">
          <h2 className="cc-h2">Tolerances</h2>
        </div>

        <div className="cc-modal-body" style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
          {capabilities.temperature && (
            <div>
              <label className="cc-label" style={{ display: 'block', marginBottom: 6 }}>
                Temperature tolerance
              </label>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <input
                  key={`temp-db-${resetCount}`}
                  className="cc-input cc-input--mono"
                  defaultValue={state.tempDb.toFixed(1)}
                  onBlur={(e) => {
                    const val = parseFloat(e.target.value)
                    const err = validateTempDb(isNaN(val) ? null : val)
                    if (!isNaN(val)) setState(s => ({ ...s, tempDb: val }))
                    setState(s => ({ ...s, tempDbError: err }))
                  }}
                  style={{
                    width: 100,
                    textAlign: 'right',
                    borderColor: state.tempDbError ? 'var(--cc-danger)' : undefined,
                  }}
                />
                <span className="cc-meta">°C</span>
              </div>
              {showHints && tempTarget != null && !state.tempDbError && (
                <p className="cc-meta" style={{ color: 'var(--cc-fg-3)', marginTop: 4 }}>
                  Heater turns on below {(tempTarget - state.tempDb).toFixed(1)}°C, off above{' '}
                  {(tempTarget + state.tempDb).toFixed(1)}°C
                </p>
              )}
              {state.tempDbError && (
                <p className="cc-meta" style={{ color: 'var(--cc-danger)', marginTop: 4 }}>
                  {state.tempDbError}
                </p>
              )}
            </div>
          )}

          {capabilities.humidity && (
            <div>
              <label className="cc-label" style={{ display: 'block', marginBottom: 6 }}>
                Humidity tolerance
              </label>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <input
                  key={`hum-db-${resetCount}`}
                  className="cc-input cc-input--mono"
                  defaultValue={state.humDb.toFixed(1)}
                  onBlur={(e) => {
                    const val = parseFloat(e.target.value)
                    const err = validateHumDb(isNaN(val) ? null : val)
                    if (!isNaN(val)) setState(s => ({ ...s, humDb: val }))
                    setState(s => ({ ...s, humDbError: err }))
                  }}
                  style={{
                    width: 100,
                    textAlign: 'right',
                    borderColor: state.humDbError ? 'var(--cc-danger)' : undefined,
                  }}
                />
                <span className="cc-meta">%</span>
              </div>
              {showHints && humTarget != null && !state.humDbError && (
                <p className="cc-meta" style={{ color: 'var(--cc-fg-3)', marginTop: 4 }}>
                  Humidifier turns on below {(humTarget - state.humDb).toFixed(1)}%, off above{' '}
                  {(humTarget + state.humDb).toFixed(1)}%
                </p>
              )}
              {state.humDbError && (
                <p className="cc-meta" style={{ color: 'var(--cc-danger)', marginTop: 4 }}>
                  {state.humDbError}
                </p>
              )}
            </div>
          )}

          <p className="cc-meta" style={{ color: 'var(--cc-fg-4)' }}>
            Wider tolerances save energy but allow more drift
          </p>
        </div>

        <div className="cc-modal-foot" style={{ flexDirection: 'column', alignItems: 'stretch' }}>
          {state.saveError && (
            <p className="cc-meta" style={{ color: 'var(--cc-danger)', marginBottom: 8, textAlign: 'right' }}>
              {state.saveError}
            </p>
          )}
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <button className="cc-btn cc-btn--ghost" onClick={onClose}>
              Cancel
            </button>
            <button
              className={`cc-btn ${hasErrors ? 'cc-btn--ghost' : 'cc-btn--primary'}`}
              onClick={handleSave}
              disabled={state.saving || !!state.tempDbError || !!state.humDbError}
            >
              {state.saving ? 'Saving…' : 'Save'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
