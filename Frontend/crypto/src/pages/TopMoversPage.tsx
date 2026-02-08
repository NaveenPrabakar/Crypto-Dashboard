import { useState, useEffect } from 'react'
import { ApiService } from '../services/api'
import type { TopMoverData } from '../types'
import { TrendingUp, RefreshCw } from 'lucide-react'

const MINUTES_OPTIONS = [
  { label: '1 hour', value: 60 },
  { label: '6 hours', value: 360 },
  { label: '24 hours', value: 1440 },
  { label: '7 days', value: 10080 },
]

export const TopMoversPage = () => {
  const [movers, setMovers] = useState<TopMoverData[]>([])
  const [minutes, setMinutes] = useState(1440)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const fetchMovers = async () => {
    setLoading(true)
    setError('')
    try {
      const data = await ApiService.getTopMovers(minutes)
      setMovers(data)
    } catch (err) {
      setError('Failed to fetch top movers')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchMovers()
  }, [minutes])

  const formatPrice = (p: number) =>
    new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 4 }).format(p)

  return (
    <div className="top-movers-page">
      <div className="page-header">
        <h1 className="page-title">
          <TrendingUp className="page-title-icon" />
          Top Movers
        </h1>
        <p className="page-subtitle">
          Biggest price moves across all tracked coins. Use the window to compare momentum over different periods.
        </p>
      </div>

      <div className="main-controls">
        <div className="time-controls">
          <span className="ml-label">Window</span>
          <select
            value={minutes}
            onChange={(e) => setMinutes(Number(e.target.value))}
            className="time-select"
          >
            {MINUTES_OPTIONS.map(({ label, value }) => (
              <option key={value} value={value}>{label}</option>
            ))}
          </select>
        </div>
        <button onClick={fetchMovers} className="control-button" disabled={loading}>
          <RefreshCw className={loading ? 'refresh-icon spinning' : 'refresh-icon'} style={{ width: 16, height: 16, marginRight: 6 }} />
          Refresh
        </button>
      </div>

      {error && <div className="error-message">{error}</div>}
      {loading && <div className="loading">Loading top movers…</div>}

      {!loading && movers.length > 0 && (
        <div className="card top-movers-card">
          <div className="top-movers-list top-movers-list-page">
            {movers.map((m, i) => (
              <div key={`${m.coin_id}-${i}`} className="mover-item mover-item-large">
                <span className="mover-rank">#{i + 1}</span>
                <span className="mover-coin">{m.coin_id}</span>
                <span className="mover-prices">
                  {formatPrice(m.start_price)} → {formatPrice(m.end_price)}
                </span>
                <span className={`mover-change ${m.percent_change >= 0 ? 'positive' : 'negative'}`}>
                  {m.percent_change >= 0 ? '+' : ''}{m.percent_change.toFixed(2)}%
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {!loading && movers.length === 0 && !error && (
        <div className="loading">No movers data for this window.</div>
      )}
    </div>
  )
}
