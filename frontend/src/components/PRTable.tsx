import { useI18n } from '../i18n/context'
import { useGroupPulls, type PullRequest } from '../api/hooks'
import { useState } from 'react'

interface PRTableProps {
  groupId: number
}

export function PRTable({ groupId }: PRTableProps) {
  const { t } = useI18n()
  const [page, setPage] = useState(1)
  const { data } = useGroupPulls(groupId, page)

  const pulls = data?.pulls || []
  const total = data?.total || 0
  const totalPages = Math.ceil(total / 20)

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
      <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700">
        <h3 className="font-medium">{t('dashboard.pr_table_title')}</h3>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-4 py-2 text-left">{t('dashboard.pr_number')}</th>
              <th className="px-4 py-2 text-left">{t('dashboard.pr_title')}</th>
              <th className="px-4 py-2 text-left">{t('dashboard.pr_repo')}</th>
              <th className="px-4 py-2 text-left">{t('dashboard.pr_branch')}</th>
              <th className="px-4 py-2 text-left">{t('dashboard.pr_merged')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
            {pulls.map((pr: PullRequest) => (
              <tr key={pr.id} className="hover:bg-gray-50 dark:hover:bg-gray-750">
                <td className="px-4 py-2 font-mono">#{pr.pr_number}</td>
                <td className="px-4 py-2 max-w-xs truncate">{pr.title}</td>
                <td className="px-4 py-2 text-gray-500 dark:text-gray-400">{pr.repo_full_name}</td>
                <td className="px-4 py-2 text-gray-500 dark:text-gray-400 font-mono text-xs">
                  {pr.branch_name || '-'}
                </td>
                <td className="px-4 py-2 text-gray-500 dark:text-gray-400">
                  {new Date(pr.merged_at).toLocaleDateString()}
                </td>
              </tr>
            ))}
            {pulls.length === 0 && (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-gray-400">
                  {t('dashboard.no_data')}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      {totalPages > 1 && (
        <div className="px-4 py-3 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between">
          <span className="text-sm text-gray-500">{total} PRs</span>
          <div className="flex gap-2">
            <button
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="px-3 py-1 text-sm border rounded disabled:opacity-50"
            >
              Prev
            </button>
            <span className="text-sm py-1">{page} / {totalPages}</span>
            <button
              onClick={() => setPage(p => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
              className="px-3 py-1 text-sm border rounded disabled:opacity-50"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
