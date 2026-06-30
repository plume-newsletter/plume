import { useState } from 'react'
import { Send } from 'lucide-react'
import { toast } from 'sonner'
import { useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { useMe } from '@/features/auth/useAuth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'

function useTestSend(id: string) {
  return useMutation({
    mutationFn: (email: string) =>
      api<{ sent: boolean }>(`/api/campaigns/${id}/test`, {
        method: 'POST',
        body: JSON.stringify({ email }),
      }),
  })
}

export function TestSendDialog({ campaignId }: { campaignId: string }) {
  const { data: me } = useMe()
  const [open, setOpen] = useState(false)
  const [email, setEmail] = useState('')
  const test = useTestSend(campaignId)

  // Default to the current user's email when opening.
  function onOpenChange(o: boolean) {
    setOpen(o)
    if (o) setEmail((e) => e || me?.email || '')
    else test.reset()
  }

  function handleSend() {
    const addr = email.trim()
    if (!addr) return
    test.mutate(addr, {
      onSuccess: () => {
        toast.success(`Test sent to ${addr}`)
        setOpen(false)
      },
      onError: () => toast.error('Could not send test. Check your SES settings.'),
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogTrigger
        render={
          <Button type="button" variant="outline" className="gap-1.5">
            <Send className="size-3.5" aria-hidden="true" /> Send test
          </Button>
        }
      />
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Send a test email</DialogTitle>
        </DialogHeader>
        <div className="space-y-1.5">
          <Label htmlFor="test-email">Send to</Label>
          <Input
            id="test-email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
            onKeyDown={(e) => { if (e.key === 'Enter') handleSend() }}
          />
          <p className="text-xs text-muted-foreground">
            Sends the current draft as-is. No tracking, unsubscribe, or open pixel is added.
          </p>
        </div>
        <DialogFooter>
          <Button onClick={handleSend} disabled={!email.trim() || test.isPending} className="gap-1.5">
            <Send className="size-3.5" aria-hidden="true" />
            {test.isPending ? 'Sending…' : 'Send test'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
