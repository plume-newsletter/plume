import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type Step = { kind: string; subject: string; html: string; waitDays: number }
export type Automation = { id: string; name: string; listId: string; status: string; createdAt: string; steps: Step[]; stepSends: number; inFlow: number; completePct: number }

export function useAutomations() {
  return useQuery({ queryKey: ['automations'], queryFn: () => api<Automation[]>('/api/automations') })
}
export function useCreateAutomation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (b: { name: string; listId: string }) => api<Automation>('/api/automations', { method: 'POST', body: JSON.stringify(b) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['automations'] }),
  })
}
export function useReplaceSteps() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, steps }: { id: string; steps: Step[] }) => api<void>(`/api/automations/${id}/steps`, { method: 'PUT', body: JSON.stringify({ steps }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['automations'] }),
  })
}
export function useSetStatus() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) => api<void>(`/api/automations/${id}/status`, { method: 'POST', body: JSON.stringify({ status }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['automations'] }),
  })
}
export function useDeleteAutomation() {
  const qc = useQueryClient()
  return useMutation({ mutationFn: (id: string) => api<void>(`/api/automations/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['automations'] }) })
}
