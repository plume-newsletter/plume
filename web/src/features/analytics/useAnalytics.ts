import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type Point = { date: string; gained?: number; lost?: number; sent?: number; opens?: number }
export type SendTime = { label: string; rate: number }
export type CampaignMetric = { id: string; subject: string; status: string; sent: number; openRate: number; clickRate: number }
export type TopCampaign = { id: string; subject: string; opens: number }
export type Overview = {
  subscribers: number
  netNewSubs: number
  avgOpenRate: number
  clickRate: number
  sendCost: number
  subscriberGrowth: Point[]
  sendVolume: Point[]
  bestSendTimes: SendTime[]
  campaigns: CampaignMetric[]
  topCampaigns: TopCampaign[]
}

export function useAnalytics(window: 30 | 90) {
  return useQuery({
    queryKey: ['analytics', window],
    queryFn: () => api<Overview>(`/api/analytics/overview?window=${window}`),
  })
}
