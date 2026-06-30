import { createContext, useContext, useState, type ReactNode } from 'react'
import { Sparkles, Send, X } from 'lucide-react'
import { toast } from 'sonner'
import { useAiChat, type ChatMessage } from '@/features/ai/useAiChat'
import { cn } from '@/lib/utils'

type Ctx = { open: boolean; toggle: () => void; setOpen: (v: boolean) => void }
const AiCtx = createContext<Ctx | null>(null)

export function useAiAssistant() {
  const c = useContext(AiCtx)
  if (!c) throw new Error('useAiAssistant must be used within AiAssistantProvider')
  return c
}

const TRY_PROMPTS = [
  '✍️ Write a re-engagement email for cold subscribers',
  '🎯 Build a segment: opened in last 30d but never clicked',
  "📊 Summarize my last campaign's performance",
  '🔁 Draft a 4-email welcome automation',
]

export function AiAssistantProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const chat = useAiChat()

  const send = (text: string) => {
    const trimmed = text.trim()
    if (!trimmed || chat.isPending) return
    const next: ChatMessage[] = [...messages, { role: 'user', content: trimmed }]
    setMessages(next)
    setInput('')
    chat.mutate(next, {
      onSuccess: ({ reply }) =>
        setMessages((m) => [...m, { role: 'assistant', content: reply }]),
      onError: () =>
        setMessages((m) => [
          ...m,
          { role: 'assistant', content: 'Sorry — something went wrong. Check your AI key in Settings and try again.' },
        ]),
    })
  }

  return (
    <AiCtx.Provider value={{ open, toggle: () => setOpen((v) => !v), setOpen }}>
      {children}
      {open && (
        <AiPanel
          messages={messages}
          input={input}
          pending={chat.isPending}
          onInput={setInput}
          onSend={send}
          onClose={() => setOpen(false)}
        />
      )}
    </AiCtx.Provider>
  )
}

function AiPanel({
  messages, input, pending, onInput, onSend, onClose,
}: {
  messages: ChatMessage[]
  input: string
  pending: boolean
  onInput: (v: string) => void
  onSend: (text: string) => void
  onClose: () => void
}) {
  return (
    <>
      <div onClick={onClose} className="fixed inset-0 z-40 bg-black/40 backdrop-blur-[2px]" aria-hidden="true" />
      <aside className="fixed inset-y-0 right-0 z-50 flex w-[400px] max-w-[92vw] flex-col border-l bg-surface shadow-2xl">
        <header className="flex items-center gap-2.5 border-b px-4 py-3.5">
          <span className="flex size-7 items-center justify-center rounded-lg bg-[linear-gradient(135deg,var(--primary),var(--purple))] text-white">
            <Sparkles className="size-4" aria-hidden="true" />
          </span>
          <div className="flex-1">
            <div className="text-sm font-bold">Plume AI</div>
            <div className="flex items-center gap-1.5 text-xs text-success">
              <span className="size-1.5 rounded-full bg-success" /> Ready
            </div>
          </div>
          <button onClick={onClose} aria-label="Close" className="flex size-7 items-center justify-center rounded-lg text-muted-foreground hover:bg-surface-2">
            <X className="size-4" aria-hidden="true" />
          </button>
        </header>

        <div className="flex-1 space-y-3.5 overflow-y-auto p-4">
          <div className="max-w-[90%] rounded-xl rounded-tl-sm bg-surface-2 px-3.5 py-3 text-sm">
            Hi! I can help write campaigns, build segments, draft automations, or think through your last send. What do you need?
          </div>

          {messages.length === 0 && (
            <>
              <div className="text-xs font-semibold uppercase tracking-wide text-faint">Try</div>
              <div className="flex flex-col gap-2.5">
                {TRY_PROMPTS.map((p) => (
                  <button
                    key={p}
                    onClick={() => onSend(p)}
                    className="rounded-xl border px-3.5 py-3 text-left text-sm hover:border-primary hover:bg-primary-weak"
                  >
                    {p}
                  </button>
                ))}
              </div>
            </>
          )}

          {messages.map((m, i) => (
            <MessageBubble key={i} message={m} />
          ))}

          {pending && (
            <div className="max-w-[90%] rounded-xl rounded-tl-sm bg-surface-2 px-3.5 py-3 text-sm text-muted-foreground">
              <TypingDots />
            </div>
          )}
        </div>

        <form
          onSubmit={(e) => {
            e.preventDefault()
            onSend(input)
          }}
          className="border-t p-3.5"
        >
          <div className="flex items-center gap-2 rounded-xl border bg-background py-1.5 pl-3.5 pr-1.5 focus-within:border-primary focus-within:ring-2 focus-within:ring-ring">
            <input
              value={input}
              onChange={(e) => onInput(e.target.value)}
              placeholder="Ask Plume AI anything…"
              aria-label="Ask Plume AI"
              className="flex-1 bg-transparent text-sm outline-none"
            />
            <button
              type="submit"
              disabled={pending || !input.trim()}
              aria-label="Send"
              className="flex size-8 items-center justify-center rounded-lg bg-primary text-white disabled:opacity-50"
            >
              <Send className="size-4" aria-hidden="true" />
            </button>
          </div>
        </form>
      </aside>
    </>
  )
}

function MessageBubble({ message }: { message: ChatMessage }) {
  const isUser = message.role === 'user'
  return (
    <div className={cn('flex flex-col gap-1', isUser ? 'items-end' : 'items-start')}>
      <div
        className={cn(
          'max-w-[90%] whitespace-pre-wrap rounded-xl px-3.5 py-3 text-sm',
          isUser ? 'rounded-tr-sm bg-primary text-white' : 'rounded-tl-sm bg-surface-2',
        )}
      >
        {message.content}
      </div>
      {!isUser && (
        <button
          onClick={() => {
            navigator.clipboard?.writeText(message.content)
            toast.success('Copied')
          }}
          className="text-xs text-muted-foreground hover:text-foreground"
        >
          Copy
        </button>
      )}
    </div>
  )
}

function TypingDots() {
  return (
    <span className="inline-flex gap-1" aria-label="Plume AI is typing">
      <span className="size-1.5 animate-bounce rounded-full bg-muted-foreground [animation-delay:-0.3s]" />
      <span className="size-1.5 animate-bounce rounded-full bg-muted-foreground [animation-delay:-0.15s]" />
      <span className="size-1.5 animate-bounce rounded-full bg-muted-foreground" />
    </span>
  )
}
