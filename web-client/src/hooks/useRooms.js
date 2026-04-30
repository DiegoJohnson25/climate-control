import useSWR from 'swr'
import { fetcher } from '@/api/fetcher'

export function useRooms() {
  const { data, error, isLoading, mutate } = useSWR('/api/v1/rooms', fetcher, {
    refreshInterval: 30000,
  })
  return { rooms: data ?? [], error, isLoading, mutate }
}
