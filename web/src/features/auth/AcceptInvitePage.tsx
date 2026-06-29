import { useNavigate, useParams, Link } from 'react-router-dom'
import { useQuery, useMutation } from '@tanstack/react-query'
import { useState } from 'react'
import { Feather } from 'lucide-react'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type InviteInfo = { email: string; workspaceName: string }

export function AcceptInvitePage() {
  const { token } = useParams<{ token: string }>()
  const nav = useNavigate()

  const { data, isError, isPending } = useQuery<InviteInfo>({
    queryKey: ['invite', token],
    queryFn: () => api<InviteInfo>(`/api/invites/${token}`),
    retry: false,
  })

  const accept = useMutation({
    mutationFn: (body: { fullName: string; password: string }) =>
      api<{ email: string }>(`/api/invites/${token}/accept`, {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    onSuccess: () => nav('/', { replace: true }),
  })

  const [fullName, setFullName] = useState('')
  const [password, setPassword] = useState('')
  const [validationError, setValidationError] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setValidationError('')
    if (password.length < 8) {
      setValidationError('Password must be at least 8 characters')
      return
    }
    accept.mutate({ fullName, password })
  }

  return (
    <div className="grid min-h-screen lg:grid-cols-2 bg-background text-foreground">
      {/* Left: marketing hero */}
      <div className="hidden lg:flex lg:flex-col lg:justify-between bg-sidebar p-12 border-r">
        <div className="flex items-center gap-2">
          <span className="flex size-9 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Feather className="size-5" aria-hidden="true" />
          </span>
          <span className="text-xl font-semibold">Plume</span>
        </div>
        <div className="max-w-md">
          <h2 className="text-3xl font-bold leading-tight">
            You've been invited to collaborate.
          </h2>
          <p className="mt-3 text-muted-foreground">
            Join your team on Plume — the self-hosted newsletter platform you actually own.
          </p>
        </div>
        <p className="font-mono text-xs text-faint">$ docker compose up -d &nbsp;·&nbsp; AGPL-3.0</p>
      </div>

      {/* Right: form */}
      <div className="flex items-center justify-center p-6">
        <div className="w-full max-w-sm">
          <div className="mb-6 flex items-center gap-2 lg:hidden">
            <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <Feather className="size-4" aria-hidden="true" />
            </span>
            <span className="text-lg font-semibold">Plume</span>
          </div>

          {isPending && (
            <p className="text-sm text-muted-foreground">Loading invite…</p>
          )}

          {isError && (
            <div className="space-y-4">
              <h1 className="text-2xl font-semibold">Invite no longer valid</h1>
              <p className="text-sm text-muted-foreground">
                This invite link has expired, already been used, or is invalid.
              </p>
              <Link to="/login" className="text-sm text-primary hover:underline">
                Go to login
              </Link>
            </div>
          )}

          {data && (
            <>
              <h1 className="text-2xl font-semibold">Join {data.workspaceName}</h1>
              <p className="mt-1 text-sm text-muted-foreground">
                You've been invited as <span className="font-medium">{data.email}</span>. Set a password to get started.
              </p>

              <form onSubmit={handleSubmit} className="mt-6 space-y-4">
                <div className="space-y-1">
                  <Label htmlFor="fullName">Full name</Label>
                  <Input
                    id="fullName"
                    type="text"
                    value={fullName}
                    onChange={(e) => setFullName(e.target.value)}
                    required
                  />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="password">Password</Label>
                  <Input
                    id="password"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                  />
                </div>
                {validationError && (
                  <p className="text-sm text-destructive">{validationError}</p>
                )}
                {accept.isError && (
                  <p className="text-sm text-destructive">Something went wrong. Please try again.</p>
                )}
                <Button type="submit" className="w-full" disabled={accept.isPending}>
                  {accept.isPending ? 'Joining…' : 'Accept invite'}
                </Button>
              </form>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
