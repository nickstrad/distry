export class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export async function api(path, options = {}) {
  return requestJSON(path, options)
}

export async function apiMaybe(path, options = {}) {
  return requestJSON(path, { ...options, allowNotFound: true })
}

async function requestJSON(path, { allowNotFound = false, ...options } = {}) {
  const fetchOptions = Object.keys(options).length ? options : undefined
  const res = await fetch(path, fetchOptions)
  const data = await res.json().catch(() => null)

  if (allowNotFound && res.status === 404) {
    return null
  }
  if (res.status === 401 && window.location.pathname !== '/login') {
    window.location.assign('/login')
  }

  if (!res.ok) {
    throw new ApiError(data?.error || 'Something went wrong', res.status)
  }
  return data
}
