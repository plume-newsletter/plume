import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type Template = {
  id: string
  name: string
  category: string
  bodyJson: unknown[]
  prebuilt: boolean
  createdAt: string
}

export function useTemplates(category?: string) {
  const qs = category && category !== 'All' ? `?category=${encodeURIComponent(category)}` : ''
  return useQuery({ queryKey: ['templates', category ?? 'All'], queryFn: () => api<Template[]>(`/api/templates${qs}`) })
}

export function useCreateTemplate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (b: { name: string; category: string; bodyJson: unknown[] }) =>
      api<Template>('/api/templates', { method: 'POST', body: JSON.stringify(b) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['templates'] }),
  })
}

export function useDeleteTemplate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api<void>(`/api/templates/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['templates'] }),
  })
}

export function useUseTemplate() {
  return useMutation({
    mutationFn: (b: { id: string; brandId: string; subject: string }) =>
      api<{ campaignId: string }>(`/api/templates/${b.id}/use`, { method: 'POST', body: JSON.stringify({ brandId: b.brandId, subject: b.subject }) }),
  })
}
