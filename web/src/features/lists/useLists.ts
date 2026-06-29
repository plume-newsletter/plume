import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type List = { id: string; owner_id: string; brand_id: string; name: string; created_at: string }

export function useLists() {
  return useQuery({ queryKey: ['lists'], queryFn: () => api<List[]>('/api/lists') })
}
export function useCreateList() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: { brandId: string; name: string }) =>
      api<List>('/api/lists', { method: 'POST', body: JSON.stringify(input) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['lists'] }),
  })
}
export function useUpdateList() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      api<List>(`/api/lists/${id}`, { method: 'PUT', body: JSON.stringify({ name }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['lists'] }),
  })
}
export function useDeleteList() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api<void>(`/api/lists/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['lists'] }),
  })
}
