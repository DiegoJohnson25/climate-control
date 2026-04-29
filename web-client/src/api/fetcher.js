import { doRefresh, getToken, clearToken } from './auth.jsx'

// ---------------------------------------------------------------------------
// SWR global fetcher
// Configured once in App.jsx via <SWRConfig value={{ fetcher }}>.
// All useSWR hooks in the app inherit this automatically.
//
// Flow:
//   1. Attach Authorization header and send request
//   2. On 401 — attempt one silent refresh then retry the original request
//   3. If retry also 401s — clear token and redirect to login
//   4. On other non-ok responses — throw with status code for SWR error state
// ---------------------------------------------------------------------------

export async function fetcher(url) {
  const res = await sendRequest(url)

  if (res.status === 401) {
    try {
      await doRefresh()
    } catch {
      clearToken()
      window.location.href = '/login'
      throw new Error('Unauthorized')
    }

    const retry = await sendRequest(url)

    if (retry.status === 401) {
      clearToken()
      window.location.href = '/login'
      throw new Error('Unauthorized')
    }

    if (!retry.ok) throw new Error(`Request failed: ${retry.status}`)
    return retry.json()
  }

  if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  return res.json()
}

// Sends a single GET request with the current access token attached.
function sendRequest(url) {
  return fetch(url, {
    credentials: 'include',
    headers: {
      Authorization: `Bearer ${getToken()}`,
    },
  })
}
