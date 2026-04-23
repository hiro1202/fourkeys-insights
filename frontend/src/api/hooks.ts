import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

const BASE = '/api/v1'

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || `HTTP ${res.status}`)
  }
  if (res.status === 204 || res.headers.get('content-length') === '0') {
    return undefined as T
  }
  return res.json()
}

// Auth
export function useValidateAuth() {
  return useMutation({
    mutationFn: () => apiFetch<{ login: string }>('/auth/validate', { method: 'POST' }),
  })
}

// Repos
export function useRepos() {
  return useQuery({
    queryKey: ['repos'],
    queryFn: () => apiFetch<Repo[]>('/repos'),
    enabled: false, // manual trigger
  })
}

// Groups
export function useGroups() {
  return useQuery({
    queryKey: ['groups'],
    queryFn: () => apiFetch<Group[]>('/groups'),
  })
}

export function useCreateGroup() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: { name: string; aggregation_unit: string; repo_ids: number[] }) =>
      apiFetch<Group>('/groups', { method: 'POST', body: JSON.stringify(data) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['groups'] }),
  })
}

export function useDeleteGroup() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (groupId: number) =>
      apiFetch<void>(`/groups/${groupId}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['groups'] }),
  })
}

// Metrics
export function useGroupMetrics(groupId: number | null) {
  return useQuery({
    queryKey: ['metrics', groupId],
    queryFn: () => apiFetch<ExtendedMetricsResult>(`/groups/${groupId}/metrics`),
    enabled: groupId !== null,
  })
}

// Trends
export function useGroupTrends(groupId: number | null, params?: { since?: string; until?: string; unit?: string }) {
  const qs = new URLSearchParams()
  if (params?.since) qs.set('since', params.since)
  if (params?.until) qs.set('until', params.until)
  if (params?.unit) qs.set('unit', params.unit)
  const query = qs.toString()

  return useQuery({
    queryKey: ['trends', groupId, params],
    queryFn: () => apiFetch<TrendsResult>(`/groups/${groupId}/trends${query ? `?${query}` : ''}`),
    enabled: groupId !== null,
  })
}

// Group Settings
export function useGroupSettings(groupId: number | null) {
  return useQuery({
    queryKey: ['group-settings', groupId],
    queryFn: () => apiFetch<GroupSettings>(`/groups/${groupId}/settings`),
    enabled: groupId !== null,
  })
}

export function useUpdateGroupSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ groupId, ...data }: {
      groupId: number
      name?: string
      aggregation_unit?: string
      lead_time_start?: string
      mttr_start?: string
      incident_rules?: string
    }) =>
      apiFetch<{ status: string }>(`/groups/${groupId}/settings`, { method: 'PUT', body: JSON.stringify(data) }),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: ['group-settings', variables.groupId] })
      qc.invalidateQueries({ queryKey: ['metrics', variables.groupId] })
      qc.invalidateQueries({ queryKey: ['trends', variables.groupId] })
    },
  })
}

// Pulls
export function useGroupPulls(groupId: number | null, page: number) {
  return useQuery({
    queryKey: ['pulls', groupId, page],
    queryFn: () => apiFetch<{ pulls: PullRequest[]; total: number }>(`/groups/${groupId}/pulls?page=${page}&per_page=20`),
    enabled: groupId !== null,
  })
}

// Sync
export function useStartSync() {
  return useMutation({
    mutationFn: (groupId: number) =>
      apiFetch<{ job_id: number }>(`/groups/${groupId}/sync`, { method: 'POST' }),
  })
}

export function useJob(jobId: number | null) {
  return useQuery({
    queryKey: ['job', jobId],
    queryFn: () => apiFetch<Job>(`/jobs/${jobId}`),
    enabled: jobId !== null,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      if (status === 'fetching' || status === 'computing') return 1000
      return false
    },
  })
}

export function useCancelJob() {
  return useMutation({
    mutationFn: (jobId: number) =>
      apiFetch<unknown>(`/jobs/${jobId}/cancel`, { method: 'POST' }),
  })
}

// Types
export interface Repo {
  id: number
  owner: string
  name: string
  full_name: string
  default_branch: string
}

export interface Group {
  id: number
  name: string
  aggregation_unit: string
  lead_time_start: string
  mttr_start: string
  incident_rules: string
  repos?: Repo[]
}

export interface FourKeysResult {
  lead_time_hours: number
  deploy_frequency: number
  change_failure_rate: number
  mttr_hours: number | null
  lead_time_level: string
  deploy_frequency_level: string
  change_failure_rate_level: string
  mttr_level: string | null
  overall_level: string
  total_prs: number
  incident_prs: number
  period_days: number
  fallback_count: number
}

export interface ExtendedMetricsResult extends FourKeysResult {
  previous_period?: FourKeysResult
  last_sync_at?: string
  period_start: string
  period_end: string
  aggregation_unit: string
  lead_time_start: string
  mttr_start: string
}

export interface TrendDataPoint {
  period_start: string
  period_end: string
  lead_time_hours: number
  deploy_frequency: number
  change_failure_rate: number
  mttr_hours: number | null
  total_prs: number
  incident_prs: number
}

export interface TrendsResult {
  data_points: TrendDataPoint[]
  unit: string
  since: string
  until: string
}

export interface RepoFallbackStats {
  repo_id: number
  total_prs: number
  lead_time_fallbacks: number
  mttr_fallbacks: number
}

export interface GroupSettings {
  name: string
  aggregation_unit: string
  lead_time_start: string
  mttr_start: string
  incident_rules: string
  repos: Repo[]
  fallback_stats?: RepoFallbackStats[]
}

export interface PullRequest {
  id: number
  repo_id: number
  pr_number: number
  title: string
  branch_name: string | null
  labels: string | null
  merged_at: string
  additions: number
  deletions: number
  repo_full_name: string
}

export interface Job {
  id: number
  group_id: number
  status: string
  progress: string | null
  error: string | null
}
