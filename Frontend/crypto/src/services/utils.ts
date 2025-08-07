export const formatPrice = (price: number): string => {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 6
  }).format(price)
}

export const formatTime = (timestamp: string): string => {
  return new Date(timestamp).toLocaleTimeString()
}

export const formatDate = (timestamp: string): string => {
  return new Date(timestamp).toLocaleDateString()
}

export const formatDateTime = (timestamp: string): string => {
  return new Date(timestamp).toLocaleString()
}

export const calculatePriceChange = (priceHistory: Array<{ price_usd: number }>): number => {
  if (priceHistory.length < 2) return 0
  const firstPrice = priceHistory[0].price_usd
  const lastPrice = priceHistory[priceHistory.length - 1].price_usd
  return ((lastPrice - firstPrice) / firstPrice) * 100
}

export const getTimeRanges = () => {
  const ranges = [
    { label: 'Last 15 minutes', minutes: 15 },
    { label: 'Last 30 minutes', minutes: 30 },
    { label: 'Last hour', minutes: 60 },
    { label: 'Last 2 hours', minutes: 120 },
    { label: 'Last 4 hours', minutes: 240 },
    { label: 'Last 24 hours', minutes: 1440 }
  ]
  return ranges
}

export const getCustomTimeRanges = () => {
  const now = new Date()
  const ranges = [
    { 
      label: 'Last hour', 
      start: new Date(now.getTime() - 60 * 60 * 1000).toISOString(),
      end: now.toISOString()
    },
    { 
      label: 'Last 24 hours', 
      start: new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString(),
      end: now.toISOString()
    },
    { 
      label: 'Last 7 days', 
      start: new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString(),
      end: now.toISOString()
    },
    { 
      label: 'Last 30 days', 
      start: new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000).toISOString(),
      end: now.toISOString()
    }
  ]
  return ranges
} 