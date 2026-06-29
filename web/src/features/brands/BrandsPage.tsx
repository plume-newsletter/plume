import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Building2, Plus } from 'lucide-react'
import { toast } from 'sonner'
import {
  useBrands, useCreateBrand, useUpdateBrand, type Brand, type BrandInput,
} from './useBrands'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/EmptyState'

const schema = z.object({
  name: z.string().min(1),
  fromName: z.string().min(1),
  fromEmail: z.string().email(),
  replyTo: z.string().email().or(z.literal('')),
})

function BrandDialog({ brand, trigger }: { brand?: Brand; trigger: React.ReactElement }) {
  const [open, setOpen] = useState(false)
  const create = useCreateBrand()
  const update = useUpdateBrand()
  const { register, handleSubmit, reset, formState: { errors } } = useForm<BrandInput>({
    resolver: zodResolver(schema),
    defaultValues: { name: '', fromName: '', fromEmail: '', replyTo: '' },
    values: brand
      ? { name: brand.name, fromName: brand.from_name, fromEmail: brand.from_email, replyTo: brand.reply_to }
      : undefined,
  })
  const onSubmit = (data: BrandInput) => {
    const close = () => { setOpen(false); reset() }
    if (brand) {
      update.mutate({ id: brand.id, ...data }, {
        onSuccess: () => { close(); toast.success('Brand saved') },
        onError: () => toast.error('Something went wrong'),
      })
    } else {
      create.mutate(data, {
        onSuccess: () => { close(); toast.success('Brand created') },
        onError: () => toast.error('Something went wrong'),
      })
    }
  }
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={trigger} />
      <DialogContent>
        <DialogHeader><DialogTitle>{brand ? 'Edit brand' : 'New brand'}</DialogTitle></DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-3">
          <div className="space-y-1.5">
            <Label htmlFor="name">Name</Label>
            <Input id="name" {...register('name')} />
            {errors.name && <p className="text-sm text-destructive">Required</p>}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="fromName">From name</Label>
            <Input id="fromName" {...register('fromName')} />
            {errors.fromName && <p className="text-sm text-destructive">Required</p>}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="fromEmail">From email</Label>
            <Input id="fromEmail" {...register('fromEmail')} />
            {errors.fromEmail && <p className="text-sm text-destructive">Valid email required</p>}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="replyTo">Reply-to</Label>
            <Input id="replyTo" {...register('replyTo')} />
          </div>
          <DialogFooter>
            <Button type="submit">{brand ? 'Save' : 'Create'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

// ponytail: DKIM/SPF/DMARC status and list/sub counts are sample until domain-auth
// + audience counts are tracked; brand identity (name/from) is real.
const AUTH = ['DKIM', 'SPF', 'DMARC']
const GRADIENTS = [
  'from-[#1E40AF] to-[#3b6fd4]', 'from-[#D97706] to-[#f0a83c]',
  'from-[#7c3aed] to-[#a78bfa]', 'from-[#0d9488] to-[#2dd4bf]',
]

export function BrandsPage() {
  const { data: brands, isLoading } = useBrands()

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Brands</h1>
          <p className="mt-1 text-muted-foreground">Sender identities — each with its own from-address, domain, and signature.</p>
        </div>
        <BrandDialog trigger={<Button className="gap-1.5"><Plus className="size-4" aria-hidden="true" />New brand</Button>} />
      </div>

      {isLoading ? (
        <div className="grid gap-4 [grid-template-columns:repeat(auto-fill,minmax(290px,1fr))]">
          {[0, 1, 2].map((i) => <Skeleton key={i} className="h-40 rounded-2xl" />)}
        </div>
      ) : !brands?.length ? (
        <EmptyState icon={Building2} title="No brands yet"
          description="Create your first sender identity to start sending."
          action={<BrandDialog trigger={<Button>New brand</Button>} />} />
      ) : (
        <div className="grid gap-4 [grid-template-columns:repeat(auto-fill,minmax(290px,1fr))]">
          {brands.map((b, i) => (
            <div key={b.id} className="rounded-2xl border bg-card p-5 shadow-[var(--shadow-sm)]">
              <div className="mb-3.5 flex items-center gap-3">
                <span className={`flex size-11 items-center justify-center rounded-xl bg-gradient-to-br ${GRADIENTS[i % GRADIENTS.length]} text-lg font-bold text-white`}>
                  {b.name.slice(0, 1).toUpperCase()}
                </span>
                <div className="min-w-0">
                  <div className="truncate font-bold">{b.name}</div>
                  <div className="truncate font-mono text-xs text-muted-foreground">{b.from_email}</div>
                </div>
              </div>
              <div className="mb-3.5 flex flex-wrap gap-1.5">
                {AUTH.map((a) => (
                  <span key={a} className="rounded-md bg-success-weak px-2 py-0.5 text-xs font-semibold text-success">✓ {a}</span>
                ))}
              </div>
              <div className="flex items-center justify-between text-sm text-muted-foreground">
                <span>— lists · — subs</span>
                <BrandDialog brand={b} trigger={
                  <button type="button" className="font-semibold text-primary-text hover:underline">Manage</button>
                } />
              </div>
            </div>
          ))}
          <BrandDialog trigger={
            <button type="button" className="flex min-h-40 flex-col items-center justify-center gap-2 rounded-2xl border-[1.5px] border-dashed border-border-strong bg-card text-muted-foreground hover:border-primary hover:text-primary-text">
              <Plus className="size-6" aria-hidden="true" />
              <span className="text-sm font-semibold">Add a brand</span>
            </button>
          } />
        </div>
      )}
    </div>
  )
}
