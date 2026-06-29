import { useRef, useState } from 'react'
import { Upload } from 'lucide-react'
import { toast } from 'sonner'
import { useImportCsv } from './useImport'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'

export function ImportDialog({ listId }: { listId: string }) {
  const [open, setOpen] = useState(false)
  const [file, setFile] = useState<File | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const imp = useImportCsv(listId)

  function handleClose() {
    setOpen(false)
    setFile(null)
    imp.reset()
    if (inputRef.current) inputRef.current.value = ''
  }

  function handleUpload() {
    if (!file) return
    imp.mutate(file, {
      onSuccess: (r) => {
        toast.success(`Imported ${r.Imported}`)
      },
      onError: () => {
        toast.error('Import failed')
      },
    })
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) handleClose(); else setOpen(true) }}>
      <DialogTrigger render={<Button variant="outline">Import CSV</Button>} />
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Import subscribers from CSV</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="flex flex-col gap-2">
            <label
              htmlFor="csv-file-input"
              className="text-sm font-medium text-foreground"
            >
              CSV file
            </label>
            <input
              id="csv-file-input"
              ref={inputRef}
              type="file"
              accept=".csv"
              aria-label="CSV file"
              className="block w-full text-sm text-muted-foreground file:mr-3 file:rounded-md file:border file:border-input file:bg-background file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-foreground hover:file:bg-muted cursor-pointer"
              onChange={(e) => {
                setFile(e.target.files?.[0] ?? null)
                imp.reset()
              }}
            />
          </div>
          {imp.isSuccess && imp.data && (
            <div className="rounded-md bg-muted/50 px-3 py-2 text-sm space-y-1">
              <p className="text-foreground font-medium">Import complete</p>
              <ul className="text-muted-foreground space-y-0.5">
                <li>Imported {imp.data.Imported}</li>
                <li>Skipped {imp.data.Skipped}</li>
                <li>Failed {imp.data.Failed}</li>
              </ul>
            </div>
          )}
          {imp.isError && (
            <p className="text-sm text-destructive">
              {imp.error instanceof Error ? imp.error.message : 'Import failed'}
            </p>
          )}
        </div>
        <DialogFooter>
          <Button
            onClick={handleUpload}
            disabled={!file || imp.isPending}
          >
            {imp.isPending ? (
              <>
                <Upload className="size-4 animate-pulse" aria-hidden />
                Uploading…
              </>
            ) : (
              <>
                <Upload className="size-4" aria-hidden />
                Upload
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
