import { useState, useEffect } from 'react'
import { ApiService } from '../services/api'
import type { 
  AveragePriceData, 
  PriceRangeData, 
  PriceData, 
  VolatilityData, 
  TrendData, 
  TopMoverData 
} from '../types'
import { getCustomTimeRanges } from '../services/utils'

interface AdvancedAnalyticsProps {
  selectedCoin: string
}

export const AdvancedAnalytics = ({ selectedCoin }: AdvancedAnalyticsProps) => {
  const [averageData, setAverageData] = useState<AveragePriceData | null>(null)
  const [rangeData, setRangeData] = useState<PriceRangeData | null>(null)
  const [priceAtTime, setPriceAtTime] = useState<PriceData | null>(null)
  const [volatilityData, setVolatilityData] = useState<VolatilityData | null>(null)
  const [trendData, setTrendData] = useState<TrendData | null>(null)
  const [topMovers, setTopMovers] = useState<TopMoverData[]>([])
  const [customTimestamp, setCustomTimestamp] = useState<string>('')
  const [selectedTimeRange, setSelectedTimeRange] = useState<string>('')
  const [loading, setLoading] = useState<boolean>(false)
  const [error, setError] = useState<string>('')

  // Chat assistant state
  type ChatRole = 'user' | 'assistant'
  interface ChatMessage {
    role: ChatRole
    text?: string
    results?: PriceData[]
  }
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [aiQuestion, setAiQuestion] = useState<string>('')
  const [aiLoading, setAiLoading] = useState<boolean>(false)

  const timeRanges = getCustomTimeRanges()

  const fetchAveragePrice = async (start: string, end: string) => {
    setLoading(true)
    setError('')
    try {
      const data = await ApiService.getAveragePrice(selectedCoin, start, end)
      setAverageData(data)
    } catch (err) {
      setError('Failed to fetch average price data')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const fetchPriceRange = async (start: string, end: string) => {
    setLoading(true)
    setError('')
    try {
      const data = await ApiService.getPriceRange(selectedCoin, start, end)
      setRangeData(data)
    } catch (err) {
      setError('Failed to fetch price range data')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const fetchPriceAtTime = async (timestamp: string | Date) => {
  setLoading(true)
  setError('')
  try {
    const isoString = typeof timestamp === 'string'
      ? new Date(timestamp).toISOString()
      : timestamp.toISOString();

    const data = await ApiService.getPriceAtTime(selectedCoin, isoString)
    setPriceAtTime(data)
  } catch (err) {
    setError('Failed to fetch price at specified time')
    console.error(err)
  } finally {
    setLoading(false)
  }
}


  const fetchVolatility = async (start: string, end: string) => {
    setLoading(true)
    setError('')
    try {
      const data = await ApiService.getVolatility(selectedCoin, start, end)
      setVolatilityData(data)
    } catch (err) {
      setError('Failed to fetch volatility data')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const fetchTrend = async (start: string, end: string) => {
    setLoading(true)
    setError('')
    try {
      const data = await ApiService.getTrend(selectedCoin, start, end)
      setTrendData(data)
    } catch (err) {
      setError('Failed to fetch trend data')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const fetchTopMovers = async () => {
    setLoading(true)
    setError('')
    try {
      const data = await ApiService.getTopMovers(60)
      setTopMovers(data)
    } catch (err) {
      setError('Failed to fetch top movers data')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleTimeRangeSelect = (range: { start: string; end: string }) => {
    setSelectedTimeRange(`${range.start} to ${range.end}`)
    fetchAveragePrice(range.start, range.end)
    fetchPriceRange(range.start, range.end)
    fetchVolatility(range.start, range.end)
    fetchTrend(range.start, range.end)
  }

  const handleCustomTimestampSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (customTimestamp) {
      fetchPriceAtTime(customTimestamp)
    }
  }

  const handleAiAskSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = aiQuestion.trim()
    if (!trimmed) return
    const userMessage: ChatMessage = { role: 'user', text: trimmed }
    setMessages((prev) => [...prev, userMessage])
    setAiLoading(true)
    try {
      const data = await ApiService.askAI(trimmed)
      const assistantMessage: ChatMessage = { role: 'assistant', results: data }
      setMessages((prev) => [...prev, assistantMessage])
      setAiQuestion('')
    } catch (err) {
      console.error(err)
      const errorMessage: ChatMessage = { role: 'assistant', text: 'Sorry, I could not process that request.' }
      setMessages((prev) => [...prev, errorMessage])
    } finally {
      setAiLoading(false)
    }
  }

  useEffect(() => {
    fetchTopMovers()
  }, [])

  const formatPrice = (price: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(price)
  }

  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleString()
  }

  const getTrendColor = (trend: string) => {
    switch (trend.toLowerCase()) {
      case 'uptrend':
        return 'positive'
      case 'downtrend':
        return 'negative'
      default:
        return 'neutral'
    }
  }

  return (
    <div className="advanced-analytics">
      <h2>Advanced Analytics</h2>
      
      {error && <div className="error-message">{error}</div>}
      
      <div className="analytics-grid">
        {/* Time Range Selector */}
        <div className="analytics-section">
          <h3>Time Range Analysis</h3>
          <div className="time-range-buttons">
            {timeRanges.map((range, index) => (
              <button
                key={index}
                onClick={() => handleTimeRangeSelect(range)}
                className="time-range-button"
              >
                {range.label}
              </button>
            ))}
          </div>
          {selectedTimeRange && (
            <p className="selected-range">Selected: {selectedTimeRange}</p>
          )}
        </div>

        {/* Custom Timestamp */}
        <div className="analytics-section">
          <h3>Price at Specific Time</h3>
          <form onSubmit={handleCustomTimestampSubmit} className="timestamp-form">
            <input
              type="datetime-local"
              value={customTimestamp}
              onChange={(e) => setCustomTimestamp(e.target.value)}
              className="timestamp-input"
            />
            <button type="submit" className="submit-button">
              Get Price
            </button>
          </form>
          {priceAtTime && (
            <div className="price-at-time">
              <p>Price: {formatPrice(priceAtTime.price_usd)}</p>
              <p>Time: {formatTime(priceAtTime.timestamp)}</p>
            </div>
          )}
        </div>

        {/* Average Price */}
        {averageData && (
          <div className="analytics-section">
            <h3>Average Price</h3>
            <div className="metric-card">
              <p className="metric-value">{formatPrice(averageData.average)}</p>
              <p className="metric-label">Average Price</p>
              <p className="metric-detail">Data points: {averageData.data_points}</p>
            </div>
          </div>
        )}

        {/* Price Range */}
        {rangeData && (
          <div className="analytics-section">
            <h3>Price Range</h3>
            <div className="range-metrics">
              <div className="metric-card">
                <p className="metric-value">{formatPrice(rangeData.min)}</p>
                <p className="metric-label">Minimum</p>
              </div>
              <div className="metric-card">
                <p className="metric-value">{formatPrice(rangeData.max)}</p>
                <p className="metric-label">Maximum</p>
              </div>
            </div>
          </div>
        )}

        {/* Volatility */}
        {volatilityData && (
          <div className="analytics-section">
            <h3>Volatility Analysis</h3>
            <div className="volatility-metrics">
              <div className="metric-card">
                <p className="metric-value">{formatPrice(volatilityData.stddev_price)}</p>
                <p className="metric-label">Standard Deviation</p>
              </div>
              <div className="metric-card">
                <p className="metric-value">{formatPrice(volatilityData.mean_price)}</p>
                <p className="metric-label">Mean Price</p>
              </div>
              <p className="metric-detail">Data points: {volatilityData.data_points}</p>
            </div>
          </div>
        )}

        {/* Trend */}
        {trendData && (
          <div className="analytics-section">
            <h3>Trend Analysis</h3>
            <div className="trend-metrics">
              <div className="metric-card">
                <p className={`metric-value trend-${getTrendColor(trendData.trend)}`}>
                  {trendData.trend}
                </p>
                <p className="metric-label">Trend Direction</p>
              </div>
              <div className="metric-card">
                <p className="metric-value">{trendData.slope.toFixed(6)}</p>
                <p className="metric-label">Slope</p>
              </div>
              <p className="metric-detail">Data points: {trendData.data_points}</p>
            </div>
          </div>
        )}

        {/* Top Movers */}
        <div className="analytics-section">
          <h3>Top Movers (Last Hour)</h3>
          <button onClick={fetchTopMovers} className="refresh-button">
            Refresh
          </button>
          <div className="top-movers-list">
            {topMovers.slice(0, 10).map((mover, index) => (
              <div key={index} className="mover-item">
                <span className="mover-coin">{mover.coin_id}</span>
                <span className={`mover-change ${mover.percent_change >= 0 ? 'positive' : 'negative'}`}>
                  {mover.percent_change >= 0 ? '+' : ''}{mover.percent_change.toFixed(2)}%
                </span>
                <span className="mover-prices">
                  {formatPrice(mover.start_price)} → {formatPrice(mover.end_price)}
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* AI Chat Assistant */}
        <div className="analytics-section chat-assistant">
          <h3>Assistant</h3>
          <div className="chat-container">
            <div className="chat-window">
              {messages.length === 0 && (
                <div className="chat-empty">
                  Ask about prices with natural language. Try: “Show {selectedCoin} prices for the last 15 minutes”.
                </div>
              )}
              {messages.map((m, idx) => (
                <div key={idx} className={`chat-message ${m.role}`}>
                  <div className={`chat-bubble ${m.role}`}>
                    {m.text && <p className="chat-text">{m.text}</p>}
                    {m.results && m.results.length > 0 && (
                      <div className="chat-results">
                        <div className="results-header">
                          <span>Coin</span>
                          <span>Time</span>
                          <span>Price</span>
                        </div>
                        {m.results.map((row, rIdx) => (
                          <div key={rIdx} className="results-row">
                            <span className="coin">{row.coin_id}</span>
                            <span className="time">{formatTime(row.timestamp)}</span>
                            <span className="price">{formatPrice(row.price_usd)}</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              ))}
              {aiLoading && (
                <div className="chat-message assistant">
                  <div className="chat-bubble assistant">
                    <div className="typing">
                      <span></span><span></span><span></span>
                    </div>
                  </div>
                </div>
              )}
            </div>
            <form onSubmit={handleAiAskSubmit} className="chat-input-bar">
              <input
                type="text"
                value={aiQuestion}
                onChange={(e) => setAiQuestion(e.target.value)}
                className="chat-input"
                placeholder={`Ask about ${selectedCoin} prices...`}
                disabled={aiLoading}
              />
              <button type="submit" className="send-button" disabled={aiLoading || !aiQuestion.trim()}>
                Send
              </button>
            </form>
          </div>
        </div>
      </div>

      {loading && <div className="loading">Loading analytics data...</div>}
    </div>
  )
} 