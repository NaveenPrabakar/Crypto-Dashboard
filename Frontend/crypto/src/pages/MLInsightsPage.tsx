import { useState, useEffect } from 'react'
import { ApiService } from '../services/api'
import { getCustomTimeRanges } from '../services/utils'
import type { TrendData, VolatilityData, TopMoverData, PriceData, PredictData } from '../types'
import { Brain, TrendingUp, Activity, Zap, MessageSquare, Target } from 'lucide-react'

const COINS = ['bitcoin', 'ethereum', 'cardano', 'solana', 'polkadot', 'chainlink']
const HORIZONS = [
  { label: '1 hour', minutes: 60 },
  { label: '6 hours', minutes: 360 },
  { label: '24 hours', minutes: 1440 },
]

export const MLInsightsPage = () => {
  const [selectedCoin, setSelectedCoin] = useState('bitcoin')
  const [trend, setTrend] = useState<TrendData | null>(null)
  const [volatility, setVolatility] = useState<VolatilityData | null>(null)
  const [topMovers, setTopMovers] = useState<TopMoverData[]>([])
  const [prediction, setPrediction] = useState<PredictData | null>(null)
  const [predictionLoading, setPredictionLoading] = useState(false)
  const [horizonMinutes, setHorizonMinutes] = useState(60)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [aiQuestion, setAiQuestion] = useState('')
  const [aiLoading, setAiLoading] = useState(false)
  const [aiResults, setAiResults] = useState<PriceData[] | null>(null)
  const [aiError, setAiError] = useState('')

  const timeRanges = getCustomTimeRanges()

  const fetchSignals = async (start: string, end: string) => {
    setLoading(true)
    setError('')
    try {
      const [trendData, volData, moversData] = await Promise.all([
        ApiService.getTrend(selectedCoin, start, end),
        ApiService.getVolatility(selectedCoin, start, end),
        ApiService.getTopMovers(1440),
      ])
      setTrend(trendData)
      setVolatility(volData)
      setTopMovers(moversData)
    } catch (err) {
      setError('Failed to load ML signals')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const fetchPrediction = async () => {
    setPredictionLoading(true)
    try {
      const data = await ApiService.getPredict(selectedCoin, horizonMinutes)
      setPrediction(data)
    } catch (err) {
      setPrediction(null)
      console.error(err)
    } finally {
      setPredictionLoading(false)
    }
  }

  useEffect(() => {
    const range = timeRanges[1]
    if (range) fetchSignals(range.start, range.end)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedCoin])

  useEffect(() => {
    fetchPrediction()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedCoin, horizonMinutes])

  const handleAskAI = async (e: React.FormEvent) => {
    e.preventDefault()
    const q = aiQuestion.trim()
    if (!q) return
    setAiLoading(true)
    setAiError('')
    setAiResults(null)
    try {
      const data = await ApiService.askAI(q)
      setAiResults(data)
      setAiQuestion('')
    } catch (err: unknown) {
      setAiError(err instanceof Error ? err.message : 'AI query failed')
      console.error(err)
    } finally {
      setAiLoading(false)
    }
  }

  const formatPrice = (p: number) =>
    new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(p)

  return (
    <div className="ml-page">
      <div className="page-header">
        <h1 className="page-title">
          <Brain className="page-title-icon" />
          ML Insights
        </h1>
        <p className="page-subtitle">
          Backend ML predicts future price and range from linear regression on history. Plus trend, volatility, momentum & AI query.
        </p>
      </div>

      {/* ML Price Prediction (backend) */}
      <div className="card ml-card ml-predict-section">
        <div className="card-header">
          <Target className="card-icon" />
          <h2>ML Price Prediction</h2>
        </div>
        <div className="ml-controls">
          <label className="ml-label">Horizon</label>
          <select
            value={horizonMinutes}
            onChange={(e) => setHorizonMinutes(Number(e.target.value))}
            className="time-range-select"
          >
            {HORIZONS.map((h) => (
              <option key={h.minutes} value={h.minutes}>{h.label}</option>
            ))}
          </select>
          <button type="button" className="control-button" onClick={fetchPrediction} disabled={predictionLoading}>
            {predictionLoading ? 'Predicting…' : 'Refresh'}
          </button>
        </div>
        {prediction && (
          <div className="predict-results">
            <div className="predict-main">
              <span className="predict-label">Predicted price</span>
              <span className="predict-price">{formatPrice(prediction.predicted_price)}</span>
              <span className={`predict-trend trend-${prediction.trend.toLowerCase() === 'uptrend' ? 'positive' : prediction.trend.toLowerCase() === 'downtrend' ? 'negative' : 'neutral'}`}>
                {prediction.trend}
              </span>
            </div>
            <div className="predict-range">
              <span className="predict-range-label">95% range</span>
              <span className="predict-range-values">
                {formatPrice(prediction.price_low)} – {formatPrice(prediction.price_high)}
              </span>
            </div>
            <div className="predict-meta">
              By {new Date(prediction.horizon_end_time).toLocaleString()} · {prediction.data_points} points
            </div>
          </div>
        )}
      </div>

      <div className="ml-controls">
        <label className="ml-label">Asset</label>
        <select
          value={selectedCoin}
          onChange={(e) => setSelectedCoin(e.target.value)}
          className="time-range-select"
        >
          {COINS.map((id) => (
            <option key={id} value={id}>{id}</option>
          ))}
        </select>
        <div className="time-range-buttons">
          {timeRanges.map((r, i) => (
            <button
              key={i}
              className="time-range-button"
              onClick={() => fetchSignals(r.start, r.end)}
              disabled={loading}
            >
              {r.label}
            </button>
          ))}
        </div>
      </div>

      {error && <div className="error-message">{error}</div>}
      {loading && <div className="loading">Loading signals…</div>}

      {!loading && (trend || volatility || topMovers.length > 0) && (
        <div className="ml-grid">
          {trend && (
            <div className="card ml-card">
              <div className="card-header">
                <TrendingUp className="card-icon" />
                <h2>Trend Signal</h2>
              </div>
              <div className="ml-metrics">
                <div className="metric-card">
                  <span className={`metric-value trend-${trend.trend.toLowerCase() === 'uptrend' ? 'positive' : trend.trend.toLowerCase() === 'downtrend' ? 'negative' : 'neutral'}`} style={{ fontSize: '1.25rem' }}>
                    {trend.trend}
                  </span>
                  <span className="metric-label">Direction</span>
                </div>
                <div className="metric-card">
                  <span className="metric-value">{trend.slope.toFixed(6)}</span>
                  <span className="metric-label">Slope</span>
                </div>
              </div>
            </div>
          )}

          {volatility && (
            <div className="card ml-card">
              <div className="card-header">
                <Activity className="card-icon" />
                <h2>Volatility Index</h2>
              </div>
              <div className="ml-metrics">
                <div className="metric-card">
                  <span className="metric-value">{formatPrice(volatility.stddev_price)}</span>
                  <span className="metric-label">Std Dev</span>
                </div>
                <div className="metric-card">
                  <span className="metric-value">{formatPrice(volatility.mean_price)}</span>
                  <span className="metric-label">Mean</span>
                </div>
              </div>
            </div>
          )}

          <div className="card ml-card ml-card-wide">
            <div className="card-header">
              <Zap className="card-icon" />
              <h2>Momentum (Top Movers)</h2>
            </div>
            <div className="top-movers-list">
              {topMovers.slice(0, 8).map((m, i) => (
                <div key={i} className="mover-item">
                  <span className="mover-coin">{m.coin_id}</span>
                  <span className={`mover-change ${m.percent_change >= 0 ? 'positive' : 'negative'}`}>
                    {m.percent_change >= 0 ? '+' : ''}{m.percent_change.toFixed(2)}%
                  </span>
                  <span className="mover-prices">{formatPrice(m.start_price)} → {formatPrice(m.end_price)}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      <div className="card ml-card ml-ai-section">
        <div className="card-header">
          <MessageSquare className="card-icon" />
          <h2>AI Query</h2>
        </div>
        <form onSubmit={handleAskAI} className="chat-input-bar">
          <input
            type="text"
            className="chat-input"
            placeholder="e.g. What was Bitcoin price at 2024-01-15 12:00?"
            value={aiQuestion}
            onChange={(e) => setAiQuestion(e.target.value)}
          />
          <button type="submit" className="send-button" disabled={aiLoading}>
            {aiLoading ? '…' : 'Ask'}
          </button>
        </form>
        {aiError && <div className="error-message">{aiError}</div>}
        {aiResults && (
          <div className="chat-results">
            <div className="results-header">
              <span>Coin</span>
              <span>Timestamp</span>
              <span className="price">Price</span>
            </div>
            {aiResults.map((row, i) => (
              <div key={i} className="results-row">
                <span>{row.coin_id}</span>
                <span>{new Date(row.timestamp).toLocaleString()}</span>
                <span className="price">{formatPrice(row.price_usd)}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
