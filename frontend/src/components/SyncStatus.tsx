import { useI18n } from '../i18n/context'
import { useJob, useCancelJob, useStartSync } from '../api/hooks'

interface SyncStatusProps {
  groupId: number
  jobId: number | null
  onJobStarted: (jobId: number) => void
  onComplete: () => void
  lastSyncAt?: string | null
}

export function SyncStatus({ groupId, jobId, onJobStarted, onComplete, lastSyncAt }: SyncStatusProps) {
  const { t } = useI18n()
  const { data: job } = useJob(jobId)
  const startSync = useStartSync()
  const cancelJob = useCancelJob()

  const status = job?.status || 'idle'
  const progress = job?.progress ? JSON.parse(job.progress) : null

  if (status === 'complete' && job) {
    onComplete()
  }

  const handleStart = async () => {
    const result = await startSync.mutateAsync(groupId)
    onJobStarted(result.job_id)
  }

  const formatDateTime = (dateStr: string) => {
    const d = new Date(dateStr)
    return `${d.toLocaleDateString()} ${d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`
  }

  const lastSyncLabel = lastSyncAt ? (
    <span className="text-xs text-gray-500 dark:text-gray-400">
      {t('status.last_sync')}: {formatDateTime(lastSyncAt)}
    </span>
  ) : null

  return (
    <div className="flex items-center gap-3">
      {status === 'idle' && (
        <>
          <button
            onClick={handleStart}
            disabled={startSync.isPending}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 text-sm"
          >
            {t('sync.start')}
          </button>
          {lastSyncLabel}
        </>
      )}

      {status === 'fetching' && progress && (
        <>
          <div className="flex-1">
            <div className="text-sm text-gray-600 dark:text-gray-400 mb-1">
              {t('sync.fetching', { fetched: progress.fetched, total: progress.total })}
            </div>
            <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
              <div
                className="bg-blue-600 h-2 rounded-full transition-all"
                style={{ width: `${progress.total ? (progress.fetched / progress.total) * 100 : 0}%` }}
              />
            </div>
          </div>
          <button
            onClick={() => jobId && cancelJob.mutate(jobId)}
            className="px-3 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
          >
            {t('sync.cancel')}
          </button>
        </>
      )}

      {status === 'computing' && (
        <div className="text-sm text-gray-600 dark:text-gray-400 flex items-center gap-2">
          <div className="animate-spin h-4 w-4 border-2 border-blue-600 border-t-transparent rounded-full" />
          {t('sync.computing')}
        </div>
      )}

      {status === 'complete' && (
        <div className="flex items-center gap-3">
          <button
            onClick={handleStart}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 text-sm"
          >
            {t('sync.start')}
          </button>
          <span className="text-sm text-green-600 dark:text-green-400">{t('sync.complete')}</span>
          {lastSyncLabel}
        </div>
      )}

      {status === 'failed' && (
        <div className="flex items-center gap-3">
          <span className="text-sm text-red-600 dark:text-red-400">
            {t('sync.failed', { error: job?.error || '' })}
          </span>
          <button
            onClick={handleStart}
            className="px-3 py-1 text-sm bg-red-600 text-white rounded hover:bg-red-700"
          >
            {t('sync.retry')}
          </button>
        </div>
      )}

      {status === 'cancelled' && (
        <div className="flex items-center gap-3">
          <span className="text-sm text-gray-500">{t('sync.cancelled')}</span>
          <button
            onClick={handleStart}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 text-sm"
          >
            {t('sync.start')}
          </button>
        </div>
      )}
    </div>
  )
}
