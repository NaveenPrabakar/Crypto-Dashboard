import { useState } from 'react'
import { CoinSelector, AdvancedAnalytics, CoinManager } from '../components'
import type { CoinInfo } from '../types'

const COINS: CoinInfo[] = [
  { id: 'bitcoin', name: 'Bitcoin', symbol: 'BTC', color: '#f7931a' },
  { id: 'ethereum', name: 'Ethereum', symbol: 'ETH', color: '#627eea' },
  { id: 'cardano', name: 'Cardano', symbol: 'ADA', color: '#0033ad' },
  { id: 'solana', name: 'Solana', symbol: 'SOL', color: '#14f195' },
  { id: 'polkadot', name: 'Polkadot', symbol: 'DOT', color: '#e6007a' },
  { id: 'chainlink', name: 'Chainlink', symbol: 'LINK', color: '#2a5ada' },
]

export const AnalyticsPage = () => {
  const [selectedCoin, setSelectedCoin] = useState<string>('bitcoin')
  const [showCoinManager, setShowCoinManager] = useState(false)

  return (
    <div className="analytics-page">
      <div className="page-header">
        <h1 className="page-title">Analytics</h1>
        <p className="page-subtitle">Historical averages, ranges, volatility, trends & price lookups</p>
      </div>
      <div className="main-controls">
        <CoinSelector
          coins={COINS}
          selectedCoin={selectedCoin}
          onCoinSelect={(id) => {
            setSelectedCoin(id)
            setShowCoinManager(false)
          }}
        />
        <button
          onClick={() => setShowCoinManager(!showCoinManager)}
          className={`control-button ${showCoinManager ? 'active' : ''}`}
        >
          {showCoinManager ? 'Hide' : 'Show'} Coins
        </button>
      </div>
      {showCoinManager && (
        <div className="coin-manager-section">
          <CoinManager onCoinSelect={(id) => setSelectedCoin(id)} />
        </div>
      )}
      <div className="advanced-analytics-section">
        <AdvancedAnalytics selectedCoin={selectedCoin} />
      </div>
    </div>
  )
}
