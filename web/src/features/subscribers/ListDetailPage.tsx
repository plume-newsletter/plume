import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Users, ChevronLeft } from 'lucide-react'
import { toast } from 'sonner'
import { useLists } from '@/features/lists/useLists'
import {
  useSubscribers,
  useAddSubscriber,
  useSetSubscriberStatus,
  useDeleteSubscriber,
  type Subscriber,
} from './useSubscribers'
import { SubscriberStatusBadge } from './SubscriberStatusBadge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { ConfirmDialog } from '@/components/ConfirmDialog'
import { PageHeader } from '@/components/PageHeader'
import { EmptyState } from '@/components/EmptyState'
import { ImportDialog } from './ImportDialog'

const STATUS_OPTIONS = ['active', 'unsubscribed', 'bounced', 'complained', 'pending'] as const

const addSchema = z.object({
  email: z.string().min(1, 'Required').email('Valid email required'),
  name: z.string().optional(),
})
type AddFormValues = z.infer<typeof addSchema>

function AddSubscriberDialog({ listId }: { listId: string }) {
  const [open, setOpen] = useState(false)
  const add = useAddSubscriber(listId)
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<AddFormValues>({
    resolver: zodResolver(addSchema),
    defaultValues: { email: '', name: '' },
  })
  const onSubmit = (data: AddFormValues) => {
    add.mutate(
      { email: data.email, name: data.name ?? '' },
      {
        onSuccess: () => {
          setOpen(false)
          reset()
          // ponytail: duplicate-vs-created not distinguished — the add API returns a bare subscriber (created/duplicate differ only by 201/200 status, which api() discards). To show "Already on the list", make the endpoint return { subscriber, created } and surface it. Deferred.
          toast.success('Subscriber added')
        },
        onError: () => toast.error('Something went wrong'),
      },
    )
  }
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      {/* @base-ui DialogTrigger uses render prop instead of asChild */}
      <DialogTrigger render={<Button>Add subscriber</Button>} />
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add subscriber</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-3">
          <div>
            <Label htmlFor="sub-email">Email</Label>
            <Input id="sub-email" type="email" {...register('email')} />
            {errors.email && <p className="text-sm text-destructive">{errors.email.message}</p>}
          </div>
          <div>
            <Label htmlFor="sub-name">Name</Label>
            <Input id="sub-name" {...register('name')} />
          </div>
          <DialogFooter>
            <Button type="submit" disabled={add.isPending}>Add</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function StatusSelect({ subscriber, listId }: { subscriber: Subscriber; listId: string }) {
  const setStatus = useSetSubscriberStatus(listId)
  return (
    <Select
      value={subscriber.status}
      onValueChange={(v) => {
        const status = v ?? ''
        if (status && status !== subscriber.status) {
          setStatus.mutate(
            { id: subscriber.id, status },
            {
              onSuccess: () => toast.success('Status updated'),
              onError: () => toast.error('Something went wrong'),
            },
          )
        }
      }}
    >
      <SelectTrigger size="sm" className="w-36">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {STATUS_OPTIONS.map((s) => (
          <SelectItem key={s} value={s}>{s}</SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

export function ListDetailPage() {
  const { id } = useParams<{ id: string }>()
  const listId = id ?? ''

  const { data: lists } = useLists()
  const { data: subscribers, isLoading } = useSubscribers(listId)
  const del = useDeleteSubscriber(listId)

  const list = lists?.find((l) => l.id === listId)
  const listName = list?.name ?? 'List'

  return (
    <div className="space-y-4">
      <Link
        to="/lists"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ChevronLeft className="size-4" aria-hidden />
        Lists
      </Link>
      <PageHeader
        title={listName}
        description="Manage subscribers, import, and status."
        actions={
          <>
            <AddSubscriberDialog listId={listId} />
            <ImportDialog listId={listId} />
          </>
        }
      />
      {subscribers && (
        <p className="text-sm text-muted-foreground">
          {subscribers.length} {subscribers.length === 1 ? 'subscriber' : 'subscribers'}
        </p>
      )}
      {isLoading ? (
        <Card>
          <CardContent className="space-y-3 py-4">
            {[0, 1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-8 w-full" />
            ))}
          </CardContent>
        </Card>
      ) : !subscribers?.length ? (
        <EmptyState
          icon={Users}
          title="No subscribers yet"
          description="Add one manually or import a CSV."
          action={<AddSubscriberDialog listId={listId} />}
        />
      ) : (
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Email</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {subscribers.map((s) => (
                  <TableRow key={s.id} className="hover:bg-muted/50">
                    <TableCell>{s.email}</TableCell>
                    <TableCell className={s.name ? '' : 'text-muted-foreground'}>
                      {s.name || '—'}
                    </TableCell>
                    <TableCell>
                      <SubscriberStatusBadge status={s.status} />
                    </TableCell>
                    <TableCell className="text-right space-x-2">
                      <StatusSelect subscriber={s} listId={listId} />
                      <ConfirmDialog
                        title={`Delete ${s.email}?`}
                        trigger={<Button variant="destructive" size="sm">Delete</Button>}
                        onConfirm={() =>
                          del.mutate(s.id, {
                            onSuccess: () => toast.success('Subscriber removed'),
                            onError: () => toast.error('Something went wrong'),
                          })
                        }
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
