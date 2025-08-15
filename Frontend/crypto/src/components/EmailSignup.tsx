import { useState } from 'react'
import { ApiService } from '../services/api'

export const EmailSignup = () => {
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  const handleSubscribe = async (e: React.FormEvent) => {
    e.preventDefault()
    setMessage('')
    setError('')

    const emailTrimmed = email.trim()
    if (!emailTrimmed || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(emailTrimmed)) {
      setError('Please enter a valid email address')
      return
    }

    setLoading(true)
    try {
      const res = await ApiService.subscribeToReports(emailTrimmed)
      setMessage(res.message || 'Subscription successful')
      setEmail('')
    } catch (err: any) {
      setError(err?.message || 'Failed to subscribe')
    } finally {
      setLoading(false)
    }
  }

  const handleUnsubscribe = async () => {
    setMessage('')
    setError('')
    const emailTrimmed = email.trim()
    if (!emailTrimmed || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(emailTrimmed)) {
      setError('Please enter a valid email address to unsubscribe')
      return
    }

    setLoading(true)
    try {
      const res = await ApiService.unsubscribeFromReports(emailTrimmed)
      setMessage(res.message || 'Successfully unsubscribed')
      setEmail('')
    } catch (err: any) {
      setError(err?.message || 'Failed to unsubscribe')
    } finally {
      setLoading(false)
    }
  }

  const handleBack = () => {
    window.history.back()
  }

  return (
    <div className="card" style={{ maxWidth: 560, margin: '0 auto' }}>
      <div className="card-header">
        <h2>Daily Email Reports</h2>
        <span className="time-range">Free</span>
      </div>
      <p style={{ color: '#94a3b8', marginBottom: '1rem' }}>
        Get a daily summary of crypto prices and trends delivered to your inbox.
      </p>
      <form onSubmit={handleSubscribe} className="timestamp-form" style={{ marginBottom: 0 }}>
        <input
          type="email"
          className="timestamp-input"
          placeholder="you@example.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          aria-label="Email address"
        />
        <button className="submit-button" type="submit" disabled={loading}>
          {loading ? 'Processing…' : 'Subscribe'}
        </button>
      </form>

      <button
        type="button"
        onClick={handleUnsubscribe}
        disabled={loading}
        style={{
          marginTop: '0.75rem',
          padding: '0.5rem 1rem',
          background: '#ff4136',
          color: '#ffffff',
          border: 'none',
          borderRadius: '0.375rem',
          cursor: 'pointer',
        }}
      >
        {loading ? 'Processing…' : 'Unsubscribe'}
      </button>

      {message && (
        <div className="success-message" style={{ marginTop: '1rem', color: '#10b981' }}>
          {message}
        </div>
      )}
      {error && (
        <div className="error-message" style={{ marginTop: '1rem' }}>
          {error}
        </div>
      )}

      <div style={{ marginTop: '1.5rem' }}>
        <button
          type="button"
          onClick={handleBack}
          style={{
            padding: '0.5rem 1rem',
            background: '#001f3f',
            color: '#ffffff',
            border: 'none',
            borderRadius: '0.375rem',
            cursor: 'pointer',
          }}
        >
          ← Back
        </button>
      </div>
    </div>
  )
}
