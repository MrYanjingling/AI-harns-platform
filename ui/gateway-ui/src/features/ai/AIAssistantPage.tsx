import { useEffect, useRef, useState, type ClipboardEvent } from 'react'
import { Paperclip, Send, Trash2, X } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  Card,
  CopyButton,
  Textarea,
  showErrorToast,
  useI18n,
} from '@harns/core'
import { chatWithAssistant } from '@/api/gateway'
import type { AIChatAttachment, AIChatMessage, AIChatResponse } from '@/api/types'

const STORAGE_KEY = 'gateway.ai.assistant.session'
const MAX_FILES = 5
const MAX_FILE_BYTES = 128 << 10
const MAX_TOTAL_BYTES = 256 << 10
const ACCEPTED_FILE = /\.(txt|log|json|ya?ml|csv|md|xml)$/i

interface UIMessage extends AIChatMessage {
  tools?: string[]
  files?: string[]
}

const RECOMMENDATIONS = [
  'gateway.ai.assistant.recommendCollections',
  'gateway.ai.assistant.recommendConnectors',
  'gateway.ai.assistant.recommendPush',
  'gateway.ai.assistant.recommendErrors',
  'gateway.ai.assistant.recommendDrivers',
] as const

function loadMessages(): UIMessage[] {
  try {
    const raw = window.sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as UIMessage[]
    if (!Array.isArray(parsed)) return []
    return parsed.filter(
      (item) =>
        item &&
        (item.role === 'user' || item.role === 'assistant') &&
        typeof item.content === 'string',
    )
  } catch {
    return []
  }
}

export function AIAssistantPage() {
  const { t } = useI18n()
  const [messages, setMessages] = useState<UIMessage[]>(loadMessages)
  const [input, setInput] = useState('')
  const [attachments, setAttachments] = useState<AIChatAttachment[]>([])
  const [sending, setSending] = useState(false)
  const listRef = useRef<HTMLDivElement>(null)
  const fileRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    try {
      window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(messages))
    } catch {
      // Ignore storage quota failures.
    }
  }, [messages])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      const list = listRef.current
      if (list) list.scrollTop = list.scrollHeight
    }, 0)
    return () => window.clearTimeout(timer)
  }, [messages, sending])

  const addFiles = async (files: FileList | File[]) => {
    const selected = Array.from(files)
    if (selected.length === 0) return
    if (attachments.length + selected.length > MAX_FILES) {
      toast.error(t('gateway.ai.assistant.fileLimit', { count: MAX_FILES }))
      return
    }
    const next = [...attachments]
    let total = next.reduce((sum, item) => sum + item.content.length, 0)
    for (const file of selected) {
      if (!ACCEPTED_FILE.test(file.name) && !file.type.startsWith('text/')) {
        toast.error(t('gateway.ai.assistant.fileTypeInvalid', { name: file.name }))
        continue
      }
      if (file.size > MAX_FILE_BYTES) {
        toast.error(t('gateway.ai.assistant.fileTooLarge', { name: file.name }))
        continue
      }
      total += file.size
      if (total > MAX_TOTAL_BYTES) {
        toast.error(t('gateway.ai.assistant.fileTotalTooLarge'))
        break
      }
      try {
        next.push({ name: file.name, content: await file.text() })
      } catch {
        toast.error(t('gateway.ai.assistant.fileReadFailed', { name: file.name }))
      }
    }
    setAttachments(next)
  }

  const removeAttachment = (name: string) => {
    setAttachments((current) => current.filter((item) => item.name !== name))
  }

  const onPaste = (event: ClipboardEvent<HTMLTextAreaElement>) => {
    const files = Array.from(event.clipboardData.files)
    if (files.length === 0) return
    event.preventDefault()
    void addFiles(files)
  }

  const sendText = async (value: string) => {
    const content = value.trim()
    if ((!content && attachments.length === 0) || sending) return
    const currentAttachments = attachments
    const next = [
      ...messages,
      {
        role: 'user' as const,
        content: content || t('gateway.ai.assistant.analyzeAttachment'),
        files: currentAttachments.map((item) => item.name),
      },
    ]
    setMessages(next)
    setInput('')
    setAttachments([])
    setSending(true)
    try {
      const response: AIChatResponse = await chatWithAssistant(
        next.map((item) => ({ role: item.role, content: item.content })),
        currentAttachments,
      )
      setMessages((current) => [
        ...current,
        { role: 'assistant', content: response.reply, tools: response.tools },
      ])
    } catch (error) {
      showErrorToast(error)
    } finally {
      setSending(false)
    }
  }

  const clear = () => {
    setMessages([])
    setInput('')
    setAttachments([])
    window.sessionStorage.removeItem(STORAGE_KEY)
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <Card className="flex min-h-0 flex-1 flex-col overflow-hidden">
        <div ref={listRef} className="flex min-h-0 flex-1 flex-col overflow-y-auto px-4 pt-0 pb-4">
          {messages.length === 0 ? (
            <div className="flex flex-1 items-center justify-center px-4">
              <div className="w-full max-w-xl">
                <div className="flex flex-wrap justify-center gap-2">
                  {RECOMMENDATIONS.map((key) => {
                    const label = t(key)
                    return (
                      <button
                        key={key}
                        type="button"
                        disabled={sending}
                        onClick={() => void sendText(label)}
                        className="rounded-lg border border-primary bg-primary px-3 py-1.5 text-left text-[13px] leading-relaxed text-primary-fg transition-colors hover:bg-primary-hover active:translate-y-px disabled:pointer-events-none disabled:opacity-50"
                      >
                        {label}
                      </button>
                    )
                  })}
                </div>
              </div>
            </div>
          ) : (
            <div className="flex flex-col gap-3">
              {messages.map((message, index) => (
                <div
                  key={`${message.role}-${index}`}
                  className={
                    message.role === 'user'
                      ? 'ml-auto max-w-[85%] rounded-lg border border-border bg-surface-2 px-3 py-1.5 text-sm text-fg'
                      : 'mr-auto max-w-[85%] rounded-lg border border-border bg-surface-2 px-3 py-1.5 text-sm text-fg'
                  }
                >
                  <div className="flex items-start gap-2">
                    <p className="min-w-0 flex-1 whitespace-pre-wrap break-words">
                      {message.content}
                    </p>
                    <CopyButton value={message.content} className="size-5 shrink-0" />
                  </div>
                  {message.files && message.files.length > 0 ? (
                    <p className="mt-1 text-[13px] opacity-80">
                      {t('gateway.ai.assistant.filesAttached', {
                        files: message.files.join(', '),
                      })}
                    </p>
                  ) : null}
                  {message.role === 'assistant' &&
                  message.tools &&
                  message.tools.length > 0 ? (
                    <p className="mt-2 text-[13px] text-muted">
                      {t('gateway.ai.assistant.toolsUsed', {
                        tools: message.tools.join(', '),
                      })}
                    </p>
                  ) : null}
                </div>
              ))}
              {sending ? (
                <div
                  role="status"
                  aria-live="polite"
                  className="mr-auto flex items-center gap-2 rounded-lg border border-border bg-surface-2 px-3 py-2 text-[13px] text-muted"
                >
                  <span className="ai-thinking-orb" aria-hidden="true" />
                  <span>{t('gateway.ai.assistant.thinking')}</span>
                </div>
              ) : null}
            </div>
          )}
        </div>

        <form
          className="shrink-0 border-t border-border p-3"
          onSubmit={(event) => {
            event.preventDefault()
            void sendText(input)
          }}
        >
          <div className="rounded-xl border border-border-strong bg-surface-2 px-3 transition-colors focus-within:border-primary focus-within:ring-2 focus-within:ring-primary/15">
            <Textarea
              autoFocus
              value={input}
              placeholder={t('gateway.ai.assistant.placeholder')}
              disabled={sending}
              rows={3}
              className="min-h-[60px] resize-none border-0 bg-transparent px-0 py-0 text-sm focus:ring-0"
              onChange={(event) => setInput(event.target.value)}
              onPaste={onPaste}
              onKeyDown={(event) => {
                if (event.key === 'Enter' && !event.shiftKey) {
                  event.preventDefault()
                  void sendText(input)
                }
              }}
            />
            {attachments.length > 0 ? (
              <div className="flex flex-wrap gap-1.5">
                {attachments.map((item) => (
                  <span
                    key={item.name}
                    className="inline-flex items-center gap-1 rounded border border-border bg-surface px-2 py-0.5 text-[13px] text-muted"
                  >
                    <Paperclip className="size-3" />
                    <span className="max-w-48 truncate">{item.name}</span>
                    <button
                      type="button"
                      aria-label={t('gateway.ai.assistant.removeFile')}
                      title={t('gateway.ai.assistant.removeFile')}
                      onClick={() => removeAttachment(item.name)}
                      className="text-faint transition-colors hover:text-fg"
                    >
                      <X className="size-3" />
                    </button>
                  </span>
                ))}
              </div>
            ) : null}
            <div className="flex items-center justify-between gap-3">
              <div className="flex min-w-0 items-center gap-1.5">
                <input
                  ref={fileRef}
                  type="file"
                  multiple
                  accept=".txt,.log,.json,.yaml,.yml,.csv,.md,.xml,text/*"
                  className="hidden"
                  onChange={(event) => {
                    if (event.currentTarget.files) void addFiles(event.currentTarget.files)
                    event.currentTarget.value = ''
                  }}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-label={t('gateway.ai.assistant.attachFile')}
                  title={t('gateway.ai.assistant.attachFile')}
                  icon={<Paperclip className="size-3.5" />}
                  onClick={() => fileRef.current?.click()}
                />
                <p className="truncate text-[13px] text-muted">
                  {t('gateway.ai.assistant.composerHint')}
                </p>
              </div>
              <div className="flex items-center gap-2 pb-1">
                <Button
                  type="button"
                  variant="dangerSoft"
                  icon={<Trash2 className="size-3.5" />}
                  disabled={(messages.length === 0 && attachments.length === 0) || sending}
                  onClick={clear}
                >
                  {t('gateway.ai.assistant.clear')}
                </Button>
                <Button
                  type="submit"
                  variant="primary"
                  loading={sending}
                  icon={<Send className="size-3.5" />}
                  disabled={(!input.trim() && attachments.length === 0) || sending}
                >
                  {t('gateway.ai.assistant.send')}
                </Button>
              </div>
            </div>
          </div>
        </form>
      </Card>
    </div>
  )
}
