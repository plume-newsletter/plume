import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from '@/lib/queryClient'
import { App } from '@/App'
import { ThemeProvider, useTheme } from '@/components/theme/ThemeProvider'
import { Toaster } from '@/components/ui/sonner'
import './index.css'

function ThemedToaster() {
  const { theme } = useTheme()
  return <Toaster theme={theme} richColors position="top-right" />
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <App />
        <ThemedToaster />
      </QueryClientProvider>
    </ThemeProvider>
  </StrictMode>,
)
