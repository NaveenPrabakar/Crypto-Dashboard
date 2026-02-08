import { Activity, Clock, Settings } from 'lucide-react'
import { Link, useLocation } from 'react-router-dom'

interface HeaderProps {
  timeRange?: number
  onTimeRangeChange?: (minutes: number) => void
}

const NAV_LINKS = [
  { path: '/', label: 'Dashboard' },
  { path: '/analytics', label: 'Analytics' },
  { path: '/ml', label: 'ML Insights' },
  { path: '/top-movers', label: 'Top Movers' },
  { path: '/subscribe', label: 'Subscribe' },
]

export const Header = ({ timeRange = 60, onTimeRangeChange }: HeaderProps) => {
  const location = useLocation()
  const isDashboard = location.pathname === '/'

  return (
    <header className="header">
      <div className="header-content">
        <Link to="/" className="logo">
          <Activity className="logo-icon" />
          <h1>Crypto Dashboard</h1>
        </Link>

        <nav className="nav-links">
          {NAV_LINKS.map(({ path, label }) => (
            <Link
              key={path}
              to={path}
              className={`nav-link ${location.pathname === path ? 'active' : ''}`}
            >
              {label}
            </Link>
          ))}
        </nav>

        <div className="header-right">
          {isDashboard && onTimeRangeChange && (
            <div className="time-controls">
              <Clock className="time-icon" />
              <select
                value={timeRange}
                onChange={(e) => onTimeRangeChange(Number(e.target.value))}
                className="time-select"
              >
                <option value={15}>15m</option>
                <option value={30}>30m</option>
                <option value={60}>1h</option>
                <option value={120}>2h</option>
                <option value={240}>4h</option>
                <option value={1440}>24h</option>
              </select>
            </div>
          )}
          <Link to="/subscribe" className="settings-link" aria-label="Subscribe">
            <Settings className="settings-icon" />
          </Link>
        </div>
      </div>
    </header>
  )
}
