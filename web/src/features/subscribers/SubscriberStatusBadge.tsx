import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

type Status = 'active' | 'pending' | 'unsubscribed' | 'bounced' | 'complained' | string

export function SubscriberStatusBadge({ status }: { status: Status }) {
  if (status === 'active') {
    return (
      <Badge
        className={cn(
          'bg-emerald-600/15 text-emerald-700 dark:text-emerald-400',
          'border-transparent',
        )}
      >
        {status}
      </Badge>
    )
  }
  if (status === 'bounced') {
    return (
      <Badge
        className={cn(
          'bg-amber-500/15 text-amber-700 dark:text-amber-400',
          'border-transparent',
        )}
      >
        {status}
      </Badge>
    )
  }
  if (status === 'complained') {
    return <Badge variant="destructive">{status}</Badge>
  }
  if (status === 'unsubscribed') {
    return <Badge variant="outline">{status}</Badge>
  }
  // pending or unknown
  return <Badge variant="secondary">{status}</Badge>
}
