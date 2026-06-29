import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type Settings = {
  sesConfigured: boolean
  sesRegion: string
  aiConfigured: boolean
  aiModel: string
}
export type SESInput = { accessKeyId: string; secretAccessKey: string; region: string }
export type AIInput = { apiKey: string; model: string }

export function useSettings() {
  return useQuery({ queryKey: ['settings'], queryFn: () => api<Settings>('/api/settings') })
}

export function useSaveSES() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: SESInput) =>
      api<void>('/api/settings/ses', { method: 'PUT', body: JSON.stringify(input) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings'] }),
  })
}

export function useSaveAI() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: AIInput) =>
      api<void>('/api/settings/ai', { method: 'PUT', body: JSON.stringify(input) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings'] }),
  })
}
