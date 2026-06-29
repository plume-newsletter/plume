import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type Me = { email: string; fullName: string; role: string; workspaceName: string }

export function useMe() {
  return useQuery({ queryKey: ['me'], queryFn: () => api<Me>('/api/me') })
}

export function useLogin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (c: { email: string; password: string }) =>
      api<Me>('/api/login', { method: 'POST', body: JSON.stringify(c) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['me'] }),
  })
}

export function useLogout() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api<void>('/api/logout', { method: 'POST' }),
    onSuccess: () => qc.clear(),
  })
}
