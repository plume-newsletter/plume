import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type ReportSummary = {
  recipients: number
  sent: number
  opens: { total: number; unique: number }
  clicks: { total: number; unique: number }
  bounces: number
  complaints: number
  unsubscribes: number
}

export function useReport(id: string) {
  return useQuery({ queryKey: ['report', id], queryFn: () => api<ReportSummary>(`/api/campaigns/${id}/report`) })
}
