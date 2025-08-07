import type { PriceData } from '../types'
import { formatPrice } from '../services/utils'

interface StatsCardProps {
  priceHistory: PriceData[]
}

export const StatsCard = ({ priceHistory }: StatsCardProps) => {
  const high = priceHistory.length > 0 ? Math.max(...priceHistory.map(p => p.price_usd)) : 0
  const low = priceHistory.length > 0 ? Math.min(...priceHistory.map(p => p.price_usd)) : 0
  const average = priceHistory.length > 0 
    ? priceHistory.reduce((sum, p) => sum + p.price_usd, 0) / priceHistory.length 
    : 0

  return (
    <div className="card stats-card">
      <div className="card-header">
        <h2>Statistics</h2>
      </div>
      <div className="stats-grid">
        <div className="stat-item">
          <div className="stat-label">Data Points</div>
          <div className="stat-value">{priceHistory.length}</div>
        </div>
        <div className="stat-item">
          <div className="stat-label">High</div>
          <div className="stat-value">
            {priceHistory.length > 0 ? formatPrice(high) : 'N/A'}
          </div>
        </div>
        <div className="stat-item">
          <div className="stat-label">Low</div>
          <div className="stat-value">
            {priceHistory.length > 0 ? formatPrice(low) : 'N/A'}
          </div>
        </div>
        <div className="stat-item">
          <div className="stat-label">Average</div>
          <div className="stat-value">
            {priceHistory.length > 0 ? formatPrice(average) : 'N/A'}
          </div>
        </div>
      </div>
    </div>
  )
} 