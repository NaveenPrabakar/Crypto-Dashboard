import { useState, useEffect } from 'react'
import { BarChart3, Calendar, Clock } from 'lucide-react'
import { ApiService } from '../services/api'
import type { AveragePriceData, PriceRangeData, PriceData } from '../types'
import { formatPrice, formatDateTime, getCustomTimeRanges } from '../services/utils'

interface AdvancedAnalyticsProps {
  selectedCoin: string
}

export const AdvancedAnalytics = ({ selectedCoin }: AdvancedAnalyticsProps) => {
  const [averageData, setAverageData] = useState<AveragePriceData | null>(null)
  const [rangeData, setRangeData] = useState<PriceRangeData | null>(null)
  const [priceAtTime, setPriceAtTime] = useState<PriceData | null>(null)
  const [customTimestamp, setCustomTimestamp] = useState<string>('')
  const [selectedTimeRange, setSelectedTimeRange] = useState<string>('')
  const [loading, setLoading] = useState<boolean>(false)
  const [error, setError] = useState<string>('')

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

  const fetchPriceAtTime = async (timestamp: string) => {
    setLoading(true)
    setError('')
    try {
      const data = await ApiService.getPriceAtTime(selectedCoin, timestamp)
      setPriceAtTime(data)
    } catch (err) {
      setError('Failed to fetch price at specified time')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleTimeRangeSelect = (range: { start: string; end: string }) => {
    setSelectedTimeRange(`${range.start} to ${range.end}`)
    fetchAveragePrice(range.start, range.end)
    fetchPriceRange(range.start, range.end)
  }

  const handleCustomTimestamp = () => {
    if (customTimestamp) {
      fetchPriceAtTime(customTimestamp)
    }
  }

  useEffect(() => {
    // Reset data when coin changes
    setAverageData(null)
    setRangeData(null)
    setPriceAtTime(null)
    setSelectedTimeRange('')
  }, [selectedCoin])

  return (
    <div className="advanced-analytics">
      <div className="analytics-header">
        <BarChart3 className="analytics-icon" />
        <h2>Advanced Analytics</h2>
      </div>

      {error && (
        <div className="error-message">
          {error}
        </div>
      )}

      <div className="analytics-grid">
        {/* Time Range Analysis */}
        <div className="analytics-card">
          <div className="card-header">
            <Calendar className="card-icon" />
            <h3>Time Range Analysis</h3>
          </div>
          <div className="time-range-selector">
            <select 
              value={selectedTimeRange} 
              onChange={(e) => {
                const range = timeRanges.find(r => `${r.start} to ${r.end}` === e.target.value)
                if (range) {
                  handleTimeRangeSelect(range)
                }
              }}
              className="time-range-select"
            >
              <option value="">Select time range...</option>
              {timeRanges.map((range, index) => (
                <option key={index} value={`${range.start} to ${range.end}`}>
                  {range.label}
                </option>
              ))}
            </select>
          </div>
          
          {averageData && (
            <div className="analytics-results">
              <div className="result-item">
                <span className="result-label">Average Price:</span>
                <span className="result-value">{formatPrice(averageData.average)}</span>
              </div>
              <div className="result-item">
                <span className="result-label">Data Points:</span>
                <span className="result-value">{averageData.data_points}</span>
              </div>
            </div>
          )}

          {rangeData && (
            <div className="analytics-results">
              <div className="result-item">
                <span className="result-label">Min Price:</span>
                <span className="result-value">{formatPrice(rangeData.min)}</span>
              </div>
              <div className="result-item">
                <span className="result-label">Max Price:</span>
                <span className="result-value">{formatPrice(rangeData.max)}</span>
              </div>
            </div>
          )}
        </div>

        {/* Price at Specific Time */}
        <div className="analytics-card">
          <div className="card-header">
            <Clock className="card-icon" />
            <h3>Price at Specific Time</h3>
          </div>
          <div className="timestamp-input">
            <input
              type="datetime-local"
              value={customTimestamp}
              onChange={(e) => setCustomTimestamp(e.target.value)}
              className="timestamp-field"
            />
            <button 
              onClick={handleCustomTimestamp}
              disabled={!customTimestamp || loading}
              className="fetch-button"
            >
              {loading ? 'Loading...' : 'Fetch Price'}
            </button>
          </div>
          
          {priceAtTime && (
            <div className="analytics-results">
              <div className="result-item">
                <span className="result-label">Price at {formatDateTime(priceAtTime.timestamp)}:</span>
                <span className="result-value">{formatPrice(priceAtTime.price_usd)}</span>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
} 