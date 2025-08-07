import type { PriceData, AveragePriceData, PriceRangeData } from '../types'

const API_BASE_URL = 'http://localhost:8000'

export class ApiService {
  static async getLatestPrice(coinId: string): Promise<PriceData> {
    const response = await fetch(`${API_BASE_URL}/latest/${coinId}`)
    if (!response.ok) {
      throw new Error('Failed to fetch latest price')
    }
    return response.json()
  }

  static async getPriceHistory(coinId: string, minutes: number): Promise<PriceData[]> {
    const response = await fetch(`${API_BASE_URL}/history/${coinId}?minutes=${minutes}`)
    if (!response.ok) {
      throw new Error('Failed to fetch price history')
    }
    return response.json()
  }

  static async getAveragePrice(coinId: string, start: string, end: string): Promise<AveragePriceData> {
    const response = await fetch(`${API_BASE_URL}/average/${coinId}?start=${start}&end=${end}`)
    if (!response.ok) {
      throw new Error('Failed to fetch average price')
    }
    return response.json()
  }

  static async getPriceAtTime(coinId: string, timestamp: string): Promise<PriceData> {
    const response = await fetch(`${API_BASE_URL}/at/${coinId}?timestamp=${timestamp}`)
    if (!response.ok) {
      throw new Error('Failed to fetch price at time')
    }
    return response.json()
  }

  static async getPriceRange(coinId: string, start: string, end: string): Promise<PriceRangeData> {
    const response = await fetch(`${API_BASE_URL}/range/${coinId}?start=${start}&end=${end}`)
    if (!response.ok) {
      throw new Error('Failed to fetch price range')
    }
    return response.json()
  }

  static async getAvailableCoins(): Promise<string[]> {
    const response = await fetch(`${API_BASE_URL}/coins`)
    if (!response.ok) {
      throw new Error('Failed to fetch available coins')
    }
    return response.json()
  }

  static async getVolatility(coinId: string, start: string, end: string): Promise<VolatilityData> {
    const response = await fetch(`${API_BASE_URL}/volatility/${coinId}?start=${start}&end=${end}`)
    if (!response.ok) {
      throw new Error('Failed to fetch volatility data')
    }
    return response.json()
  }

  static async getTrend(coinId: string, start: string, end: string): Promise<TrendData> {
    const response = await fetch(`${API_BASE_URL}/trend/${coinId}?start=${start}&end=${end}`)
    if (!response.ok) {
      throw new Error('Failed to fetch trend data')
    }
    return response.json()
  }

  static async getTopMovers(minutes: number = 60): Promise<TopMoverData[]> {
    const response = await fetch(`${API_BASE_URL}/top-movers?minutes=${minutes}`)
    if (!response.ok) {
      throw new Error('Failed to fetch top movers data')
    }
    return response.json()
  }
} 