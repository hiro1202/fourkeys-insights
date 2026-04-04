import { useI18n } from '../i18n/context'
import type { ExtendedMetricsResult } from '../api/hooks'

interface StatusBarProps {
  metrics: ExtendedMetricsResult
}

export function StatusBar({ metrics }: StatusBarProps) {
  const { t } = useI18n()

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString()
  }

  const formatDateTime = (dateStr: string) => {
    const d = new Date(dateStr)
    return `${d.toLocaleDateString()} ${d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`
  }

  const unitLabel = metrics.aggregation_unit === 'monthly'
    ? t('settings.aggregation_monthly')
    : t('settings.aggregation_weekly')

  return (
    <div className="flex flex-wrap items-center gap-4 text-xs text-gray-500 dark:text-gray-400 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 px-4 py-2">
      <span>
        {t('status.period')}: {formatDate(metrics.period_start)} ~ {formatDate(metrics.period_end)} ({unitLabel})
      </span>
      <span className="text-gray-300 dark:text-gray-600">|</span>
      <span>
        {t('status.total_prs')}: {metrics.total_prs}
      </span>
      <span className="text-gray-300 dark:text-gray-600">|</span>
      <span>
        {t('status.lead_time_start')}: {t(`settings.lead_time_${metrics.lead_time_start.replace('.', '_').replace('_at', '')}`)}
      </span>
      {metrics.last_sync_at && (
        <>
          <span className="text-gray-300 dark:text-gray-600">|</span>
          <span>
            {t('status.last_sync')}: {formatDateTime(metrics.last_sync_at)}
          </span>
        </>
      )}
      {metrics.fallback_count > 0 && (
        <>
          <span className="text-gray-300 dark:text-gray-600">|</span>
          <span className="text-amber-600 dark:text-amber-400">
            {t('dashboard.fallback_warning', { count: metrics.fallback_count })}
          </span>
        </>
      )}
    </div>
  )
}
