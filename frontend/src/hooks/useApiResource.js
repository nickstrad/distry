import { useEffect, useState } from 'react'
import { api } from '../api.js'

const initialResource = { data: null, loading: true, error: '' }

export function useApiResource(path) {
  const [state, setState] = useState(initialResource)

  useEffect(() => {
    let active = true
    setState(initialResource)

    api(path)
      .then((data) => {
        if (active) setState({ data, loading: false, error: '' })
      })
      .catch((err) => {
        if (active) setState({ data: null, loading: false, error: err.message })
      })

    return () => {
      active = false
    }
  }, [path])

  return state
}
