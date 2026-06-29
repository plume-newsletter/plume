import { Sparkles } from 'lucide-react'

export function ComingSoon({ title, description }: { title: string; description: string }) {
  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center text-center">
      <div className="mb-5 flex size-14 items-center justify-center rounded-2xl bg-[linear-gradient(135deg,var(--primary),var(--purple))] text-white shadow-[var(--shadow-sm)]">
        <Sparkles className="size-6" aria-hidden="true" />
      </div>
      <h1 className="text-2xl font-bold tracking-tight">{title}</h1>
      <p className="mt-2 max-w-md text-sm text-muted-foreground">{description}</p>
      <span className="mt-5 inline-flex items-center rounded-full border border-primary-weak bg-primary-weak px-3 py-1 text-xs font-semibold text-primary-text">
        Coming soon
      </span>
    </div>
  )
}
