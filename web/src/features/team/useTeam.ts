import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type Member = { id: string; email: string; fullName: string; role: string }
export type Invite = { id: string; email: string; role: string; token: string; expiresAt: string }

export function useTeam() {
  return useQuery({ queryKey: ['team'], queryFn: () => api<Member[]>('/api/team') })
}
export function useInvites() {
  return useQuery({ queryKey: ['team-invites'], queryFn: () => api<Invite[]>('/api/team/invites') })
}
export function useInvite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (b: { email: string; role: string }) =>
      api<{ invite: Invite; acceptUrl: string }>('/api/team/invites', { method: 'POST', body: JSON.stringify(b) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['team-invites'] }),
  })
}
export function useRevokeInvite() {
  const qc = useQueryClient()
  return useMutation({ mutationFn: (id: string) => api<void>(`/api/team/invites/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['team-invites'] }) })
}
export function useSetRole() {
  const qc = useQueryClient()
  return useMutation({ mutationFn: ({ id, role }: { id: string; role: string }) => api<void>(`/api/team/members/${id}/role`, { method: 'PUT', body: JSON.stringify({ role }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['team'] }) })
}
export function useRemoveMember() {
  const qc = useQueryClient()
  return useMutation({ mutationFn: (id: string) => api<void>(`/api/team/members/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['team'] }) })
}
export function useRenameWorkspace() {
  return useMutation({ mutationFn: (name: string) => api<void>('/api/workspace', { method: 'PUT', body: JSON.stringify({ name }) }) })
}
