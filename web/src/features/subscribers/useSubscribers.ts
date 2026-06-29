import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type Subscriber = {
  id: string; owner_id: string; list_id: string
  email: string; name: string; status: string; created_at: string
}

export function useSubscribers(listId: string) {
  return useQuery({
    queryKey: ['subscribers', listId],
    queryFn: () => api<Subscriber[]>(`/api/lists/${listId}/subscribers`),
  })
}

export function useAddSubscriber(listId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: { email: string; name: string; status?: string }) =>
      api<Subscriber>(`/api/lists/${listId}/subscribers`, { method: 'POST', body: JSON.stringify(input) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['subscribers', listId] }),
  })
}

export function useSetSubscriberStatus(listId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      api<Subscriber>(`/api/subscribers/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['subscribers', listId] }),
  })
}

export function useDeleteSubscriber(listId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api<void>(`/api/subscribers/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['subscribers', listId] }),
  })
}
