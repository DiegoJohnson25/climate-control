import { useState } from 'react'
import { Plus, X } from 'lucide-react'
import { useUser } from '@/hooks/useUser'
import { useRooms } from '@/hooks/useRooms'
import { updateMe } from '@/api/users'
import { getToken } from '@/api/auth.jsx'
import TimezonePrompt from '@/components/TimezonePrompt'
import RoomCard from '@/components/RoomCard'

// ---------------------------------------------------------------------------
// DashboardPage
// Room grid with timezone prompt and add-room modal. Phase 6b.
// ---------------------------------------------------------------------------

export default function DashboardPage() {
  const { user, mutate: mutateUser } = useUser()
  const { rooms, isLoading: roomsLoading, mutate: mutateRooms } = useRooms()

  const [addOpen, setAddOpen] = useState(false)
  const [addValue, setAddValue] = useState('')
  const [addError, setAddError] = useState(null)
  const [addLoading, setAddLoading] = useState(false)

  async function handleAddRoom() {
    setAddError(null)
    setAddLoading(true)
    try {
      const res = await fetch('/api/v1/rooms', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${getToken()}`,
        },
        credentials: 'include',
        body: JSON.stringify({ name: addValue.trim() }),
      })
      if (res.status === 201) {
        mutateRooms()
        setAddOpen(false)
        setAddValue('')
      } else if (res.status === 409) {
        setAddError('A room with that name already exists.')
      } else {
        setAddError('Something went wrong.')
      }
    } catch {
      setAddError('Something went wrong.')
    } finally {
      setAddLoading(false)
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
      <TimezonePrompt
        userTimezone={user?.timezone}
        onSave={async (timezone) => {
          await updateMe({ timezone })
          mutateUser()
        }}
      />

      {/* Page header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'baseline',
          justifyContent: 'space-between',
          marginBottom: 24,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 12 }}>
          <h1 className="cc-h1">Rooms</h1>
          <span
            style={{
              fontFamily: 'var(--cc-font-mono)',
              fontSize: 13,
              color: 'var(--cc-fg-3)',
            }}
          >
            {rooms.length} total
          </span>
        </div>
        <button
          className="cc-btn cc-btn--primary"
          style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}
          onClick={() => setAddOpen(true)}
        >
          <Plus size={14} />
          Add Room
        </button>
      </div>

      {!roomsLoading && rooms.length > 0 && (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
            gap: 16,
            alignItems: 'stretch',
          }}
        >
          {rooms.map((room) => (
            <RoomCard key={room.id} room={room} />
          ))}
        </div>
      )}

      {/* Add room modal */}
      {addOpen && (
        <div className="cc-modal-bg" onClick={() => setAddOpen(false)}>
          <div className="cc-modal" onClick={(e) => e.stopPropagation()}>
            <div className="cc-modal-head">
              <h2 className="cc-h2">Add room</h2>
              <div style={{ flex: 1 }} />
              <button className="cc-iconbtn" onClick={() => setAddOpen(false)}>
                <X size={16} />
              </button>
            </div>

            <div className="cc-modal-body">
              <label className="cc-label" style={{ display: 'block', marginBottom: 6 }}>
                Room name
              </label>
              <input
                className="cc-input"
                value={addValue}
                onChange={(e) => setAddValue(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleAddRoom()}
                autoFocus
              />
              {addError && (
                <span
                  className="cc-meta"
                  style={{ color: 'var(--cc-danger-fg)', marginTop: 6, display: 'block' }}
                >
                  {addError}
                </span>
              )}
            </div>

            <div className="cc-modal-foot">
              <button className="cc-btn cc-btn--secondary" onClick={() => setAddOpen(false)}>
                Cancel
              </button>
              <button
                className="cc-btn cc-btn--primary"
                onClick={handleAddRoom}
                disabled={addLoading || addValue.trim() === ''}
              >
                {addLoading ? 'Creating…' : 'Create Room'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
