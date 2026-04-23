import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { apiFetch } from './hooks'

describe('apiFetch', () => {
  const fetchMock = vi.fn<typeof fetch>()

  beforeEach(() => {
    fetchMock.mockReset()
    vi.stubGlobal('fetch', fetchMock)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('returns parsed JSON for a 200 response with a body', async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ id: 1, name: 'foo' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )

    const result = await apiFetch<{ id: number; name: string }>('/groups/1')
    expect(result).toEqual({ id: 1, name: 'foo' })
  })

  it('resolves without throwing for a 204 No Content response (DELETE)', async () => {
    fetchMock.mockResolvedValueOnce(new Response(null, { status: 204 }))

    const result = await apiFetch<void>('/groups/1', { method: 'DELETE' })
    expect(result).toBeUndefined()
  })

  it('resolves without throwing when content-length is 0', async () => {
    fetchMock.mockResolvedValueOnce(
      new Response('', {
        status: 200,
        headers: { 'content-length': '0' },
      }),
    )

    const result = await apiFetch<void>('/groups/1', { method: 'DELETE' })
    expect(result).toBeUndefined()
  })

  it('throws the server-provided error message on a 4xx/5xx response', async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ error: 'Invalid group ID' }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' },
      }),
    )

    await expect(apiFetch('/groups/abc', { method: 'DELETE' })).rejects.toThrow(
      'Invalid group ID',
    )
  })

  it('falls back to HTTP status in the error message when the body has no error field', async () => {
    fetchMock.mockResolvedValueOnce(new Response('oops', { status: 500 }))

    await expect(apiFetch('/groups/1', { method: 'DELETE' })).rejects.toThrow(
      'HTTP 500',
    )
  })
})
