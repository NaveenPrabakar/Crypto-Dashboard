import { useState, useEffect } from 'react'
import { TrendingUp, TrendingDown, DollarSign, Clock, Activity } from 'lucide-react'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, AreaChart, Area } from 'recharts'
import './App.css'

interface PriceData {
  coin_id: string
  timestamp: string
  price_usd: number
}

interface CoinInfo {
  id: string
  name: string
  symbol: string
  color: string
}

const COINS: CoinInfo[] = [
  { id: 'bitcoin', name: 'Bitcoin', symbol: 'BTC', color: '#f7931a' },
  { id: 'ethereum', name: 'Ethereum', symbol: 'ETH', color: '#627eea' },
  { id: 'cardano', name: 'Cardano', symbol: 'ADA', color: '#0033ad' },
  { id: 'solana', name: 'Solana', symbol: 'SOL', color: '#14f195' },
  { id: 'polkadot', name: 'Polkadot', symbol: 'DOT', color: '#e6007a' },
  { id: 'chainlink', name: 'Chainlink', symbol: 'LINK', color: '#2a5ada' }
]

function App() {
  const [selectedCoin, setSelectedCoin] = useState<string>('bitcoin')
  const [latestPrice, setLatestPrice] = useState<PriceData | null>(null)
  const [priceHistory, setPriceHistory] = useState<PriceData[]>([])
  const [timeRange, setTimeRange] = useState<number>(60)
  const [loading, setLoading] = useState<boolean>(false)
  const [error, setError] = useState<string>('')

  const fetchLatestPrice = async (coinId: string) => {
    try {
      const response = await fetch(`http://localhost:8000/latest/${coinId}`)
      if (!response.ok) throw new Error('Failed to fetch latest price')
      const data = await response.json()
      setLatestPrice(data)
    } catch (err) {
      setError('Failed to fetch latest price')
      console.error(err)
    }
  }

  const fetchPriceHistory = async (coinId: string, minutes: number) => {
    setLoading(true)
    try {
      const response = await fetch(`http://localhost:8000/history/${coinId}?minutes=${minutes}`)
      if (!response.ok) throw new Error('Failed to fetch price history')
      const data = await response.json()
      setPriceHistory(data)
      setError('')
    } catch (err) {
      setError('Failed to fetch price history')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchLatestPrice(selectedCoin)
    fetchPriceHistory(selectedCoin, timeRange)
  }, [selectedCoin, timeRange])

  const selectedCoinInfo = COINS.find(coin => coin.id === selectedCoin)
  const priceChange = priceHistory.length >= 2 
    ? ((priceHistory[priceHistory.length - 1]?.price_usd || 0) - (priceHistory[0]?.price_usd || 0)) / (priceHistory[0]?.price_usd || 1) * 100
    : 0

  const formatPrice = (price: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
      maximumFractionDigits: 6
    }).format(price)
  }

  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleTimeString()
  }

  const chartData = priceHistory.map(item => ({
    time: formatTime(item.timestamp),
    price: item.price_usd,
    timestamp: new Date(item.timestamp).getTime()
  })).sort((a, b) => a.timestamp - b.timestamp)

  return (
    <div className="app">
      <header className="header">
        <div className="header-content">
          <div className="logo">
            <Activity className="logo-icon" />
            <h1>Crypto Dashboard</h1>
          </div>
          <div className="time-controls">
            <Clock className="time-icon" />
            <select 
              value={timeRange} 
              onChange={(e) => setTimeRange(Number(e.target.value))}
              className="time-select"
            >
              <option value={15}>15 minutes</option>
              <option value={30}>30 minutes</option>
              <option value={60}>1 hour</option>
              <option value={120}>2 hours</option>
              <option value={240}>4 hours</option>
            </select>
          </div>
        </div>
      </header>

      <main className="main">
        <div className="coin-selector">
          {COINS.map(coin => (
            <button
              key={coin.id}
              className={`coin-button ${selectedCoin === coin.id ? 'active' : ''}`}
              onClick={() => setSelectedCoin(coin.id)}
              style={{ '--coin-color': coin.color } as React.CSSProperties}
            >
              <div className="coin-icon" style={{ backgroundColor: coin.color }}></div>
              <span className="coin-symbol">{coin.symbol}</span>
            </button>
          ))}
        </div>

        {error && (
          <div className="error-message">
            {error}
          </div>
        )}

        <div className="dashboard-grid">
          {/* Price Card */}
          <div className="card price-card">
            <div className="card-header">
              <h2>{selectedCoinInfo?.name} Price</h2>
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

          {/* Chart Card */}
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
                        <stop offset="5%" stopColor={selectedCoinInfo?.color} stopOpacity={0.3}/>
                        <stop offset="95%" stopColor={selectedCoinInfo?.color} stopOpacity={0}/>
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
                      stroke={selectedCoinInfo?.color}
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

          {/* Stats Card */}
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
                  {priceHistory.length > 0 
                    ? formatPrice(Math.max(...priceHistory.map(p => p.price_usd)))
                    : 'N/A'
                  }
                </div>
              </div>
              <div className="stat-item">
                <div className="stat-label">Low</div>
                <div className="stat-value">
                  {priceHistory.length > 0 
                    ? formatPrice(Math.min(...priceHistory.map(p => p.price_usd)))
                    : 'N/A'
                  }
                </div>
              </div>
              <div className="stat-item">
                <div className="stat-label">Average</div>
                <div className="stat-value">
                  {priceHistory.length > 0 
                    ? formatPrice(priceHistory.reduce((sum, p) => sum + p.price_usd, 0) / priceHistory.length)
                    : 'N/A'
                  }
                </div>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}

export default App
