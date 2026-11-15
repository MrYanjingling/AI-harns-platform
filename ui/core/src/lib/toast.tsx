import type { ReactNode } from 'react'
import { toast } from 'sonner'
import { errorMessage, isApiError } from './http'

function trimToastText(value: string): string {
  return value.replace(/[。．.]+$/u, '').trimEnd()
}

export function errorToastContent(error: unknown): ReactNode {
  if (!isApiError(error)) return trimToastText(errorMessage(error))
  const messages = Array.from(
    new Set(
      error.errors
        .map((item) => item.message?.trim())
        .filter((item): item is string => Boolean(item))
        .map(trimToastText),
    ),
  )
  if (messages.length <= 1) return trimToastText(errorMessage(error))
  return (
    <div className="space-y-1">
      {messages.map((message, index) => (
        <p key={`${index}-${message}`} className="whitespace-pre-line">
          {message}
        </p>
      ))}
    </div>
  )
}

export function showErrorToast(error: unknown): string | number {
  return toast.error(errorToastContent(error))
}
