import useSWR from 'swr'
import { fetcher } from '@/api/fetcher'

export function useSchedules(roomId) {
  const { data, error, isLoading } = useSWR(
    roomId ? `/api/v1/rooms/${roomId}/schedules` : null,
    fetcher
  )
  return { schedules: data ?? [], error, isLoading }
}
