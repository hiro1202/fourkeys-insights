import { useState } from 'react'
import { useI18n } from '../i18n/context'

const levelColors: Record<string, string> = {
  elite: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
  high: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
  medium: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
  low: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
}

interface MetricsCardProps {
  title: string
  description?: string
  value: string
  unit: string
  level: string
  note?: string
  previousValue?: number | null
  currentValue?: number | null
  invertComparison?: boolean
  badge?: string
}

export function MetricsCard({ title, description, value, unit, level, note, previousValue, currentValue, invertComparison = false, badge }: MetricsCardProps) {
  const { t } = useI18n()
  const [showTooltip, setShowTooltip] = useState(false)
  const colorClass = levelColors[level] || levelColors.low

  let changeIndicator: { text: string; color: string } | null = null
  if (previousValue != null && currentValue != null && previousValue !== 0) {
    const pctChange = ((currentValue - previousValue) / Math.abs(previousValue)) * 100
    if (Math.abs(pctChange) >= 0.1) {
      const isImprovement = invertComparison ? pctChange < 0 : pctChange > 0
      const arrow = pctChange > 0 ? '\u2191' : '\u2193'
      changeIndicator = {
        text: `${arrow} ${Math.abs(pctChange).toFixed(1)}%`,
        color: isImprovement
          ? 'text-green-600 dark:text-green-400'
          : 'text-red-600 dark:text-red-400',
      }
    }
  }

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-1">
          <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400">{title}</h3>
          {description && (
            <div className="relative">
              <button
                className="text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300"
                onMouseEnter={() => setShowTooltip(true)}
                onMouseLeave={() => setShowTooltip(false)}
                onClick={() => setShowTooltip(!showTooltip)}
              >
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" className="w-4 h-4">
                  <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM8.94 6.94a.75.75 0 11-1.061-1.061 3 3 0 112.871 5.026v.345a.75.75 0 01-1.5 0v-.5c0-.72.57-1.172 1.081-1.287A1.5 1.5 0 108.94 6.94zM10 15a1 1 0 100-2 1 1 0 000 2z" clipRule="evenodd" />
                </svg>
              </button>
              {showTooltip && (
                <div className="absolute z-10 left-0 top-5 w-56 p-2 text-xs bg-gray-900 dark:bg-gray-700 text-white rounded shadow-lg">
                  {description}
                </div>
              )}
            </div>
          )}
        </div>
        <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${colorClass}`}>
          {t(`level.${level}`)}
        </span>
      </div>
      <div className="flex items-baseline gap-1">
        <span className="text-2xl font-bold">{value}</span>
        <span className="text-sm text-gray-500 dark:text-gray-400">{unit}</span>
        {changeIndicator && (
          <span className={`text-xs ml-2 font-medium ${changeIndicator.color}`}>
            {changeIndicator.text} {t('dashboard.vs_prev')}
          </span>
        )}
      </div>
      {badge && (
        <span className="inline-block mt-1 text-xs px-2 py-0.5 rounded bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200">
          {badge}
        </span>
      )}
      {note && (
        <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">{note}</p>
      )}
    </div>
  )
}
