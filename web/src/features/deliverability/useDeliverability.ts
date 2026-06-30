import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type Suppression = { email: string; reason: string; date: string }
export type Deliverability = {
  sent: number
  bounces: number
  complaints: number
  bounceRate: number
  complaintRate: number
  suppressed: number
  suppressions: Suppression[]
}

export function useDeliverability(window: 30 | 90) {
  return useQuery({
    queryKey: ['deliverability', window],
    queryFn: () => api<Deliverability>(`/api/analytics/deliverability?window=${window}`),
  })
}
