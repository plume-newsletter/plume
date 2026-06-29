import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { toast } from 'sonner'
import { CheckCircle2, Sparkles } from 'lucide-react'
import { useSettings, useSaveSES, useSaveAI } from './useSettings'
import { useMe } from '@/features/auth/useAuth'
import { useTeam, useInvites, useInvite, useRevokeInvite, useSetRole, useRemoveMember, useRenameWorkspace } from '@/features/team/useTeam'
import { useTheme } from '@/components/theme/ThemeProvider'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'

const schema = z.object({
  accessKeyId: z.string().min(1),
  secretAccessKey: z.string().min(1),
  region: z.string().min(1),
})
type Form = z.infer<typeof schema>

const aiSchema = z.object({
  apiKey: z.string().min(1),
  model: z.string().min(1),
})
type AIForm = z.infer<typeof aiSchema>

function AICard() {
  const settings = useSettings()
  const save = useSaveAI()
  const { register, handleSubmit, reset } = useForm<AIForm>({
    resolver: zodResolver(aiSchema),
    defaultValues: { model: 'claude-opus-4-8' },
  })
  const configured = settings.data?.aiConfigured

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-4">
        <CardTitle>AI (Claude)</CardTitle>
        {configured ? (
          <Badge className="bg-success/10 text-success border-success/30">
            AI connected ({settings.data?.aiModel || 'claude-opus-4-8'})
          </Badge>
        ) : (
          <Badge variant="secondary">AI not configured</Badge>
        )}
      </CardHeader>
      <CardContent>
        <form
          onSubmit={handleSubmit((d) =>
            save.mutate(d, {
              onSuccess: () => {
                reset({ apiKey: '', model: d.model })
                toast.success('AI settings saved')
              },
              onError: () => toast.error('Could not save'),
            }),
          )}
          className="space-y-4"
        >
          <div className="space-y-1">
            <Label htmlFor="ai-key">Anthropic API Key</Label>
            <Input id="ai-key" type="password" {...register('apiKey')} />
            <p className="text-xs text-muted-foreground">Stored encrypted; never displayed.</p>
          </div>
          <div className="space-y-1">
            <Label htmlFor="ai-model">Model</Label>
            <select
              id="ai-model"
              {...register('model')}
              className="w-full border rounded-lg p-2 text-sm bg-background text-foreground"
            >
              <option value="claude-opus-4-8">claude-opus-4-8 (best quality)</option>
              <option value="claude-haiku-4-5">claude-haiku-4-5 (fastest/cheapest)</option>
            </select>
          </div>
          <Button type="submit" disabled={save.isPending}>
            {save.isPending ? 'Saving…' : 'Save AI settings'}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}

function SESCard() {
  const settings = useSettings()
  const save = useSaveSES()
  const [editing, setEditing] = useState(false)
  const { register, handleSubmit, reset, formState: { errors } } = useForm<Form>({ resolver: zodResolver(schema) })
  const configured = settings.data?.sesConfigured

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-4">
        <CardTitle>Amazon SES</CardTitle>
        {settings.isLoading ? (
          <Skeleton className="h-5 w-32" />
        ) : configured ? (
          <Badge className="bg-success/10 text-success border-success/30">
            SES connected ({settings.data?.sesRegion})
          </Badge>
        ) : (
          <Badge variant="secondary">SES not configured</Badge>
        )}
      </CardHeader>
      <CardContent>
        {settings.isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
          </div>
        ) : configured && !editing ? (
          <div className="space-y-3">
            <div className="flex items-center gap-2 text-success">
              <CheckCircle2 className="h-4 w-4" />
              <span className="font-medium">SES is connected</span>
              <span className="text-muted-foreground text-sm">({settings.data?.sesRegion})</span>
            </div>
            <p className="text-xs text-muted-foreground">
              Your credentials are stored encrypted and are not displayed.
            </p>
            <Button variant="outline" onClick={() => setEditing(true)}>
              Replace credentials
            </Button>
          </div>
        ) : (
          <div className="space-y-4">
            {!configured && (
              <p className="text-sm text-muted-foreground">
                Connect your AWS SES account to start sending.
              </p>
            )}
            <form
              onSubmit={handleSubmit((d) =>
                save.mutate(d, {
                  onSuccess: () => {
                    reset()
                    setEditing(false)
                    toast.success('SES credentials saved')
                  },
                  onError: () => toast.error('Could not save'),
                })
              )}
              className="space-y-4"
            >
              <div className="space-y-1">
                <Label htmlFor="ak">AWS Access Key ID</Label>
                <Input id="ak" {...register('accessKeyId')} />
                {errors.accessKeyId && <p className="text-sm text-destructive">Required</p>}
              </div>
              <div className="space-y-1">
                <Label htmlFor="sk">AWS Secret Access Key</Label>
                <Input id="sk" type="password" {...register('secretAccessKey')} />
                {errors.secretAccessKey && <p className="text-sm text-destructive">Required</p>}
                <p className="text-xs text-muted-foreground">Stored encrypted; never displayed.</p>
              </div>
              <div className="space-y-1">
                <Label htmlFor="rg">Region</Label>
                <Input id="rg" placeholder="us-east-1" {...register('region')} />
                {errors.region && <p className="text-sm text-destructive">Required</p>}
              </div>
              {save.isError && <p className="text-sm text-destructive">Could not save credentials</p>}
              <div className="flex gap-2">
                <Button type="submit" disabled={save.isPending}>Save SES credentials</Button>
                {configured && editing && (
                  <Button type="button" variant="outline" onClick={() => { setEditing(false); reset() }}>
                    Cancel
                  </Button>
                )}
              </div>
            </form>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function WorkspaceCard() {
  const me = useMe()
  const rename = useRenameWorkspace()
  const [wsName, setWsName] = useState('')
  const isAdmin = me.data?.role === 'owner' || me.data?.role === 'admin'

  useEffect(() => {
    if (me.data?.workspaceName != null) setWsName(me.data.workspaceName)
  }, [me.data?.workspaceName])

  return (
    <Card>
      <CardHeader><CardTitle>Workspace</CardTitle></CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-1">
            <Label htmlFor="ws-name">Workspace name</Label>
            <Input
              id="ws-name"
              value={wsName}
              onChange={(e) => setWsName(e.target.value)}
              placeholder="My workspace"
              disabled={!isAdmin}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="ws-tz">Timezone</Label>
            <Input id="ws-tz" defaultValue="Asia/Bangkok (GMT+7)" />
          </div>
        </div>
        {isAdmin && (
          <Button
            onClick={() =>
              rename.mutate(wsName, {
                onSuccess: () => toast.success('Workspace renamed'),
                onError: () => toast.error('Could not rename workspace'),
              })
            }
            disabled={rename.isPending || !wsName.trim()}
          >
            {rename.isPending ? 'Saving…' : 'Save'}
          </Button>
        )}
      </CardContent>
    </Card>
  )
}

function AppearanceCard() {
  const { theme, toggle } = useTheme()
  return (
    <Card>
      <CardHeader><CardTitle>Appearance</CardTitle></CardHeader>
      <CardContent className="flex items-center justify-between">
        <div>
          <div className="text-sm font-semibold">Theme</div>
          <div className="text-sm text-muted-foreground">Light or dark — your choice is remembered.</div>
        </div>
        <Button variant="outline" onClick={toggle} className="capitalize">{theme}</Button>
      </CardContent>
    </Card>
  )
}

function TeamPanel() {
  const me = useMe()
  const team = useTeam()
  const invites = useInvites()
  const sendInvite = useInvite()
  const revokeInvite = useRevokeInvite()
  const setMemberRole = useSetRole()
  const removeMember = useRemoveMember()

  const [open, setOpen] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteRole, setInviteRole] = useState('editor')
  const [acceptUrl, setAcceptUrl] = useState('')

  const isAdmin = me.data?.role === 'owner' || me.data?.role === 'admin'

  function handleClose() {
    setOpen(false)
    setInviteEmail('')
    setInviteRole('editor')
    setAcceptUrl('')
    sendInvite.reset()
  }

  function handleInviteSubmit(e: React.FormEvent) {
    e.preventDefault()
    sendInvite.mutate(
      { email: inviteEmail, role: inviteRole },
      { onSuccess: (data) => setAcceptUrl(data.acceptUrl) },
    )
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between gap-4">
          <CardTitle>Team members</CardTitle>
          {isAdmin && (
            <Dialog
              open={open}
              onOpenChange={(o) => {
                if (!o) handleClose()
                else setOpen(true)
              }}
            >
              <DialogTrigger render={<Button size="sm">Invite member</Button>} />
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Invite member</DialogTitle>
                </DialogHeader>
                {acceptUrl ? (
                  <div className="space-y-3">
                    <p className="text-sm">Invite sent! Share this link:</p>
                    <p className="font-mono text-xs break-all select-all">{acceptUrl}</p>
                    <Button onClick={handleClose}>Done</Button>
                  </div>
                ) : (
                  <form onSubmit={handleInviteSubmit} className="space-y-4">
                    <div className="space-y-1">
                      <Label htmlFor="invite-email">Email</Label>
                      <Input
                        id="invite-email"
                        type="email"
                        value={inviteEmail}
                        onChange={(e) => setInviteEmail(e.target.value)}
                        required
                      />
                    </div>
                    <div className="space-y-1">
                      <Label htmlFor="invite-role">Role</Label>
                      <select
                        id="invite-role"
                        value={inviteRole}
                        onChange={(e) => setInviteRole(e.target.value)}
                        className="w-full border rounded-lg p-2 text-sm bg-background text-foreground"
                      >
                        <option value="admin">admin</option>
                        <option value="editor">editor</option>
                        <option value="viewer">viewer</option>
                      </select>
                    </div>
                    {sendInvite.isError && (
                      <p className="text-sm text-destructive">Failed to send invite</p>
                    )}
                    <Button type="submit" disabled={sendInvite.isPending || !inviteEmail}>
                      {sendInvite.isPending ? 'Sending…' : 'Send invite'}
                    </Button>
                  </form>
                )}
              </DialogContent>
            </Dialog>
          )}
        </CardHeader>
        <CardContent className="p-0">
          {team.isLoading && (
            <div className="px-6 py-4 space-y-2">
              <Skeleton className="h-8 w-full" />
            </div>
          )}
          {team.data?.map((m, i) => (
            <div
              key={m.id}
              className={cn(
                'flex items-center gap-3 px-6 py-3',
                i < (team.data?.length ?? 0) - 1 && 'border-b',
              )}
            >
              <span className="flex size-9 items-center justify-center rounded-full bg-primary-weak text-xs font-bold text-primary-text">
                {(m.fullName || m.email).slice(0, 2).toUpperCase()}
              </span>
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-semibold">{m.fullName || m.email}</div>
                <div className="truncate font-mono text-xs text-muted-foreground">{m.email}</div>
              </div>
              <span
                className={cn(
                  'rounded-md px-2 py-0.5 text-xs font-semibold',
                  m.role === 'owner'
                    ? 'bg-primary-weak text-primary-text'
                    : 'bg-surface-2 text-muted-foreground',
                )}
              >
                {m.role}
              </span>
              {isAdmin && m.email !== me.data?.email && (
                <div className="flex items-center gap-2">
                  <select
                    value={m.role}
                    onChange={(e) => setMemberRole.mutate({ id: m.id, role: e.target.value })}
                    className="border rounded-md p-1 text-xs bg-background text-foreground"
                  >
                    <option value="admin">admin</option>
                    <option value="editor">editor</option>
                    <option value="viewer">viewer</option>
                  </select>
                  <Button
                    size="sm"
                    variant="destructive"
                    onClick={() => removeMember.mutate(m.id)}
                  >
                    Remove
                  </Button>
                </div>
              )}
            </div>
          ))}
        </CardContent>
      </Card>

      {isAdmin && (
        <Card>
          <CardHeader><CardTitle>Pending invites</CardTitle></CardHeader>
          <CardContent className="p-0">
            {invites.isLoading && (
              <div className="px-6 py-4">
                <Skeleton className="h-8 w-full" />
              </div>
            )}
            {!invites.isLoading && invites.data?.length === 0 && (
              <p className="px-6 py-4 text-sm text-muted-foreground">No pending invites.</p>
            )}
            {invites.data?.map((inv, i) => (
              <div
                key={inv.id}
                className={cn(
                  'flex items-center gap-3 px-6 py-3',
                  i < (invites.data?.length ?? 0) - 1 && 'border-b',
                )}
              >
                <div className="min-w-0 flex-1">
                  <div className="truncate text-sm font-semibold">{inv.email}</div>
                  <div className="text-xs text-muted-foreground">{inv.role}</div>
                </div>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => revokeInvite.mutate(inv.id)}
                >
                  Revoke
                </Button>
              </div>
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function ComingSoonPanel({ title }: { title: string }) {
  return (
    <Card>
      <CardContent className="py-10 text-center text-sm text-muted-foreground">
        {title} settings are coming soon.
      </CardContent>
    </Card>
  )
}

const TABS = ['General', 'Team', 'API & webhooks', 'Hooks', 'Billing'] as const
type Tab = (typeof TABS)[number]

export function SettingsPage() {
  const [tab, setTab] = useState<Tab>('General')

  return (
    <div className="space-y-5">
      <h1 className="text-2xl font-bold tracking-tight">Settings</h1>

      <div className="flex flex-wrap gap-1.5 border-b">
        {TABS.map((t) => (
          <button key={t} type="button" onClick={() => setTab(t)}
            className={cn('mr-3.5 border-b-2 px-1 py-2.5 text-sm font-medium',
              tab === t ? 'border-primary font-semibold text-primary-text' : 'border-transparent text-muted-foreground')}>
            {t}
          </button>
        ))}
      </div>

      {tab === 'General' && (
        <div className="space-y-4">
          <WorkspaceCard />
          <AppearanceCard />
          <SESCard />
          <AICard />
          <Card>
            <CardContent className="flex items-start gap-3 py-4 text-sm text-muted-foreground">
              <span className="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-lg bg-[linear-gradient(135deg,var(--primary),var(--purple))] text-white">
                <Sparkles className="size-4" aria-hidden="true" />
              </span>
              <p>Bring your own key — Plume never sees your prompts or subscriber data. Your dev tooling and subscriptions stay on your side; the only cost shown to a product owner is pure SES sending.</p>
            </CardContent>
          </Card>
        </div>
      )}
      {tab === 'Team' && <TeamPanel />}
      {tab === 'API & webhooks' && <ComingSoonPanel title="API & webhooks" />}
      {tab === 'Hooks' && <ComingSoonPanel title="Hooks" />}
      {tab === 'Billing' && <ComingSoonPanel title="Billing" />}
    </div>
  )
}
