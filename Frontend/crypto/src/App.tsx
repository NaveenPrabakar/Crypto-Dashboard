import { useState, useEffect } from 'react'
import {
  Header,
  CoinSelector,
  PriceCard,
  ChartCard,
  StatsCard,
  CoinManager,
  EmailSignup,
} from './components'
import { AnalyticsPage, MLInsightsPage, TopMoversPage } from './pages'
import { Routes, Route } from 'react-router-dom'
import { ApiService } from './services/api'
import type { PriceData, CoinInfo } from './types'
import './App.css'

const COINS: CoinInfo[] = [
  { id: 'bitcoin', name: 'Bitcoin', symbol: 'BTC', color: '#f7931a' },
  { id: 'ethereum', name: 'Ethereum', symbol: 'ETH', color: '#627eea' },
  { id: 'cardano', name: 'Cardano', symbol: 'ADA', color: '#0033ad' },
  { id: 'solana', name: 'Solana', symbol: 'SOL', color: '#14f195' },
  { id: 'polkadot', name: 'Polkadot', symbol: 'DOT', color: '#e6007a' },
  { id: 'chainlink', name: 'Chainlink', symbol: 'LINK', color: '#2a5ada' },
]

function App() {
  const [selectedCoin, setSelectedCoin] = useState<string>('bitcoin')
  const [latestPrice, setLatestPrice] = useState<PriceData | null>(null)
  const [priceHistory, setPriceHistory] = useState<PriceData[]>([])
  const [timeRange, setTimeRange] = useState<number>(60)
  const [loading, setLoading] = useState<boolean>(false)
  const [error, setError] = useState<string>('')
  const [showCoinManager, setShowCoinManager] = useState<boolean>(false)

  const fetchLatestPrice = async (coinId: string) => {
    try {
      const data = await ApiService.getLatestPrice(coinId)
      setLatestPrice(data)
    } catch (err) {
      setError('Failed to fetch latest price')
      console.error(err)
    }
  }

  const fetchPriceHistory = async (coinId: string, minutes: number) => {
    setLoading(true)
    try {
      const data = await ApiService.getPriceHistory(coinId, minutes)
      setPriceHistory(data)
      setError('')
    } catch (err) {
      setError('Failed to fetch price history')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleCoinSelect = (coinId: string) => {
    setSelectedCoin(coinId)
    setShowCoinManager(false)
  }

  const handleTimeRangeChange = (minutes: number) => {
    setTimeRange(minutes)
  }

  useEffect(() => {
    fetchLatestPrice(selectedCoin)
    fetchPriceHistory(selectedCoin, timeRange)
  }, [selectedCoin, timeRange])

  const selectedCoinInfo = COINS.find((coin) => coin.id === selectedCoin)

  return (
    <div className="app">
      <Header timeRange={timeRange} onTimeRangeChange={handleTimeRangeChange} />

      <main className="main">
        <Routes>
          <Route
            path="/"
            element={
              <>
                <div className="main-controls">
                  <CoinSelector
                    coins={COINS}
                    selectedCoin={selectedCoin}
                    onCoinSelect={handleCoinSelect}
                  />
                  <div className="control-buttons">
                    <button
                      onClick={() => setShowCoinManager(!showCoinManager)}
                      className={`control-button ${showCoinManager ? 'active' : ''}`}
                    >
                      {showCoinManager ? 'Hide' : 'Show'} Coins
                    </button>
                  </div>
                </div>

                {error && <div className="error-message">{error}</div>}

                {showCoinManager && (
                  <div className="coin-manager-section">
                    <CoinManager onCoinSelect={handleCoinSelect} />
                  </div>
                )}

                <div className="dashboard-grid">
                  <PriceCard
                    coinName={selectedCoinInfo?.name || selectedCoin}
                    latestPrice={latestPrice}
                    priceHistory={priceHistory}
                  />
                  <ChartCard
                    priceHistory={priceHistory}
                    timeRange={timeRange}
                    loading={loading}
                    coinColor={selectedCoinInfo?.color || '#d4af37'}
                  />
                  <StatsCard priceHistory={priceHistory} />
                </div>
              </>
            }
          />
          <Route path="/analytics" element={<AnalyticsPage />} />
          <Route path="/ml" element={<MLInsightsPage />} />
          <Route path="/top-movers" element={<TopMoversPage />} />
          <Route path="/subscribe" element={<EmailSignup />} />
        </Routes>
      </main>
    </div>
  )
}

export default App
