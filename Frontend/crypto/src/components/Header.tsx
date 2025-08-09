import { Activity, Clock, Settings } from 'lucide-react'
import { Link } from 'react-router-dom'

interface HeaderProps {
  timeRange: number
  onTimeRangeChange: (minutes: number) => void
}

export const Header = ({ timeRange, onTimeRangeChange }: HeaderProps) => {
  return (
    <header className="header">
      <div className="header-content">
        <div className="logo">
          <Activity className="logo-icon" />
          <h1>Crypto Dashboard</h1>
        </div>
        <div className="header-right">
          <div className="time-controls">
            <Clock className="time-icon" />
            <select 
              value={timeRange} 
              onChange={(e) => onTimeRangeChange(Number(e.target.value))}
              className="time-select"
            >
              <option value={15}>15 minutes</option>
              <option value={30}>30 minutes</option>
              <option value={60}>1 hour</option>
              <option value={120}>2 hours</option>
              <option value={240}>4 hours</option>
              <option value={1440}>24 hours</option>
            </select>
          </div>
          <Link to="/subscribe" className="settings-link" aria-label="Settings">
            <Settings className="settings-icon" />
          </Link>
        </div>
      </div>
    </header>
  )
} 