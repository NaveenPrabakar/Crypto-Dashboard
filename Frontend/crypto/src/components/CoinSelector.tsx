import type { CoinInfo } from '../types'

interface CoinSelectorProps {
  coins: CoinInfo[]
  selectedCoin: string
  onCoinSelect: (coinId: string) => void
}

export const CoinSelector = ({ coins, selectedCoin, onCoinSelect }: CoinSelectorProps) => {
  return (
    <div className="coin-selector">
      {coins.map(coin => (
        <button
          key={coin.id}
          className={`coin-button ${selectedCoin === coin.id ? 'active' : ''}`}
          onClick={() => onCoinSelect(coin.id)}
          style={{ '--coin-color': coin.color } as React.CSSProperties}
        >
          <div className="coin-icon" style={{ backgroundColor: coin.color }}></div>
          <span className="coin-symbol">{coin.symbol}</span>
        </button>
      ))}
    </div>
  )
} 