import { useEffect, useState } from 'react'

export default function App() {
  const [message, setMessage] = useState('loading...')

  useEffect(() => {
    fetch('/api/hello')
      .then((res) => res.json())
      .then((data) => setMessage(data.message))
      .catch(() => setMessage('failed to reach backend'))
  }, [])

  return (
    <main style={{ fontFamily: 'system-ui, sans-serif', padding: '2rem' }}>
      <h1>Hello World</h1>
      <p>Message from the chi backend: <strong>{message}</strong></p>
    </main>
  )
}
