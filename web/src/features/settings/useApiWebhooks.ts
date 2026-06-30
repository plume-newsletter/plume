import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type ApiKey = { id: string; name: string; prefix: string; createdAt: string; lastUsedAt: string | null }
export type Webhook = { id: string; url: string; secret: string; events: string[]; active: boolean; createdAt: string }

export function useApiKeys() {
  return useQuery({ queryKey: ['api-keys'], queryFn: () => api<ApiKey[]>('/api/api-keys') })
}

export function useCreateApiKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) =>
      api<{ key: ApiKey; secret: string }>('/api/api-keys', { method: 'POST', body: JSON.stringify({ name }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['api-keys'] }),
  })
}

export function useDeleteApiKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api<void>(`/api/api-keys/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['api-keys'] }),
  })
}

export function useWebhooks() {
  return useQuery({ queryKey: ['webhooks'], queryFn: () => api<{ endpoints: Webhook[]; events: string[] }>('/api/webhooks') })
}

export function useCreateWebhook() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: { url: string; events: string[] }) =>
      api<Webhook>('/api/webhooks', { method: 'POST', body: JSON.stringify(input) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['webhooks'] }),
  })
}

export function useDeleteWebhook() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api<void>(`/api/webhooks/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['webhooks'] }),
  })
}
