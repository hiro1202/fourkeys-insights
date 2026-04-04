import ReactECharts from 'echarts-for-react'
import { useI18n } from '../i18n/context'
import type { PullRequest } from '../api/hooks'

interface PRSizeChartProps {
  pulls: PullRequest[]
}

export function PRSizeChart({ pulls }: PRSizeChartProps) {
  const { t } = useI18n()

  const buckets = { 'XS (0-50)': 0, 'S (51-200)': 0, 'M (201-500)': 0, 'L (501+)': 0 }

  for (const pr of pulls) {
    const lines = pr.additions + pr.deletions
    if (lines <= 50) buckets['XS (0-50)']++
    else if (lines <= 200) buckets['S (51-200)']++
    else if (lines <= 500) buckets['M (201-500)']++
    else buckets['L (501+)']++
  }

  const option = {
    tooltip: { trigger: 'axis' as const },
    xAxis: {
      type: 'category' as const,
      data: Object.keys(buckets),
    },
    yAxis: { type: 'value' as const },
    series: [{
      data: Object.values(buckets),
      type: 'bar' as const,
      itemStyle: {
        color: '#3b82f6',
        borderRadius: [4, 4, 0, 0],
      },
    }],
  }

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
      <h3 className="font-medium mb-3">{t('dashboard.pr_size')}</h3>
      <ReactECharts option={option} style={{ height: 250 }} />
    </div>
  )
}
