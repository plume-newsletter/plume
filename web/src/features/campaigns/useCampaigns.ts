import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type Campaign = {
  id: string
  owner_id: string
  brand_id: string
  subject: string
  html_body: string
  plain_body: string
  body_json: string
  status: string
  scheduled_at: string | null
  created_at: string
}
export type CreateCampaignInput = { brandId: string; subject: string; htmlBody: string; plainBody: string }

export function useCampaigns() {
  return useQuery({ queryKey: ['campaigns'], queryFn: () => api<Campaign[]>('/api/campaigns') })
}
export function useCreateCampaign() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateCampaignInput) =>
      api<Campaign>('/api/campaigns', { method: 'POST', body: JSON.stringify(input) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['campaigns'] }),
  })
}
export function useDeleteCampaign() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api<void>(`/api/campaigns/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['campaigns'] }),
  })
}
