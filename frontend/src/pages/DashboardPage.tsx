import { useState, useCallback } from 'react'
import { useI18n } from '../i18n/context'
import { useGroups, useGroupMetrics, useGroupPulls } from '../api/hooks'
import { MetricsCard } from '../components/MetricsCard'
import { PRTable } from '../components/PRTable'
import { PRSizeChart } from '../charts/PRSizeChart'
import { SyncStatus } from '../components/SyncStatus'

export function DashboardPage() {
  const { t } = useI18n()
  const { data: groups } = useGroups()
  const [selectedGroupId, setSelectedGroupId] = useState<number | null>(null)
  const [jobId, setJobId] = useState<number | null>(null)

  // Auto-select first group
  const groupId = selectedGroupId ?? groups?.[0]?.id ?? null

  const { data: metrics, refetch: refetchMetrics } = useGroupMetrics(groupId)
  const { data: pullsData } = useGroupPulls(groupId, 1)

  const handleSyncComplete = useCallback(() => {
    refetchMetrics()
  }, [refetchMetrics])

  const handleExport = () => {
    if (groupId) {
      window.open(`/api/v1/groups/${groupId}/export`, '_blank')
    }
  }

  if (!groups || groups.length === 0) {
    return (
      <div className="max-w-4xl mx-auto py-12 px-4 text-center">
        <p className="text-gray-500">{t('dashboard.no_data')}</p>
      </div>
    )
  }

  const formatValue = (hours: number): { value: string; unit: string } => {
    if (hours < 24) return { value: hours.toFixed(1), unit: t('dashboard.hours') }
    return { value: (hours / 24).toFixed(1), unit: t('dashboard.days') }
  }

  return (
    <div className="max-w-6xl mx-auto py-6 px-4 space-y-6">
      {/* Top bar */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <select
            value={groupId ?? ''}
            onChange={e => setSelectedGroupId(Number(e.target.value))}
            className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800"
          >
            {groups.map(g => (
              <option key={g.id} value={g.id}>{g.name}</option>
            ))}
          </select>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={handleExport}
            className="px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
          >
            {t('dashboard.export')}
          </button>
        </div>
      </div>

      {/* Sync status */}
      {groupId && (
        <SyncStatus
          groupId={groupId}
          jobId={jobId}
          onJobStarted={setJobId}
          onComplete={handleSyncComplete}
        />
      )}

      {/* Metrics cards */}
      {metrics && (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <MetricsCard
              title={t('dashboard.lead_time')}
              {...formatValue(metrics.lead_time_hours)}
              level={metrics.lead_time_level}
              note={metrics.fallback_count > 0 ? t('dashboard.fallback_warning', { count: metrics.fallback_count }) : undefined}
            />
            <MetricsCard
              title={t('dashboard.deploy_freq')}
              value={metrics.deploy_frequency.toFixed(2)}
              unit={t('dashboard.per_day')}
              level={metrics.deploy_frequency_level}
              note={t('dashboard.treats_merge_as_deploy')}
            />
            <MetricsCard
              title={t('dashboard.cfr')}
              value={metrics.change_failure_rate.toFixed(1)}
              unit="%"
              level={metrics.change_failure_rate_level}
            />
            <MetricsCard
              title={t('dashboard.mttr')}
              value={metrics.mttr_hours !== null ? formatValue(metrics.mttr_hours).value : t('dashboard.na')}
              unit={metrics.mttr_hours !== null ? formatValue(metrics.mttr_hours).unit : ''}
              level={metrics.mttr_level ?? 'low'}
              note={t('dashboard.mttr_proxy')}
            />
          </div>

          {/* PR Size Distribution */}
          {pullsData && pullsData.pulls.length > 0 && (
            <PRSizeChart pulls={pullsData.pulls} />
          )}
        </>
      )}

      {/* PR Table */}
      {groupId && <PRTable groupId={groupId} />}
    </div>
  )
}
