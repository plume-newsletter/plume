import { useState } from 'react'
import { toast } from 'sonner'
import { Copy, Plus, Trash2, KeyRound, Webhook as WebhookIcon } from 'lucide-react'
import {
  useApiKeys, useCreateApiKey, useDeleteApiKey,
  useWebhooks, useCreateWebhook, useDeleteWebhook,
} from './useApiWebhooks'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger,
} from '@/components/ui/dialog'

function copy(text: string) {
  navigator.clipboard?.writeText(text).then(
    () => toast.success('Copied'),
    () => toast.error('Could not copy'),
  )
}

function ApiKeysCard() {
  const { data: keys } = useApiKeys()
  const create = useCreateApiKey()
  const del = useDeleteApiKey()
  const [name, setName] = useState('')
  const [open, setOpen] = useState(false)
  const [secret, setSecret] = useState<string | null>(null)

  function onCreate() {
    if (!name.trim()) return
    create.mutate(name.trim(), {
      onSuccess: (r) => { setSecret(r.secret); setName(''); setOpen(false); toast.success('API key created') },
      onError: () => toast.error('Could not create key'),
    })
  }

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between gap-2">
        <CardTitle className="flex items-center gap-2"><KeyRound className="size-4" aria-hidden="true" /> API keys</CardTitle>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger render={<Button size="sm" className="gap-1.5"><Plus className="size-3.5" aria-hidden="true" /> New key</Button>} />
          <DialogContent>
            <DialogHeader><DialogTitle>Create API key</DialogTitle></DialogHeader>
            <div className="space-y-1.5">
              <Label htmlFor="key-name">Name</Label>
              <Input id="key-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Zapier integration"
                onKeyDown={(e) => { if (e.key === 'Enter') onCreate() }} />
            </div>
            <Button onClick={onCreate} disabled={!name.trim() || create.isPending} className="mt-3 w-full">
              {create.isPending ? 'Creating…' : 'Create key'}
            </Button>
          </DialogContent>
        </Dialog>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">
          Use a key as a Bearer token to call the Plume API: <code className="rounded bg-surface-2 px-1 py-0.5 text-xs">Authorization: Bearer &lt;key&gt;</code>
        </p>

        {secret && (
          <div className="rounded-xl border border-success bg-success-weak p-3">
            <div className="mb-1 text-sm font-semibold text-success">Copy your key now — it won't be shown again.</div>
            <div className="flex items-center gap-2">
              <code className="flex-1 truncate rounded bg-card px-2 py-1.5 font-mono text-xs">{secret}</code>
              <Button size="sm" variant="outline" className="gap-1.5" onClick={() => copy(secret)}><Copy className="size-3.5" aria-hidden="true" /> Copy</Button>
            </div>
          </div>
        )}

        {keys && keys.length === 0 && <p className="text-sm text-muted-foreground">No API keys yet.</p>}
        <div className="flex flex-col divide-y">
          {keys?.map((k) => (
            <div key={k.id} className="flex items-center justify-between gap-3 py-2.5">
              <div className="min-w-0">
                <div className="truncate text-sm font-medium">{k.name}</div>
                <div className="text-xs text-muted-foreground">
                  <span className="font-mono">{k.prefix}…</span> · created {k.createdAt} · {k.lastUsedAt ? `last used ${k.lastUsedAt}` : 'never used'}
                </div>
              </div>
              <Button size="sm" variant="ghost" aria-label={`Revoke ${k.name}`} className="text-danger hover:text-danger"
                onClick={() => del.mutate(k.id, { onSuccess: () => toast.success('Key revoked') })}>
                <Trash2 className="size-4" aria-hidden="true" />
              </Button>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function WebhooksCard() {
  const { data } = useWebhooks()
  const create = useCreateWebhook()
  const del = useDeleteWebhook()
  const [url, setUrl] = useState('')
  const [events, setEvents] = useState<string[]>([])

  const allEvents = data?.events ?? []
  const toggle = (e: string) => setEvents((cur) => (cur.includes(e) ? cur.filter((x) => x !== e) : [...cur, e]))

  function onCreate() {
    if (!url.trim() || events.length === 0) return
    create.mutate({ url: url.trim(), events }, {
      onSuccess: () => { setUrl(''); setEvents([]); toast.success('Endpoint added') },
      onError: () => toast.error('Could not add endpoint — check the URL and events'),
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2"><WebhookIcon className="size-4" aria-hidden="true" /> Webhooks</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground">
          Plume POSTs a signed JSON payload to your endpoint when events occur. Verify the
          <code className="mx-1 rounded bg-surface-2 px-1 py-0.5 text-xs">X-Plume-Signature</code>
          header (HMAC-SHA256 of the body using the endpoint secret).
        </p>

        <div className="rounded-xl border p-3">
          <Label htmlFor="wh-url">Endpoint URL</Label>
          <Input id="wh-url" value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://example.com/hooks/plume" className="mt-1" />
          <div className="mt-3 flex flex-wrap gap-3">
            {allEvents.map((e) => (
              <label key={e} className="flex items-center gap-1.5 text-sm">
                <input type="checkbox" checked={events.includes(e)} onChange={() => toggle(e)} />
                <span className="font-mono text-xs">{e}</span>
              </label>
            ))}
          </div>
          <Button onClick={onCreate} disabled={!url.trim() || events.length === 0 || create.isPending} className="mt-3 gap-1.5">
            <Plus className="size-3.5" aria-hidden="true" /> {create.isPending ? 'Adding…' : 'Add endpoint'}
          </Button>
        </div>

        {data && data.endpoints.length === 0 && <p className="text-sm text-muted-foreground">No endpoints yet.</p>}
        <div className="flex flex-col gap-3">
          {data?.endpoints.map((wh) => (
            <div key={wh.id} className="rounded-xl border p-3">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="truncate font-mono text-sm">{wh.url}</div>
                  <div className="mt-1 flex flex-wrap gap-1.5">
                    {wh.events.map((e) => (
                      <span key={e} className="rounded-md bg-surface-2 px-1.5 py-0.5 font-mono text-xs text-muted-foreground">{e}</span>
                    ))}
                  </div>
                </div>
                <Button size="sm" variant="ghost" aria-label="Delete endpoint" className="text-danger hover:text-danger"
                  onClick={() => del.mutate(wh.id, { onSuccess: () => toast.success('Endpoint deleted') })}>
                  <Trash2 className="size-4" aria-hidden="true" />
                </Button>
              </div>
              <div className="mt-2 flex items-center gap-2">
                <span className="text-xs text-muted-foreground">Secret</span>
                <code className="flex-1 truncate rounded bg-surface-2 px-2 py-1 font-mono text-xs">{wh.secret}</code>
                <Button size="sm" variant="outline" className="gap-1.5" onClick={() => copy(wh.secret)}><Copy className="size-3.5" aria-hidden="true" /> Copy</Button>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

export function ApiWebhooksPanel() {
  return (
    <div className="space-y-4">
      <ApiKeysCard />
      <WebhooksCard />
    </div>
  )
}
