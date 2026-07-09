export class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export async function api(path, options) {
  const res = await fetch(path, options)
  const data = await res.json().catch(() => null)

  if (res.status === 401 && window.location.pathname !== '/login') {
    window.location.assign('/login')
  }

  if (!res.ok) {
    throw new ApiError(data?.error || 'Something went wrong', res.status)
  }
  return data
}
