import { useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useI18n } from '../i18n/context'
import { useValidateAuth, useRepos, useCreateGroup, useStartSync, type Repo } from '../api/hooks'

export function SetupPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const [step, setStep] = useState(1)

  // Step 1: PAT
  const [patValid, setPatValid] = useState(false)
  const [login, setLogin] = useState('')
  const validateAuth = useValidateAuth()

  // Step 2: Repo selection
  const { data: repos, refetch: fetchRepos } = useRepos()
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  const [search, setSearch] = useState('')

  // Step 3: Group
  const [groupName, setGroupName] = useState('')
  const createGroup = useCreateGroup()
  const startSync = useStartSync()

  const filteredRepos = useMemo(() => {
    if (!repos) return []
    if (!search) return repos
    const q = search.toLowerCase()
    return repos.filter(r => r.full_name.toLowerCase().includes(q))
  }, [repos, search])

  const handleValidate = async () => {
    try {
      const result = await validateAuth.mutateAsync()
      setLogin(result.login)
      setPatValid(true)
      await fetchRepos()
      setStep(2)
    } catch {
      setPatValid(false)
    }
  }

  const toggleRepo = (id: number) => {
    setSelectedIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const handleCreate = async () => {
    const group = await createGroup.mutateAsync({
      name: groupName,
      period_days: 30,
      repo_ids: Array.from(selectedIds),
    })
    await startSync.mutateAsync(group.id)
    navigate('/dashboard')
  }

  return (
    <div className="max-w-2xl mx-auto py-12 px-4">
      <h2 className="text-2xl font-bold mb-8">{t('setup.title')}</h2>

      {/* Step indicator */}
      <div className="flex items-center gap-2 mb-8">
        {[1, 2, 3].map(s => (
          <div key={s} className="flex items-center gap-2">
            <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium
              ${step >= s ? 'bg-blue-600 text-white' : 'bg-gray-200 dark:bg-gray-700 text-gray-500'}`}>
              {s}
            </div>
            {s < 3 && <div className={`w-12 h-0.5 ${step > s ? 'bg-blue-600' : 'bg-gray-200 dark:bg-gray-700'}`} />}
          </div>
        ))}
      </div>

      {/* Step 1: PAT */}
      {step === 1 && (
        <div className="space-y-4">
          <label className="block text-sm font-medium">{t('setup.pat_label')}</label>
          <p className="text-xs text-gray-500 dark:text-gray-400">{t('setup.pat_help')}</p>
          {validateAuth.isError && (
            <p className="text-sm text-red-600 dark:text-red-400">{t('error.pat_invalid')}</p>
          )}
          {patValid && (
            <p className="text-sm text-green-600 dark:text-green-400">{t('setup.pat_valid', { login })}</p>
          )}
          <button
            onClick={handleValidate}
            disabled={validateAuth.isPending}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
          >
            {validateAuth.isPending ? t('setup.validating') : t('setup.validate')}
          </button>
        </div>
      )}

      {/* Step 2: Repo Selection */}
      {step === 2 && (
        <div className="space-y-4">
          <h3 className="text-lg font-medium">{t('setup.repos_title')}</h3>
          <input
            type="text"
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder={t('setup.repos_search')}
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800"
          />
          <div className="flex items-center justify-between">
            <span className="text-sm text-gray-500">{t('setup.repos_selected', { count: selectedIds.size })}</span>
            <div className="flex gap-2">
              <button
                onClick={() => setSelectedIds(new Set(filteredRepos.map(r => r.id)))}
                className="text-sm text-blue-600 hover:underline"
              >
                {t('setup.repos_select_all')}
              </button>
              <button
                onClick={() => setSelectedIds(new Set())}
                className="text-sm text-gray-500 hover:underline"
              >
                {t('setup.repos_deselect_all')}
              </button>
            </div>
          </div>
          <div className="border border-gray-200 dark:border-gray-700 rounded max-h-80 overflow-y-auto divide-y divide-gray-200 dark:divide-gray-700">
            {filteredRepos.length === 0 && (
              <p className="px-4 py-8 text-center text-gray-400">{t('setup.repos_empty')}</p>
            )}
            {filteredRepos.map((repo: Repo) => (
              <label key={repo.id} className="flex items-center gap-3 px-4 py-2 hover:bg-gray-50 dark:hover:bg-gray-750 cursor-pointer">
                <input
                  type="checkbox"
                  checked={selectedIds.has(repo.id)}
                  onChange={() => toggleRepo(repo.id)}
                  className="rounded"
                />
                <span className="text-sm">{repo.full_name}</span>
              </label>
            ))}
          </div>
          <div className="flex justify-between">
            <button onClick={() => setStep(1)} className="px-4 py-2 border rounded">{t('setup.back')}</button>
            <button
              onClick={() => setStep(3)}
              disabled={selectedIds.size === 0}
              className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
            >
              {t('setup.next')}
            </button>
          </div>
        </div>
      )}

      {/* Step 3: Group Creation */}
      {step === 3 && (
        <div className="space-y-4">
          <h3 className="text-lg font-medium">{t('setup.group_title')}</h3>
          <div>
            <label className="block text-sm font-medium mb-1">{t('setup.group_name')}</label>
            <input
              type="text"
              value={groupName}
              onChange={e => setGroupName(e.target.value)}
              placeholder={t('setup.group_name_placeholder')}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800"
            />
          </div>
          <p className="text-sm text-gray-500">{t('setup.repos_selected', { count: selectedIds.size })}</p>
          <div className="flex justify-between">
            <button onClick={() => setStep(2)} className="px-4 py-2 border rounded">{t('setup.back')}</button>
            <button
              onClick={handleCreate}
              disabled={!groupName || createGroup.isPending}
              className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
            >
              {t('setup.group_create')}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
