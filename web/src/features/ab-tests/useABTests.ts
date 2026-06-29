import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type ABTest = { id: string; campaignId: string; listId: string; subjectA: string; subjectB: string; testPercent: number; status: string; winner: string; createdAt: string }
export type VariantResult = { variant: string; subject: string; sent: number; openRate: number; clickRate: number }
export type Results = { status: string; winner: string; variants: VariantResult[] }

export function useABTests() {
  return useQuery({ queryKey: ['ab-tests'], queryFn: () => api<ABTest[]>('/api/ab-tests') })
}
export function useABTestResults(id: string, enabled: boolean) {
  return useQuery({ queryKey: ['ab-test-results', id], queryFn: () => api<Results>(`/api/ab-tests/${id}/results`), enabled })
}
export function useCreateABTest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (b: { campaignId: string; listId: string; subjectA: string; subjectB: string; testPercent: number }) =>
      api<ABTest>('/api/ab-tests', { method: 'POST', body: JSON.stringify(b) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['ab-tests'] }),
  })
}
export function useStartABTest() {
  const qc = useQueryClient()
  return useMutation({ mutationFn: (id: string) => api<void>(`/api/ab-tests/${id}/start`, { method: 'POST' }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['ab-tests'] }); qc.invalidateQueries({ queryKey: ['ab-test-results'] }) } })
}
export function useSendWinner() {
  const qc = useQueryClient()
  return useMutation({ mutationFn: ({ id, winner }: { id: string; winner: string }) => api<void>(`/api/ab-tests/${id}/winner`, { method: 'POST', body: JSON.stringify({ winner }) }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['ab-tests'] }); qc.invalidateQueries({ queryKey: ['ab-test-results'] }) } })
}
export function useDeleteABTest() {
  const qc = useQueryClient()
  return useMutation({ mutationFn: (id: string) => api<void>(`/api/ab-tests/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['ab-tests'] }) })
}
