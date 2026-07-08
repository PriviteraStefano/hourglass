import { useState, useCallback } from 'react'
import { toast } from 'sonner'

const API_BASE = import.meta.env.VITE_API_URL || '/api'

export function useDownload() {
  const [isPending, setIsPending] = useState(false)

  const download = useCallback(async (
    path: string,
    options?: { timeout?: number }
  ) => {
    setIsPending(true)
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), options?.timeout ?? 60000)

    try {
      const response = await fetch(`${API_BASE}${path}`, {
        credentials: 'include',
        signal: controller.signal,
      })
      clearTimeout(timeoutId)

      const contentType = response.headers.get('Content-Type') || ''

      // Error responses come as JSON — parse and throw (D-04)
      if (contentType.includes('application/json') || !response.ok) {
        const body = await response.json().catch(() => ({}))
        throw new Error(
          (body as { error?: string }).error || 'Export failed. Please try again.',
        )
      }

      // Success — create blob and trigger download (D-01)
      const blob = await response.blob()
      const disposition = response.headers.get('Content-Disposition')
      let filename = 'export.csv'
      if (disposition) {
        const match = disposition.match(/filename=(.+)/)
        if (match) filename = match[1]
      }

      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    } catch (err) {
      if (err instanceof DOMException && err.name === 'AbortError') {
        toast.error('Export timed out. Please try again.')
      } else {
        toast.error(
          err instanceof Error ? err.message : 'Export failed. Please try again.',
        )
      }
    } finally {
      setIsPending(false)
    }
  }, [])

  return { download, isPending }
}
