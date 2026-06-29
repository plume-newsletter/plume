import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type Form = {
  id: string; listId: string; name: string
  heading: string; description: string; buttonText: string; createdAt: string
}
type FormBody = { listId: string; name: string; heading: string; description: string; buttonText: string }

export function useSignupForms() {
  return useQuery({ queryKey: ['signup-forms'], queryFn: () => api<Form[]>('/api/signup-forms') })
}
export function useCreateForm() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (b: FormBody) => api<Form>('/api/signup-forms', { method: 'POST', body: JSON.stringify(b) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['signup-forms'] }),
  })
}
export function useUpdateForm() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...b }: FormBody & { id: string }) =>
      api<Form>(`/api/signup-forms/${id}`, { method: 'PUT', body: JSON.stringify(b) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['signup-forms'] }),
  })
}
export function useDeleteForm() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api<void>(`/api/signup-forms/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['signup-forms'] }),
  })
}
