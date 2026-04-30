import { getToken } from '@/api/auth.jsx'

// updateMe sends a partial profile update for the authenticated user.
// Only fields present in the payload are applied — currently supports timezone.
export async function updateMe(payload) {
  const res = await fetch("/api/v1/users/me", {
    method: "PUT",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${getToken()}`,
    },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error(res.status);
}
