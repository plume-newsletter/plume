import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type ImportResult = { Imported: number; Skipped: number; Failed: number }

export function useImportCsv(listId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (file: File) => {
      const fd = new FormData()
      fd.append('file', file)
      return api<ImportResult>(`/api/lists/${listId}/import`, { method: 'POST', body: fd })
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['subscribers', listId] }),
  })
}
