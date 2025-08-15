import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import type { PriceData } from '../types'
import { formatPrice, formatTime } from '../services/utils'

interface ChartCardProps {
  priceHistory: PriceData[]
  timeRange: number
  loading: boolean
  coinColor: string
}

export const ChartCard = ({ priceHistory, timeRange, loading, coinColor }: ChartCardProps) => {
  const chartData = priceHistory.map(item => ({
    time: formatTime(item.timestamp),
    price: item.price_usd,
    timestamp: new Date(item.timestamp).getTime()
  })).sort((a, b) => a.timestamp - b.timestamp)

  let domainMin: number | 'auto' = 'auto'
  let domainMax: number | 'auto' = 'auto'

  if (chartData.length > 0) {
    const prices = chartData.map(d => d.price)
    const minPrice = Math.min(...prices)
    const maxPrice = Math.max(...prices)
    const padding = (maxPrice - minPrice) * 0.05 || minPrice * 0.001 
    domainMin = minPrice - padding
    domainMax = maxPrice + padding
  }

  return (
    <div className="card chart-card">
      <div className="card-header">
        <h2>Price History</h2>
        <span className="time-range">{timeRange} minutes</span>
      </div>
      <div className="chart-container">
        {loading ? (
          <div className="loading">Loading chart data...</div>
        ) : chartData.length > 0 ? (
          <ResponsiveContainer width="100%" height={300}>
            <AreaChart data={chartData}>
              <defs>
                <linearGradient id="priceGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={coinColor} stopOpacity={0.3}/>
                  <stop offset="95%" stopColor={coinColor} stopOpacity={0}/>
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
              <XAxis 
                dataKey="time" 
                stroke="#9ca3af"
                fontSize={12}
                tickLine={false}
              />
              <YAxis 
                stroke="#9ca3af"
                fontSize={12}
                tickLine={false}
                tickFormatter={(value) => `$${value.toFixed(2)}`}
                domain={[domainMin, domainMax]}
              />
              <Tooltip 
                contentStyle={{
                  backgroundColor: '#1f2937',
                  border: '1px solid #374151',
                  borderRadius: '8px',
                  color: '#f9fafb'
                }}
                formatter={(value: any) => [formatPrice(value), 'Price']}
                labelFormatter={(label) => `Time: ${label}`}
              />
              <Area
                type="monotone"
                dataKey="price"
                stroke={coinColor}
                strokeWidth={2}
                fill="url(#priceGradient)"
              />
            </AreaChart>
          </ResponsiveContainer>
        ) : (
          <div className="no-data">No price data available</div>
        )}
      </div>
    </div>
  )
} 