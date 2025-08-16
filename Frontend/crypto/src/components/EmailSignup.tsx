import { useState } from 'react'
import { ApiService } from '../services/api'

export const EmailSignup = () => {
  const [subEmail, setSubEmail] = useState('')
  const [loadingSubscribe, setLoadingSubscribe] = useState(false)
  const [subMessage, setSubMessage] = useState('')
  const [subError, setSubError] = useState('')

  const [unsubEmail, setUnsubEmail] = useState('')
  const [loadingUnsubscribe, setLoadingUnsubscribe] = useState(false)
  const [unsubMessage, setUnsubMessage] = useState('')
  const [unsubError, setUnsubError] = useState('')

  const handleSubscribe = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubMessage('')
    setSubError('')
    const emailTrimmed = subEmail.trim()
    if (!emailTrimmed || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(emailTrimmed)) {
      setSubError('Please enter a valid email address')
      return
    }

    setLoadingSubscribe(true)
    try {
      await ApiService.subscribeToReports(emailTrimmed)
      setSubMessage('Subscription initiated! Please check your email to verify.')
      setSubEmail('')
    } catch (err: any) {
      setSubError(err?.message || 'Failed to subscribe')
    } finally {
      setLoadingSubscribe(false)
    }
  }

  const handleUnsubscribe = async (e: React.FormEvent) => {
    e.preventDefault()
    setUnsubMessage('')
    setUnsubError('')
    const emailTrimmed = unsubEmail.trim()
    if (!emailTrimmed || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(emailTrimmed)) {
      setUnsubError('Please enter a valid email address')
      return
    }

    setLoadingUnsubscribe(true)
    try {
      await ApiService.unsubscribeFromReports(emailTrimmed)
      setUnsubMessage('Unsubscribe request received! Please check your email to verify.')
      setUnsubEmail('')
    } catch (err: any) {
      setUnsubError(err?.message || 'Failed to unsubscribe')
    } finally {
      setLoadingUnsubscribe(false)
    }
  }

  const handleBack = () => {
    window.history.back()
  }

  return (
    <div style={{ display: 'grid', gap: '1.5rem', maxWidth: 560, margin: '0 auto' }}>
      <div className="card">
        <div className="card-header">
          <h2>Subscribe to Daily Email Reports</h2>
          <span className="time-range">Free</span>
        </div>
        <p style={{ color: '#94a3b8', marginBottom: '1rem' }}>
          Get a daily summary of crypto prices and trends delivered to your inbox.
        </p>
        <form onSubmit={handleSubscribe} className="timestamp-form">
          <input
            type="email"
            className="timestamp-input"
            placeholder="you@example.com"
            value={subEmail}
            onChange={(e) => setSubEmail(e.target.value)}
            aria-label="Subscribe email"
          />
          <button className="submit-button" type="submit" disabled={loadingSubscribe}>
            {loadingSubscribe ? 'Processing…' : 'Subscribe'}
          </button>
        </form>
        {subMessage && <div style={{ color: '#10b981', marginTop: '0.5rem' }}>{subMessage}</div>}
        {subError && <div style={{ color: '#ef4444', marginTop: '0.5rem' }}>{subError}</div>}
      </div>

      <div className="card">
        <div className="card-header">
          <h2>Unsubscribe</h2>
        </div>
        <p style={{ color: '#94a3b8', marginBottom: '1rem' }}>
          Don’t want to receive daily reports anymore? Enter your email to unsubscribe.
        </p>
        <form onSubmit={handleUnsubscribe} className="timestamp-form">
          <input
            type="email"
            className="timestamp-input"
            placeholder="you@example.com"
            value={unsubEmail}
            onChange={(e) => setUnsubEmail(e.target.value)}
            aria-label="Unsubscribe email"
          />
          <button
            type="submit"
            disabled={loadingUnsubscribe}
            style={{
              padding: '0.5rem 1rem',
              background: '#ff4136',
              color: '#ffffff',
              border: 'none',
              borderRadius: '0.375rem',
              cursor: 'pointer',
            }}
          >
            {loadingUnsubscribe ? 'Processing…' : 'Unsubscribe'}
          </button>
        </form>
        {unsubMessage && <div style={{ color: '#10b981', marginTop: '0.5rem' }}>{unsubMessage}</div>}
        {unsubError && <div style={{ color: '#ef4444', marginTop: '0.5rem' }}>{unsubError}</div>}
      </div>

      <div style={{ marginTop: '-0.5rem', textAlign: 'center' }}>
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
