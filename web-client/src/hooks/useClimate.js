import useSWR from 'swr'
import { getToken, doRefresh } from '@/api/auth.jsx'

export function useClimate(roomId) {
  const { data, error, isLoading } = useSWR(
    roomId ? `/api/v1/rooms/${roomId}/climate` : null,
    async (url) => {
      const res = await fetch(url, {
        credentials: 'include',
        headers: { Authorization: `Bearer ${getToken()}` },
      })
      if (res.status === 204) return null
      if (res.status === 401) {
        await doRefresh()
        const retry = await fetch(url, {
          credentials: 'include',
          headers: { Authorization: `Bearer ${getToken()}` },
        })
        if (retry.status === 204) return null
        if (!retry.ok) throw new Error(retry.status)
        return retry.json()
      }
      if (!res.ok) throw new Error(res.status)
      return res.json()
    },
    { refreshInterval: 30000 }
  )
  return { climate: data ?? null, error, isLoading }
}
