import { TrendingUp, TrendingDown, DollarSign } from 'lucide-react'
import type { PriceData } from '../types'
import { formatPrice, formatTime, calculatePriceChange } from '../services/utils'

interface PriceCardProps {
  coinName: string
  latestPrice: PriceData | null
  priceHistory: PriceData[]
}

export const PriceCard = ({ coinName, latestPrice, priceHistory }: PriceCardProps) => {
  const priceChange = calculatePriceChange(priceHistory)

  return (
    <div className="card price-card">
      <div className="card-header">
        <h2>{coinName} Price</h2>
        <div className="price-change">
          {priceChange >= 0 ? (
            <TrendingUp className="trend-icon positive" />
          ) : (
            <TrendingDown className="trend-icon negative" />
          )}
          <span className={`change-value ${priceChange >= 0 ? 'positive' : 'negative'}`}>
            {priceChange >= 0 ? '+' : ''}{priceChange.toFixed(2)}%
          </span>
        </div>
      </div>
      <div className="current-price">
        <DollarSign className="dollar-icon" />
        <span className="price-value">
          {latestPrice ? formatPrice(latestPrice.price_usd) : 'Loading...'}
        </span>
      </div>
      <div className="price-timestamp">
        Last updated: {latestPrice ? formatTime(latestPrice.timestamp) : 'N/A'}
      </div>
    </div>
  )
} 