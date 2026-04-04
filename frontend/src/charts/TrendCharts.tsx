import ReactECharts from 'echarts-for-react'
import { useI18n } from '../i18n/context'
import type { TrendDataPoint } from '../api/hooks'

type TrendPreset = '3months' | '6months' | '1year'

interface TrendChartsProps {
  dataPoints: TrendDataPoint[]
  unit: string
  trendPreset: TrendPreset
  onPresetChange: (preset: TrendPreset) => void
}

export function TrendCharts({ dataPoints, unit, trendPreset, onPresetChange }: TrendChartsProps) {
  const { t } = useI18n()

  if (!dataPoints || dataPoints.length === 0) return null

  const labels = dataPoints.map(p => {
    const d = new Date(p.period_start + 'T00:00:00')
    if (unit === 'monthly') {
      return `${d.getFullYear()}/${d.getMonth() + 1}`
    }
    return `${d.getMonth() + 1}/${d.getDate()}`
  })
  const gridStyle = { left: '12%', right: '5%', top: '15%', bottom: '20%' }
  const axisLabel = { fontSize: 10 }
  const tooltipStyle = { trigger: 'axis' as const }
  const height = 220

  const leadTimeOption = {
    tooltip: tooltipStyle,
    grid: gridStyle,
    title: { text: t('dashboard.lead_time'), textStyle: { fontSize: 13 }, left: 'center' },
    xAxis: { type: 'category' as const, data: labels, axisLabel },
    yAxis: { type: 'value' as const, name: t('dashboard.hours'), nameTextStyle: { fontSize: 10 } },
    series: [{
      data: dataPoints.map(p => p.lead_time_hours),
      type: 'line' as const,
      smooth: true,
      itemStyle: { color: '#3b82f6' },
      areaStyle: { color: 'rgba(59, 130, 246, 0.1)' },
    }],
  }

  const deployFreqOption = {
    tooltip: tooltipStyle,
    grid: gridStyle,
    title: { text: t('dashboard.deploy_freq'), textStyle: { fontSize: 13 }, left: 'center' },
    xAxis: { type: 'category' as const, data: labels, axisLabel },
    yAxis: { type: 'value' as const, name: t('dashboard.deploys_unit'), nameTextStyle: { fontSize: 10 } },
    series: [{
      data: dataPoints.map(p => p.deploy_frequency),
      type: 'line' as const,
      smooth: true,
      itemStyle: { color: '#22c55e' },
      areaStyle: { color: 'rgba(34, 197, 94, 0.1)' },
    }],
  }

  const cfrOption = {
    tooltip: tooltipStyle,
    grid: gridStyle,
    title: { text: t('dashboard.cfr'), textStyle: { fontSize: 13 }, left: 'center' },
    xAxis: { type: 'category' as const, data: labels, axisLabel },
    yAxis: { type: 'value' as const, name: '%', nameTextStyle: { fontSize: 10 } },
    series: [{
      data: dataPoints.map(p => p.change_failure_rate),
      type: 'line' as const,
      smooth: true,
      itemStyle: { color: '#eab308' },
      areaStyle: { color: 'rgba(234, 179, 8, 0.1)' },
    }],
  }

  const mttrOption = {
    tooltip: tooltipStyle,
    grid: gridStyle,
    title: { text: t('dashboard.mttr'), textStyle: { fontSize: 13 }, left: 'center' },
    xAxis: { type: 'category' as const, data: labels, axisLabel },
    yAxis: { type: 'value' as const, name: t('dashboard.hours'), nameTextStyle: { fontSize: 10 } },
    series: [{
      data: dataPoints.map(p => p.mttr_hours ?? 0),
      type: 'line' as const,
      smooth: true,
      itemStyle: { color: '#ef4444' },
      areaStyle: { color: 'rgba(239, 68, 68, 0.1)' },
    }],
  }

  const presets: { key: TrendPreset; label: string }[] = [
    { key: '3months', label: t('dashboard.trend_3months') },
    { key: '6months', label: t('dashboard.trend_6months') },
    { key: '1year', label: t('dashboard.trend_1year') },
  ]

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-medium">
          {t('dashboard.trends_title')}
          <span className="text-xs text-gray-400 dark:text-gray-500 ml-2">
            ({unit === 'monthly' ? t('settings.aggregation_monthly') : t('settings.aggregation_weekly')})
          </span>
        </h3>
        <div className="flex items-center gap-1">
          <span className="text-xs text-gray-500 dark:text-gray-400 mr-1">{t('dashboard.trend_display_period')}:</span>
          {presets.map(p => (
            <button
              key={p.key}
              onClick={() => onPresetChange(p.key)}
              className={`text-xs px-2 py-1 rounded ${
                trendPreset === p.key
                  ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
                  : 'text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700'
              }`}
            >
              {p.label}
            </button>
          ))}
        </div>
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <ReactECharts option={leadTimeOption} style={{ height }} />
        <ReactECharts option={deployFreqOption} style={{ height }} />
        <ReactECharts option={cfrOption} style={{ height }} />
        <ReactECharts option={mttrOption} style={{ height }} />
      </div>
    </div>
  )
}
