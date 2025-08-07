export interface PriceData {
  coin_id: string
  timestamp: string
  price_usd: number
}

export interface CoinInfo {
  id: string
  name: string
  symbol: string
  color: string
}

export interface AveragePriceData {
  coin_id: string
  average: number
  data_points: number
  start: string
  end: string
}

export interface PriceRangeData {
  coin_id: string
  min: number
  max: number
  start: string
  end: string
}

export interface TimeRange {
  start: string
  end: string
} 