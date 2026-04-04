import { useI18n } from '../i18n/context'

const levelColors: Record<string, string> = {
  elite: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
  high: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
  medium: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
  low: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
}

interface MetricsCardProps {
  title: string
  value: string
  unit: string
  level: string
  note?: string
}

export function MetricsCard({ title, value, unit, level, note }: MetricsCardProps) {
  const { t } = useI18n()
  const colorClass = levelColors[level] || levelColors.low

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400">{title}</h3>
        <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${colorClass}`}>
          {t(`level.${level}`)}
        </span>
      </div>
      <div className="flex items-baseline gap-1">
        <span className="text-2xl font-bold">{value}</span>
        <span className="text-sm text-gray-500 dark:text-gray-400">{unit}</span>
      </div>
      {note && (
        <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">{note}</p>
      )}
    </div>
  )
}
