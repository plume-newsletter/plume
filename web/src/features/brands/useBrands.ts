import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type Brand = {
  id: string
  owner_id: string
  name: string
  from_name: string
  from_email: string
  reply_to: string
  created_at: string
}
export type BrandInput = { name: string; fromName: string; fromEmail: string; replyTo: string }

export function useBrands() {
  return useQuery({ queryKey: ['brands'], queryFn: () => api<Brand[]>('/api/brands') })
}
export function useCreateBrand() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (b: BrandInput) => api<Brand>('/api/brands', { method: 'POST', body: JSON.stringify(b) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['brands'] }),
  })
}
export function useUpdateBrand() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...b }: BrandInput & { id: string }) =>
      api<Brand>(`/api/brands/${id}`, { method: 'PUT', body: JSON.stringify(b) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['brands'] }),
  })
}
export function useDeleteBrand() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api<void>(`/api/brands/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['brands'] }),
  })
}
