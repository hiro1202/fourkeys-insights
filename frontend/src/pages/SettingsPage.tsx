import { useState, useEffect } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { useI18n } from '../i18n/context'
import { useGroupSettings, useUpdateGroupSettings, useGroups, useDeleteGroup } from '../api/hooks'

export function SettingsPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const groupIdParam = searchParams.get('groupId')
  const { data: groups } = useGroups()
  const groupId = groupIdParam ? Number(groupIdParam) : groups?.[0]?.id ?? null
  const { data: settings } = useGroupSettings(groupId)
  const updateSettings = useUpdateGroupSettings()
  const deleteGroup = useDeleteGroup()

  const [aggregationUnit, setAggregationUnit] = useState('weekly')
  const [leadTimeStart, setLeadTimeStart] = useState('first_commit_at')
  const [mttrStart, setMttrStart] = useState('first_commit_at')
  const [titleKeywords, setTitleKeywords] = useState('revert, hotfix')
  const [branchKeywords, setBranchKeywords] = useState('hotfix')
  const [labels, setLabels] = useState('incident, bug')
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    if (settings) {
      setAggregationUnit(settings.aggregation_unit || 'weekly')
      setLeadTimeStart(settings.lead_time_start || 'first_commit_at')
      setMttrStart(settings.mttr_start || 'first_commit_at')
      if (settings.incident_rules) {
        try {
          const rules = JSON.parse(settings.incident_rules)
          if (rules.title_keywords) setTitleKeywords(rules.title_keywords.join(', '))
          if (rules.branch_keywords) setBranchKeywords(rules.branch_keywords.join(', '))
          if (rules.labels) setLabels(rules.labels.join(', '))
        } catch {
          // use defaults
        }
      }
    }
  }, [settings])

  const handleSave = async () => {
    if (!groupId) return
    const incidentRules = JSON.stringify({
      title_keywords: titleKeywords.split(',').map(s => s.trim()).filter(Boolean),
      branch_keywords: branchKeywords.split(',').map(s => s.trim()).filter(Boolean),
      labels: labels.split(',').map(s => s.trim()).filter(Boolean),
    })

    await updateSettings.mutateAsync({
      groupId,
      aggregation_unit: aggregationUnit,
      lead_time_start: leadTimeStart,
      mttr_start: mttrStart,
      incident_rules: incidentRules,
    })
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  const handleDelete = async () => {
    if (!groupId) return
    if (!window.confirm(t('settings.delete_group_confirm'))) return
    await deleteGroup.mutateAsync(groupId)
    navigate('/dashboard')
  }

  const leadTimeStartOptions = [
    { value: 'first_commit_at', label: t('settings.lead_time_first_commit_recommended'), desc: t('settings.lead_time_first_commit_desc') },
    { value: 'issue.created_at', label: t('settings.lead_time_issue_created'), desc: t('settings.lead_time_issue_created_desc') },
    { value: 'pr_created_at', label: t('settings.lead_time_pr_created'), desc: t('settings.lead_time_pr_created_desc') },
  ]

  const mttrStartOptions = [
    { value: 'first_commit_at', label: t('settings.lead_time_first_commit'), desc: t('settings.lead_time_first_commit_desc') },
    { value: 'issue.created_at', label: t('settings.lead_time_issue_created'), desc: t('settings.lead_time_issue_created_desc') },
    { value: 'pr_created_at', label: t('settings.lead_time_pr_created'), desc: t('settings.lead_time_pr_created_desc') },
  ]

  return (
    <div className="max-w-2xl mx-auto py-8 px-4 space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold">{t('settings.title')}</h2>
        <button
          onClick={() => navigate('/dashboard')}
          className="text-sm text-blue-600 dark:text-blue-400 hover:underline"
        >
          {t('dashboard.title')} →
        </button>
      </div>

      {/* Group selector */}
      {groups && groups.length > 1 ? (
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {t('dashboard.group_select')}
          </label>
          <select
            value={groupId ?? ''}
            onChange={e => navigate(`/settings?groupId=${e.target.value}`)}
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800"
          >
            {groups.map(g => (
              <option key={g.id} value={g.id}>{g.name}</option>
            ))}
          </select>
        </div>
      ) : groups && groups.length === 1 ? (
        <p className="text-sm text-gray-600 dark:text-gray-400">{groups[0].name}</p>
      ) : null}

      {/* Aggregation Unit */}
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {t('settings.aggregation_unit')}
        </label>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-2">{t('settings.aggregation_unit_desc')}</p>
        <select
          value={aggregationUnit}
          onChange={e => setAggregationUnit(e.target.value)}
          className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800"
        >
          <option value="weekly">{t('settings.aggregation_weekly')}</option>
          <option value="monthly">{t('settings.aggregation_monthly')}</option>
        </select>
        <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">
          {aggregationUnit === 'monthly' ? t('settings.aggregation_monthly_desc') : t('settings.aggregation_weekly_desc')}
        </p>
      </div>

      {/* Lead Time Start Point */}
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {t('settings.lead_time_start')}
        </label>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-2">{t('settings.lead_time_start_desc')}</p>
        <select
          value={leadTimeStart}
          onChange={e => setLeadTimeStart(e.target.value)}
          className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800"
        >
          {leadTimeStartOptions.map(opt => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>
        <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">
          {leadTimeStartOptions.find(o => o.value === leadTimeStart)?.desc}
        </p>
      </div>

      {/* MTTR Start Point */}
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {t('settings.mttr_start')}
        </label>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-2">{t('settings.mttr_start_desc')}</p>
        <select
          value={mttrStart}
          onChange={e => setMttrStart(e.target.value)}
          className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800"
        >
          {mttrStartOptions.map(opt => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>
        <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">
          {mttrStartOptions.find(o => o.value === mttrStart)?.desc}
        </p>
      </div>

      {/* Incident Detection Rules */}
      <div className="space-y-3">
        <div>
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">
            {t('settings.incident_rules')}
          </h3>
          <p className="text-xs text-amber-600 dark:text-amber-400 mt-1">{t('settings.incident_rules_desc')}</p>
        </div>

        <div>
          <label className="block text-xs text-gray-500 dark:text-gray-400 mb-1">
            {t('settings.title_keywords')}
          </label>
          <input
            type="text"
            value={titleKeywords}
            onChange={e => setTitleKeywords(e.target.value)}
            placeholder="revert, hotfix"
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800 text-sm"
          />
          <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">{t('settings.title_keywords_desc')}</p>
        </div>

        <div>
          <label className="block text-xs text-gray-500 dark:text-gray-400 mb-1">
            {t('settings.branch_keywords')}
          </label>
          <input
            type="text"
            value={branchKeywords}
            onChange={e => setBranchKeywords(e.target.value)}
            placeholder="hotfix"
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800 text-sm"
          />
          <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">{t('settings.branch_keywords_desc')}</p>
        </div>

        <div>
          <label className="block text-xs text-gray-500 dark:text-gray-400 mb-1">
            {t('settings.incident_labels')}
          </label>
          <input
            type="text"
            value={labels}
            onChange={e => setLabels(e.target.value)}
            placeholder="incident, bug"
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800 text-sm"
          />
          <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">{t('settings.incident_labels_desc')}</p>
        </div>
      </div>

      {/* Save */}
      <div className="flex items-center gap-3">
        <button
          onClick={handleSave}
          disabled={updateSettings.isPending}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
        >
          {updateSettings.isPending ? '...' : t('settings.save')}
        </button>
        {saved && (
          <span className="text-sm text-green-600 dark:text-green-400">{t('settings.saved')}</span>
        )}
      </div>

      {/* Repositories (display only) */}
      {settings?.repos && settings.repos.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {t('settings.repos_title')}
          </h3>
          <div className="border border-gray-200 dark:border-gray-700 rounded divide-y divide-gray-200 dark:divide-gray-700">
            {settings.repos.map(repo => (
              <div key={repo.id} className="px-3 py-2 text-sm text-gray-700 dark:text-gray-300">
                {repo.full_name}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Delete Group */}
      <div className="pt-4 border-t border-gray-200 dark:border-gray-700">
        <button
          onClick={handleDelete}
          disabled={deleteGroup.isPending}
          className="px-4 py-2 text-sm text-red-600 border border-red-300 rounded hover:bg-red-50 dark:text-red-400 dark:border-red-800 dark:hover:bg-red-900/20 disabled:opacity-50"
        >
          {t('settings.delete_group')}
        </button>
      </div>
    </div>
  )
}
