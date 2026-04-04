import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

const BASE = '/api/v1'

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || `HTTP ${res.status}`)
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
    mutationFn: (data: { name: string; period_days: number; repo_ids: number[] }) =>
      apiFetch<Group>('/groups', { method: 'POST', body: JSON.stringify(data) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['groups'] }),
  })
}

// Metrics
export function useGroupMetrics(groupId: number | null) {
  return useQuery({
    queryKey: ['metrics', groupId],
    queryFn: () => apiFetch<FourKeysResult>(`/groups/${groupId}/metrics`),
    enabled: groupId !== null,
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
  period_days: number
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
