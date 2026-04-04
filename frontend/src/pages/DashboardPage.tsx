import { useState, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useI18n } from '../i18n/context'
import { useGroups, useGroupMetrics, useGroupTrends } from '../api/hooks'
import { MetricsCard } from '../components/MetricsCard'
import { PRTable } from '../components/PRTable'
import { TrendCharts } from '../charts/TrendCharts'
import { SyncStatus } from '../components/SyncStatus'
import { StatusBar } from '../components/StatusBar'

type TrendPreset = '3months' | '6months' | '1year'

export function DashboardPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const { data: groups } = useGroups()
  const [selectedGroupId, setSelectedGroupId] = useState<number | null>(null)
  const [jobId, setJobId] = useState<number | null>(null)
  const [trendPreset, setTrendPreset] = useState<TrendPreset>('6months')

  const groupId = selectedGroupId ?? groups?.[0]?.id ?? null

  const { data: metrics, refetch: refetchMetrics } = useGroupMetrics(groupId)

  const trendParams = useMemo(() => {
    const now = new Date()
    const until = now.toISOString().slice(0, 10)
    let since: string
    switch (trendPreset) {
      case '3months':
        since = new Date(now.getFullYear(), now.getMonth() - 3, now.getDate()).toISOString().slice(0, 10)
        break
      case '1year':
        since = new Date(now.getFullYear() - 1, now.getMonth(), now.getDate()).toISOString().slice(0, 10)
        break
      default: // 6months
        since = new Date(now.getFullYear(), now.getMonth() - 6, now.getDate()).toISOString().slice(0, 10)
    }
    return { since, until, unit: metrics?.aggregation_unit }
  }, [trendPreset, metrics?.aggregation_unit])

  const { data: trends, refetch: refetchTrends } = useGroupTrends(groupId, trendParams)

  const handleSyncComplete = useCallback(() => {
    refetchMetrics()
    refetchTrends()
  }, [refetchMetrics, refetchTrends])

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

  const prev = metrics?.previous_period

  return (
    <div className="max-w-6xl mx-auto py-6 px-4 space-y-6">
      {/* Top bar */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <select
            value={groupId ?? ''}
            onChange={e => setSelectedGroupId(Number(e.target.value))}
            className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800"
          >
            {groups.map(g => (
              <option key={g.id} value={g.id}>{g.name}</option>
            ))}
          </select>
          <button
            onClick={() => navigate('/setup')}
            className="px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
            title={t('settings.new_group')}
          >
            + {t('settings.new_group')}
          </button>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate(`/settings${groupId ? `?groupId=${groupId}` : ''}`)}
            className="px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
          >
            {t('settings.title')}
          </button>
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
          lastSyncAt={metrics?.last_sync_at}
        />
      )}

      {/* Status bar */}
      {metrics && <StatusBar metrics={metrics} />}

      {/* Metrics cards */}
      {metrics && (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <MetricsCard
              title={t('dashboard.lead_time')}
              description={t('dashboard.lead_time_desc')}
              {...formatValue(metrics.lead_time_hours)}
              level={metrics.lead_time_level}
              previousValue={prev?.lead_time_hours}
              currentValue={metrics.lead_time_hours}
              invertComparison={true}
            />
            <MetricsCard
              title={t('dashboard.deploy_freq')}
              description={t('dashboard.deploy_freq_desc')}
              value={String(Math.round(metrics.deploy_frequency))}
              unit={t('dashboard.deploys_unit')}
              level={metrics.deploy_frequency_level}
              previousValue={prev?.deploy_frequency}
              currentValue={metrics.deploy_frequency}
              invertComparison={false}
            />
            <MetricsCard
              title={t('dashboard.cfr')}
              description={t('dashboard.cfr_desc')}
              value={metrics.change_failure_rate.toFixed(1)}
              unit="%"
              level={metrics.change_failure_rate_level}
              previousValue={prev?.change_failure_rate}
              currentValue={metrics.change_failure_rate}
              invertComparison={true}
            />
            <MetricsCard
              title={t('dashboard.mttr')}
              description={t('dashboard.mttr_desc')}
              value={metrics.mttr_hours !== null ? formatValue(metrics.mttr_hours).value : t('dashboard.na')}
              unit={metrics.mttr_hours !== null ? formatValue(metrics.mttr_hours).unit : ''}
              level={metrics.mttr_level ?? 'low'}
              previousValue={prev?.mttr_hours}
              currentValue={metrics.mttr_hours}
              invertComparison={true}
            />
          </div>

          {/* Trend Charts */}
          {trends && trends.data_points && trends.data_points.length > 0 && (
            <TrendCharts
              dataPoints={trends.data_points}
              unit={trends.unit}
              trendPreset={trendPreset}
              onPresetChange={setTrendPreset}
            />
          )}
        </>
      )}

      {/* PR Table */}
      {groupId && <PRTable groupId={groupId} />}
    </div>
  )
}
