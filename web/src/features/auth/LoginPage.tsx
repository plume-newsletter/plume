import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useNavigate } from 'react-router-dom'
import { Feather } from 'lucide-react'

// lucide-react (installed version) doesn't export a GitHub mark; inline the logo.
function GithubMark({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M12 .5C5.73.5.5 5.74.5 12.02c0 5.1 3.29 9.42 7.86 10.95.58.1.79-.25.79-.56v-2c-3.2.7-3.88-1.37-3.88-1.37-.53-1.34-1.3-1.7-1.3-1.7-1.06-.72.08-.71.08-.71 1.17.08 1.79 1.2 1.79 1.2 1.04 1.79 2.73 1.27 3.4.97.1-.76.41-1.27.74-1.56-2.55-.29-5.23-1.28-5.23-5.7 0-1.26.45-2.29 1.19-3.1-.12-.29-.52-1.46.11-3.05 0 0 .97-.31 3.18 1.18a11 11 0 0 1 5.8 0c2.2-1.49 3.17-1.18 3.17-1.18.63 1.59.23 2.76.11 3.05.74.81 1.19 1.84 1.19 3.1 0 4.43-2.69 5.41-5.25 5.69.42.37.8 1.1.8 2.22v3.29c0 .31.21.67.8.56A11.53 11.53 0 0 0 23.5 12.02C23.5 5.74 18.27.5 12 .5Z" />
    </svg>
  )
}
import { useLogin } from './useAuth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const schema = z.object({ email: z.string().email(), password: z.string().min(1) })
type Form = z.infer<typeof schema>

const FEATURES = [
  'Drag-and-drop email builder + AI copy',
  'Visual automations & segmentation',
  'Real-time analytics & deliverability health',
]

export function LoginPage() {
  const nav = useNavigate()
  const login = useLogin()
  const { register, handleSubmit, formState: { errors } } = useForm<Form>({ resolver: zodResolver(schema) })
  const onSubmit = (data: Form) => login.mutate(data, { onSuccess: () => nav('/', { replace: true }) })

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
            The newsletter platform you actually own.
          </h2>
          <p className="mt-3 text-muted-foreground">
            Self-hosted. AI-native. Sends through your own Amazon SES at AWS cost — not per subscriber.
          </p>
          <ul className="mt-6 space-y-2 text-sm">
            {FEATURES.map((f) => (
              <li key={f} className="flex items-center gap-2">
                <span className="size-1.5 rounded-full bg-primary" aria-hidden="true" />
                {f}
              </li>
            ))}
          </ul>
        </div>
        <p className="font-mono text-xs text-faint">$ docker compose up -d &nbsp;·&nbsp; AGPL-3.0</p>
      </div>

      {/* Right: sign-in card */}
      <div className="flex items-center justify-center p-6">
        <div className="w-full max-w-sm">
          <div className="mb-6 flex items-center gap-2 lg:hidden">
            <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <Feather className="size-4" aria-hidden="true" />
            </span>
            <span className="text-lg font-semibold">Plume</span>
          </div>
          <h1 className="text-2xl font-semibold">Welcome back</h1>
          <p className="mt-1 text-sm text-muted-foreground">Sign in to your self-hosted workspace.</p>

          <form onSubmit={handleSubmit(onSubmit)} className="mt-6 space-y-4">
            <div className="space-y-1">
              <Label htmlFor="email">Email</Label>
              <Input id="email" type="email" {...register('email')} />
              {errors.email && <p className="text-sm text-destructive">Enter a valid email</p>}
            </div>
            <div className="space-y-1">
              <Label htmlFor="password">Password</Label>
              <Input id="password" type="password" {...register('password')} />
            </div>
            <div className="flex items-center justify-between text-sm">
              <label className="flex items-center gap-2 text-muted-foreground">
                <input type="checkbox" className="rounded border" /> Remember me
              </label>
              <span className="text-primary-text">Forgot password?</span>
            </div>
            {login.isError && <p className="text-sm text-destructive">Invalid credentials</p>}
            <Button type="submit" className="w-full" disabled={login.isPending}>
              {login.isPending ? 'Signing in…' : 'Sign in'}
            </Button>
          </form>

          <div className="my-5 flex items-center gap-3 text-xs text-faint">
            <span className="h-px flex-1 bg-border" /> OR <span className="h-px flex-1 bg-border" />
          </div>
          <Button type="button" variant="outline" className="w-full gap-2">
            <GithubMark className="size-4" />
            Continue with GitHub
          </Button>
          <p className="mt-6 text-center text-xs text-muted-foreground">
            Self-hosted · your data never leaves your server
          </p>
        </div>
      </div>
    </div>
  )
}
