import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'

export function useSendCampaign(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (listId: string) =>
      api<{ recipients: number }>(`/api/campaigns/${id}/send`, {
        method: 'POST',
        body: JSON.stringify({ listId }),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['campaigns', id] })
      qc.invalidateQueries({ queryKey: ['campaigns'] })
    },
  })
}
