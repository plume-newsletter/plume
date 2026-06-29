import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { Campaign } from './useCampaigns'

export type UpdateCampaignInput = { subject: string; bodyJson: string }

export function useCampaign(id: string) {
  return useQuery({ queryKey: ['campaigns', id], queryFn: () => api<Campaign>(`/api/campaigns/${id}`) })
}

export function useUpdateCampaign(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: UpdateCampaignInput) =>
      api<Campaign>(`/api/campaigns/${id}`, { method: 'PUT', body: JSON.stringify(input) }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['campaigns', id] })
      qc.invalidateQueries({ queryKey: ['campaigns'] })
    },
  })
}
