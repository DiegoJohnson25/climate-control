import { useState, useRef, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { ChevronLeft, Pencil, MoreVertical, X } from 'lucide-react'
import { useRoom } from '@/hooks/useRoom'
import { useRooms } from '@/hooks/useRooms'
import OverviewTab from '@/pages/tabs/OverviewTab'
import { getToken } from '@/api/auth.jsx'

// ---------------------------------------------------------------------------
// RoomDetailPage
// Room detail with tab bar, inline rename, and delete. Rename and delete are
// accessible via the pencil button and the kebab menu. Tab content components
// are loaded per-phase — history (6d), schedules (6e), devices (6f).
// ---------------------------------------------------------------------------

const TABS = [
  { key: 'overview',  label: 'Overview' },
  { key: 'history',   label: 'History' },
  { key: 'schedules', label: 'Schedules' },
  { key: 'devices',   label: 'Devices' },
]

// ---------------------------------------------------------------------------
// TabButton
// ---------------------------------------------------------------------------

function TabButton({ label, active, onClick }) {
  return (
    <button
      onClick={onClick}
      style={{
        position: 'relative',
        background: 'transparent',
        border: 'none',
        padding: '10px 16px',
        cursor: 'pointer',
        fontSize: 'var(--cc-fs-sm)',
        fontWeight: 'var(--cc-fw-medium)',
        color: active ? 'var(--cc-fg)' : 'var(--cc-fg-3)',
        transition: `color var(--cc-dur-fast) var(--cc-ease)`,
        fontFamily: 'var(--cc-font-sans)',
      }}
    >
      {label}
      {active && (
        <div
          style={{
            position: 'absolute',
            bottom: -1,
            left: 0,
            right: 0,
            height: 2,
            background: 'var(--cc-fg)',
          }}
        />
      )}
    </button>
  )
}

// ---------------------------------------------------------------------------
// RoomDetailPage
// ---------------------------------------------------------------------------

export default function RoomDetailPage() {
  const { id: roomId } = useParams()
  const navigate = useNavigate()
  const { room, mutate: mutateRoom } = useRoom(roomId)
  const { mutate: mutateRooms } = useRooms()

  const [tab, setTab] = useState('overview')

  const [renameOpen, setRenameOpen] = useState(false)
  const [renameValue, setRenameValue] = useState('')
  const [renameError, setRenameError] = useState(null)
  const [renameLoading, setRenameLoading] = useState(false)

  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleteLoading, setDeleteLoading] = useState(false)
  const [deleteError, setDeleteError] = useState(null)

  const [kebabOpen, setKebabOpen] = useState(false)
  const kebabRef = useRef(null)

  const [backHovered, setBackHovered] = useState(false)

  // Close kebab on outside click.
  useEffect(() => {
    if (!kebabOpen) return
    function onMouseDown(e) {
      if (kebabRef.current && !kebabRef.current.contains(e.target)) {
        setKebabOpen(false)
      }
    }
    document.addEventListener('mousedown', onMouseDown)
    return () => document.removeEventListener('mousedown', onMouseDown)
  }, [kebabOpen])

  async function handleRename() {
    setRenameError(null)
    setRenameLoading(true)
    try {
      const res = await fetch(`/api/v1/rooms/${roomId}`, {
        method: 'PUT',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${getToken()}`,
        },
        body: JSON.stringify({
          name: renameValue.trim(),
          deadband_temp: room.deadband_temp,
          deadband_hum: room.deadband_hum,
        }),
      })
      if (res.status === 409) {
        setRenameError('A room with that name already exists.')
        return
      }
      if (!res.ok) {
        setRenameError('Something went wrong.')
        return
      }
      mutateRoom()
      setRenameOpen(false)
    } catch {
      setRenameError('Something went wrong.')
    } finally {
      setRenameLoading(false)
    }
  }

  async function handleDelete() {
    setDeleteError(null)
    setDeleteLoading(true)
    try {
      const res = await fetch(`/api/v1/rooms/${roomId}`, {
        method: 'DELETE',
        credentials: 'include',
        headers: { Authorization: `Bearer ${getToken()}` },
      })
      if (!res.ok) {
        setDeleteError('Something went wrong.')
        return
      }
      await mutateRooms()
      navigate('/dashboard')
    } catch {
      setDeleteError('Something went wrong.')
    } finally {
      setDeleteLoading(false)
    }
  }

  return (
    <div
      style={{
        maxWidth: 'var(--cc-max-width)',
        margin: '0 auto',
        padding: '32px var(--cc-page-pad-x)',
      }}
    >
      {/* Back link */}
      <button
        onClick={() => navigate('/dashboard')}
        onMouseEnter={() => setBackHovered(true)}
        onMouseLeave={() => setBackHovered(false)}
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 5,
          marginBottom: 20,
          background: 'transparent',
          border: 'none',
          cursor: 'pointer',
          padding: 0,
          color: backHovered ? 'var(--cc-fg)' : 'var(--cc-fg-3)',
          transition: `color var(--cc-dur-fast) var(--cc-ease)`,
        }}
      >
        <ChevronLeft size={12} />
        <span style={{ fontSize: 12 }}>Dashboard</span>
      </button>

      {/* Header row */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 10,
          marginBottom: 24,
        }}
      >
        <h1 className="cc-h1">{room?.name ?? '—'}</h1>

        <button
          className="cc-iconbtn"
          onClick={() => {
            setRenameValue(room?.name ?? '')
            setRenameOpen(true)
          }}
          title="Rename room"
        >
          <Pencil size={14} />
        </button>

        <div style={{ flex: 1 }} />

        {/* Kebab menu */}
        <div ref={kebabRef} style={{ position: 'relative' }}>
          <button
            className="cc-iconbtn"
            onClick={() => setKebabOpen((o) => !o)}
            title="More options"
          >
            <MoreVertical size={16} />
          </button>

          {kebabOpen && (
            <div
              className="cc-pop"
              style={{ position: 'absolute', top: 'calc(100% + 6px)', right: 0, width: 180 }}
            >
              <button
                onClick={() => {
                  setKebabOpen(false)
                  setRenameValue(room?.name ?? '')
                  setRenameOpen(true)
                }}
              >
                Edit Name
              </button>
              <hr />
              <button
                className="danger"
                onClick={() => {
                  setKebabOpen(false)
                  setDeleteOpen(true)
                }}
              >
                Delete Room
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Tab bar */}
      <div
        style={{
          display: 'flex',
          gap: 0,
          borderBottom: '1px solid var(--cc-border)',
          marginBottom: 24,
        }}
      >
        {TABS.map((t) => (
          <TabButton
            key={t.key}
            label={t.label}
            active={tab === t.key}
            onClick={() => setTab(t.key)}
          />
        ))}
      </div>

      {/* Tab content */}
      {tab === 'overview' && <OverviewTab roomId={roomId} />}
      {tab === 'history' && (
        <div style={{ padding: '24px 0' }}>
          <p className="cc-body">History — coming in 6d</p>
        </div>
      )}
      {tab === 'schedules' && (
        <div style={{ padding: '24px 0' }}>
          <p className="cc-body">Schedules — coming in 6e</p>
        </div>
      )}
      {tab === 'devices' && (
        <div style={{ padding: '24px 0' }}>
          <p className="cc-body">Devices — coming in 6f</p>
        </div>
      )}

      {/* Rename modal */}
      {renameOpen && (
        <div className="cc-modal-bg" onClick={() => setRenameOpen(false)}>
          <div className="cc-modal" onClick={(e) => e.stopPropagation()}>
            <div className="cc-modal-head">
              <h2 className="cc-h2">Rename room</h2>
              <div style={{ flex: 1 }} />
              <button className="cc-iconbtn" onClick={() => setRenameOpen(false)}>
                <X size={16} />
              </button>
            </div>

            <div className="cc-modal-body">
              <span className="cc-label" style={{ display: 'block', marginBottom: 6 }}>
                Room name
              </span>
              <input
                className="cc-input"
                value={renameValue}
                onChange={(e) => setRenameValue(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') handleRename() }}
                autoFocus
              />
              {renameError && (
                <span
                  className="cc-meta"
                  style={{ color: 'var(--cc-danger-fg)', marginTop: 6, display: 'block' }}
                >
                  {renameError}
                </span>
              )}
            </div>

            <div className="cc-modal-foot">
              <button className="cc-btn cc-btn--secondary" onClick={() => setRenameOpen(false)}>
                Cancel
              </button>
              <button
                className="cc-btn cc-btn--primary"
                onClick={handleRename}
                disabled={renameLoading || renameValue.trim() === ''}
              >
                {renameLoading ? 'Saving…' : 'Save'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete modal */}
      {deleteOpen && (
        <div className="cc-modal-bg" onClick={() => setDeleteOpen(false)}>
          <div className="cc-modal" onClick={(e) => e.stopPropagation()}>
            <div className="cc-modal-head">
              <h2 className="cc-h2">Delete Room</h2>
              <div style={{ flex: 1 }} />
              <button className="cc-iconbtn" onClick={() => setDeleteOpen(false)}>
                <X size={16} />
              </button>
            </div>

            <div className="cc-modal-body">
              <p className="cc-body">
                Are you sure you want to delete <strong>{room?.name}</strong>?
              </p>
              <p className="cc-body" style={{ marginTop: 8 }}>
                This will unassign all devices in the room. Schedules and desired
                state will be permanently deleted.
              </p>
              {deleteError && (
                <span
                  className="cc-meta"
                  style={{ color: 'var(--cc-danger-fg)', marginTop: 12, display: 'block' }}
                >
                  {deleteError}
                </span>
              )}
            </div>

            <div className="cc-modal-foot">
              <button className="cc-btn cc-btn--secondary" onClick={() => setDeleteOpen(false)}>
                Cancel
              </button>
              <button
                className="cc-btn cc-btn--danger"
                onClick={handleDelete}
                disabled={deleteLoading}
              >
                {deleteLoading ? 'Deleting…' : 'Delete Room'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
