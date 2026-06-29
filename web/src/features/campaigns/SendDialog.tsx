import { useState } from 'react'
import { toast } from 'sonner'
import { useLists } from '@/features/lists/useLists'
import { useSendCampaign } from './useSendCampaign'
import { ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Label } from '@/components/ui/label'

export function SendDialog({
  campaignId,
  disabled,
  label = 'Send',
  triggerClassName = 'bg-brand-accent text-brand-accent-foreground hover:bg-brand-accent/80',
  icon,
}: {
  campaignId: string
  disabled: boolean
  label?: string
  triggerClassName?: string
  icon?: React.ReactNode
}) {
  const [open, setOpen] = useState(false)
  const [listId, setListId] = useState('')
  const { data: lists } = useLists()
  const send = useSendCampaign(campaignId)

  function handleClose() {
    setOpen(false)
    setListId('')
    send.reset()
  }

  function handleSend() {
    if (!listId) return
    send.mutate(listId, {
      onSuccess: (data) => {
        toast.success(`Queued ${data.recipients} recipients`)
      },
    })
  }

  function errorMessage(): string {
    if (!send.error) return ''
    if (send.error instanceof ApiError && send.error.status === 409) {
      return 'This campaign has already been sent or queued.'
    }
    return 'Something went wrong. Please try again.'
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) handleClose()
        else setOpen(true)
      }}
    >
      <DialogTrigger
        render={
          <Button disabled={disabled} className={triggerClassName}>
            {icon}
            {label}
          </Button>
        }
      />
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Send campaign</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <div>
            <Label htmlFor="send-list">Recipient list</Label>
            <Select value={listId} onValueChange={(v) => setListId(v ?? '')}>
              <SelectTrigger id="send-list" className="w-full">
                <SelectValue placeholder="Select a list">
                  {(v) => lists?.find((l) => l.id === v)?.name ?? 'Select a list'}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                {lists?.map((l) => (
                  <SelectItem key={l.id} value={l.id}>
                    {l.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {send.isSuccess && send.data && (
            <p className="text-sm text-success">
              Queued {send.data.recipients} recipients.
            </p>
          )}
          {send.isError && (
            <p className="text-sm text-destructive">{errorMessage()}</p>
          )}
        </div>
        <DialogFooter>
          <Button
            onClick={handleSend}
            disabled={!listId || send.isPending}
            className="bg-brand-accent text-brand-accent-foreground hover:bg-brand-accent/80"
          >
            {send.isPending ? 'Sending…' : 'Send'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
