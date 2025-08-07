import { useState, useEffect } from 'react'
import { Coins, RefreshCw } from 'lucide-react'
import { ApiService } from '../services/api'

interface CoinManagerProps {
  onCoinSelect: (coinId: string) => void
}

export const CoinManager = ({ onCoinSelect }: CoinManagerProps) => {
  const [availableCoins, setAvailableCoins] = useState<string[]>([])
  const [loading, setLoading] = useState<boolean>(false)
  const [error, setError] = useState<string>('')

  const fetchAvailableCoins = async () => {
    setLoading(true)
    setError('')
    try {
      const coins = await ApiService.getAvailableCoins()
      setAvailableCoins(coins)
    } catch (err) {
      setError('Failed to fetch available coins')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchAvailableCoins()
  }, [])

  return (
    <div className="coin-manager">
      <div className="manager-header">
        <Coins className="manager-icon" />
        <h3>Available Coins</h3>
        <button 
          onClick={fetchAvailableCoins}
          disabled={loading}
          className="refresh-button"
          title="Refresh available coins"
        >
          <RefreshCw className={`refresh-icon ${loading ? 'spinning' : ''}`} />
        </button>
      </div>

      {error && (
        <div className="error-message">
          {error}
        </div>
      )}

      <div className="available-coins">
        {loading ? (
          <div className="loading">Loading available coins...</div>
        ) : availableCoins.length > 0 ? (
          <div className="coins-grid">
            {availableCoins.map((coinId) => (
              <button
                key={coinId}
                onClick={() => onCoinSelect(coinId)}
                className="available-coin-button"
              >
                {coinId.toUpperCase()}
              </button>
            ))}
          </div>
        ) : (
          <div className="no-coins">No coins available</div>
        )}
      </div>

      <div className="coin-count">
        Total: {availableCoins.length} coins available
      </div>
    </div>
  )
} 