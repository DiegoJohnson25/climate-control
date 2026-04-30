import useSWR from 'swr'
import { fetcher } from '../api/fetcher'

export function useDesiredState(roomId) {
  const { data, error, isLoading, mutate } = useSWR(
    roomId ? `/api/v1/rooms/${roomId}/desired-state` : null,
    fetcher,
    { revalidateOnFocus: false }
  )

  return { desiredState: data, error, isLoading, mutate }
}
