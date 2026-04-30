import { getToken } from '@/api/auth'

export async function updateDesiredState(roomId, payload) {
  const res = await fetch(`/api/v1/rooms/${roomId}/desired-state`, {
    method: "PUT",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${getToken()}`,
    },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    const err = new Error(body.error || res.status)
    err.status = res.status
    throw err
  }
}
