import useSWR from 'swr'
import { fetcher } from '@/api/fetcher'

export function useUser() {
  const { data, error, isLoading, mutate } = useSWR('/api/v1/users/me', fetcher)
  return { user: data, error, isLoading, mutate }
}
