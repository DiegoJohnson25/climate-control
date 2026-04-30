import useSWR from 'swr'
import { fetcher } from '@/api/fetcher'

export function useRoom(roomId) {
  const { data, error, isLoading, mutate } = useSWR(
    roomId ? `/api/v1/rooms/${roomId}` : null,
    fetcher
  )
  return { room: data, error, isLoading, mutate }
}
