import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type Condition = { type: string; op: string; days?: number; field?: string; value?: string }
export type SubscriberLite = { id: string; email: string; name: string; status: string }
export type Preview = { count: number; total: number; percent: number; sample: SubscriberLite[] }
export type Segment = { id: string; name: string; match: string; conditions: Condition[]; count: number; createdAt: string }

export function useSegments() {
  return useQuery({ queryKey: ['segments'], queryFn: () => api<Segment[]>('/api/segments') })
}
export function useSegmentFields() {
  return useQuery({ queryKey: ['segment-fields'], queryFn: () => api<string[]>('/api/segments/fields') })
}
export function useSegmentPreview() {
  return useMutation({
    mutationFn: (b: { match: string; conditions: Condition[] }) =>
      api<Preview>('/api/segments/preview', { method: 'POST', body: JSON.stringify(b) }),
  })
}
export function useCreateSegment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (b: { name: string; match: string; conditions: Condition[] }) =>
      api<Segment>('/api/segments', { method: 'POST', body: JSON.stringify(b) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['segments'] }),
  })
}
export function useUpdateSegment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...b }: { id: string; name: string; match: string; conditions: Condition[] }) =>
      api<Segment>(`/api/segments/${id}`, { method: 'PUT', body: JSON.stringify(b) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['segments'] }),
  })
}
export function useDeleteSegment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api<void>(`/api/segments/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['segments'] }),
  })
}
