import {
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type ClipboardEvent,
  type DragEvent,
} from 'react'
import { AtSign, Paperclip, Pencil, Send, Trash2, X } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  Card,
  CopyButton,
  Textarea,
  cn,
  formatBytes,
  showErrorToast,
  useI18n,
} from '@harns/core'
import { chatWithAssistant, deleteAssistantDocument, uploadAssistantDocuments } from '@/api/gateway'
import { useAllCollections } from '@/api/hooks'
import {
  ASSISTANT_ATTACHMENTS_KEY,
  ASSISTANT_SESSION_KEY,
  clearAssistantSession,
} from '@/lib/assistant-session'
import type {
  AIChatAttachment,
  AIChatClarification,
  AIChatMessage,
  AIChatResponse,
  AIPendingTool,
  AIToolApproval,
  Collection,
} from '@/api/types'
import { AIMarkdown } from './AIMarkdown'
import {
  FILE_ACCEPT,
  MAX_FILES,
  MAX_FILE_BYTES,
  MAX_TOTAL_BYTES,
  isAcceptedFile,
} from './ai-attachments'

const STORAGE_KEY = ASSISTANT_SESSION_KEY
const ATTACHMENTS_KEY = ASSISTANT_ATTACHMENTS_KEY

interface UIMessage extends AIChatMessage {
  files?: string[]
  clarification?: AIChatClarification
  clarificationAnswered?: boolean
}

const RECOMMENDATIONS = [
  'gateway.ai.assistant.recommendCollections',
  'gateway.ai.assistant.recommendConnectors',
  'gateway.ai.assistant.recommendPush',
  'gateway.ai.assistant.recommendErrors',
  'gateway.ai.assistant.recommendDrivers',
] as const

function loadAttachments(): AIChatAttachment[] {
  try {
    const raw = window.sessionStorage.getItem(ATTACHMENTS_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as AIChatAttachment[]
    if (!Array.isArray(parsed)) return []
    return parsed.filter(
      (item) =>
        item &&
        typeof item.id === 'string' &&
        typeof item.name === 'string' &&
        typeof item.size === 'number' &&
        typeof item.textSize === 'number' &&
        typeof item.chunkCount === 'number',
    )
  } catch {
    return []
  }
}

function normalizeClarification(value: unknown): AIChatClarification | undefined {
  if (!value || typeof value !== 'object') return undefined
  const item = value as Partial<AIChatClarification>
  if (
    typeof item.question !== 'string' ||
    !Array.isArray(item.options) ||
    !item.options.every((option) => typeof option === 'string')
  ) {
    return undefined
  }
  return {
    question: item.question,
    options: item.options,
    allowCustom: item.allowCustom !== false,
  }
}

function loadMessages(): UIMessage[] {
  try {
    const raw = window.sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as UIMessage[]
    if (!Array.isArray(parsed)) return []
    return parsed.flatMap((item) => {
      if (
        !item ||
        (item.role !== 'user' && item.role !== 'assistant') ||
        typeof item.content !== 'string'
      ) {
        return []
      }
      return [{
        role: item.role,
        content: item.content,
        files: Array.isArray(item.files)
          ? item.files.filter((file): file is string => typeof file === 'string')
          : undefined,
        clarification: normalizeClarification(item.clarification),
        clarificationAnswered: item.clarificationAnswered === true,
      }]
    })
  } catch {
    return []
  }
}

function ClarificationChoices({
  clarification,
  disabled,
  onChoose,
}: {
  clarification: AIChatClarification
  disabled: boolean
  onChoose: (value: string) => void
}) {
  const { t } = useI18n()
  const [customOpen, setCustomOpen] = useState(false)
  const [customValue, setCustomValue] = useState('')
  const customRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (customOpen) customRef.current?.focus()
  }, [customOpen])

  const submitCustom = () => {
    const value = customValue.trim()
    if (!value || disabled) return
    onChoose(value)
  }

  return (
    <div className="mt-2 border-t border-border/60 pt-2">
      <div className="flex flex-col gap-1.5">
        {clarification.options.map((option) => (
          <button
            key={option}
            type="button"
            disabled={disabled}
            onClick={() => onChoose(option)}
            className="w-full rounded-md border border-border bg-surface px-2.5 py-1.5 text-left text-[13px] leading-5 text-fg transition-colors hover:border-primary hover:bg-surface-2 disabled:pointer-events-none disabled:opacity-50"
          >
            {option}
          </button>
        ))}
        {clarification.allowCustom ? (
          customOpen ? (
            <div className="flex items-center gap-1.5">
              <input
                ref={customRef}
                value={customValue}
                disabled={disabled}
                placeholder={t('gateway.ai.assistant.customAnswerPlaceholder')}
                onChange={(event) => setCustomValue(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') {
                    event.preventDefault()
                    submitCustom()
                  }
                  if (event.key === 'Escape') {
                    event.preventDefault()
                    setCustomOpen(false)
                    setCustomValue('')
                  }
                }}
                className="min-w-0 flex-1 rounded-md border border-border-strong bg-surface-2 px-2.5 py-1.5 text-[13px] text-fg outline-none focus:border-primary focus:ring-2 focus:ring-primary/15 disabled:opacity-50"
              />
              <Button
                type="button"
                size="icon"
                disabled={!customValue.trim() || disabled}
                aria-label={t('gateway.ai.assistant.send')}
                title={t('gateway.ai.assistant.send')}
                icon={<Send className="size-3.5" />}
                onClick={submitCustom}
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label={t('gateway.ai.assistant.customAnswerCancel')}
                title={t('gateway.ai.assistant.customAnswerCancel')}
                icon={<X className="size-3.5" />}
                onClick={() => {
                  setCustomOpen(false)
                  setCustomValue('')
                }}
              />
            </div>
          ) : (
            <button
              type="button"
              disabled={disabled}
              onClick={() => setCustomOpen(true)}
              className="flex w-full items-center gap-1.5 rounded-md border border-dashed border-border bg-transparent px-2.5 py-1.5 text-left text-[13px] text-muted transition-colors hover:border-primary hover:text-fg disabled:pointer-events-none disabled:opacity-50"
            >
              <Pencil className="size-3.5 shrink-0" />
              {t('gateway.ai.assistant.customAnswer')}
            </button>
          )
        ) : null}
      </div>
    </div>
  )
}

export function AIAssistantChat({
  active = true,
  className,
}: {
  active?: boolean
  className?: string
}) {
  const { t } = useI18n()
  const [messages, setMessages] = useState<UIMessage[]>(loadMessages)
  const [input, setInput] = useState('')
  const [attachments, setAttachments] = useState<AIChatAttachment[]>(loadAttachments)
  const [pending, setPending] = useState<{ id: string; tools: AIPendingTool[] } | null>(null)
  const [sending, setSending] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [dragActive, setDragActive] = useState(false)
  const listRef = useRef<HTMLDivElement>(null)
  const fileRef = useRef<HTMLInputElement>(null)
  const dragDepth = useRef(0)
  const inputId = useId()
  const [mentionRange, setMentionRange] = useState<{ start: number; end: number } | null>(null)
  const [mentionIndex, setMentionIndex] = useState(0)
  const collections = useAllCollections(Boolean(active) && mentionRange !== null)
  const mentionItems = useMemo(() => {
    const source = collections.data ?? []
    const query = mentionRange
      ? input.slice(mentionRange.start + 1, mentionRange.end).trim().toLowerCase()
      : ''
    if (!query) return source
    return source.filter((item) =>
      [item.name, item.id, item.driverType, item.driverModel]
        .join(' ')
        .toLowerCase()
        .includes(query),
    )
  }, [collections.data, input, mentionRange])
  const activeMentionIndex = Math.min(
    mentionIndex,
    Math.max(0, mentionItems.length - 1),
  )

  useEffect(() => {
    if (!active) return
    const timer = window.setTimeout(() => {
      document.getElementById(inputId)?.focus()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [active, inputId])

  useEffect(() => {
    if (!active) setMentionRange(null)
  }, [active])

  useEffect(() => {
    try {
      window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(messages))
    } catch {
      // Ignore storage quota failures.
    }
  }, [messages])

  useEffect(() => {
    try {
      window.sessionStorage.setItem(ATTACHMENTS_KEY, JSON.stringify(attachments))
    } catch {
      // Ignore storage quota failures.
    }
  }, [attachments])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      const list = listRef.current
      if (list) list.scrollTop = list.scrollHeight
    }, 0)
    return () => window.clearTimeout(timer)
  }, [messages, sending])

  const addFiles = async (files: FileList | File[]) => {
    if (uploading) return
    const selected = Array.from(files)
    if (selected.length === 0) return
    if (attachments.length + selected.length > MAX_FILES) {
      toast.error(t('gateway.ai.assistant.fileLimit', { count: MAX_FILES }))
      return
    }
    let total = attachments.reduce((sum, item) => sum + item.size, 0)
    const accepted: File[] = []
    for (const file of selected) {
      if (!isAcceptedFile(file)) {
        toast.error(t('gateway.ai.assistant.fileTypeInvalid', { name: file.name }))
        continue
      }
      if (file.size > MAX_FILE_BYTES) {
        toast.error(t('gateway.ai.assistant.fileTooLarge', { name: file.name }))
        continue
      }
      if (total + file.size > MAX_TOTAL_BYTES) {
        toast.error(t('gateway.ai.assistant.fileTotalTooLarge'))
        break
      }
      total += file.size
      accepted.push(file)
    }
    if (accepted.length === 0) return
    setUploading(true)
    try {
      const documents = await uploadAssistantDocuments(accepted)
      setAttachments((current) => [...current, ...documents])
    } catch (error) {
      showErrorToast(error)
    } finally {
      setUploading(false)
    }
  }

  const hasFiles = (event: DragEvent<HTMLElement>) =>
    Array.from(event.dataTransfer.types).includes('Files')

  const onDragEnter = (event: DragEvent<HTMLElement>) => {
    if (!hasFiles(event)) return
    event.preventDefault()
    dragDepth.current += 1
    setDragActive(true)
  }

  const onDragOver = (event: DragEvent<HTMLElement>) => {
    if (!hasFiles(event)) return
    event.preventDefault()
    event.dataTransfer.dropEffect = 'copy'
  }

  const onDragLeave = (event: DragEvent<HTMLElement>) => {
    if (dragDepth.current === 0) return
    event.preventDefault()
    dragDepth.current = Math.max(0, dragDepth.current - 1)
    if (dragDepth.current === 0) setDragActive(false)
  }

  const onDrop = (event: DragEvent<HTMLElement>) => {
    if (!hasFiles(event)) return
    event.preventDefault()
    dragDepth.current = 0
    setDragActive(false)
    void addFiles(event.dataTransfer.files)
  }

  const removeAttachment = (item: AIChatAttachment) => {
    setAttachments((current) => current.filter((entry) => entry.id !== item.id))
    void deleteAssistantDocument(item.id).catch(showErrorToast)
  }

  const onPaste = (event: ClipboardEvent<HTMLTextAreaElement>) => {
    const files = Array.from(event.clipboardData.files)
    if (files.length === 0) return
    event.preventDefault()
    void addFiles(files)
  }

  const updateMention = (value: string, cursor: number) => {
    const before = value.slice(0, cursor)
    const match = /(^|[\s(])@([^\s@]*)$/.exec(before)
    if (!match) {
      setMentionRange(null)
      return
    }
    const start = cursor - match[2].length - 1
    setMentionRange({ start, end: cursor })
    setMentionIndex(0)
  }

  const handleInputChange = (value: string, cursor: number) => {
    setInput(value)
    updateMention(value, cursor)
  }

  const selectMention = (item: Collection) => {
    if (!mentionRange) return
    const prompt = t('gateway.ai.assistant.mentionPrompt', {
      name: item.name,
      id: item.id,
    })
    const next = input.slice(0, mentionRange.start) + prompt + input.slice(mentionRange.end)
    setInput(next)
    setMentionRange(null)
    window.requestAnimationFrame(() => {
      const inputElement = document.getElementById(inputId) as HTMLTextAreaElement | null
      inputElement?.focus()
      inputElement?.setSelectionRange(next.length, next.length)
    })
  }

  const applyResponse = (response: AIChatResponse) => {
    if (response.pending?.length && response.pendingId) {
      setPending({ id: response.pendingId, tools: response.pending })
      return
    }
    setPending(null)
    if (response.reply || response.clarification) {
      setMessages((current) => [
        ...current,
        {
          role: 'assistant',
          content: response.reply,
          clarification: response.clarification,
        },
      ])
    }
  }

  const sendText = async (value: string) => {
    const content = value.trim()
    if ((!content && attachments.length === 0) || sending || uploading || pending) return
    const currentAttachments = attachments
    let answeredMessages = messages
    for (let index = messages.length - 1; index >= 0; index -= 1) {
      const message = messages[index]
      if (message.role === 'assistant' && message.clarification && !message.clarificationAnswered) {
        answeredMessages = messages.map((entry, entryIndex) =>
          entryIndex === index ? { ...entry, clarificationAnswered: true } : entry,
        )
        break
      }
    }
    const next = [
      ...answeredMessages,
      {
        role: 'user' as const,
        content: content || t('gateway.ai.assistant.analyzeAttachment'),
        files: currentAttachments.map((item) => item.name),
      },
    ]
    setMessages(next)
    setInput('')
    setSending(true)
    try {
      const response: AIChatResponse = await chatWithAssistant(
        next.map((item) => ({ role: item.role, content: item.content })),
        currentAttachments,
      )
      applyResponse(response)
    } catch (error) {
      showErrorToast(error)
    } finally {
      setSending(false)
    }
  }

  const resolvePending = async (approved: boolean) => {
    if (!pending || sending) return
    const approvals: AIToolApproval[] = pending.tools.map((item) => ({
      id: item.id,
      approved,
    }))
    setSending(true)
    try {
      const response = await chatWithAssistant(
        messages.map((item) => ({ role: item.role, content: item.content })),
        [],
        { pendingId: pending.id, approvals },
      )
      applyResponse(response)
    } catch (error) {
      showErrorToast(error)
      setPending(null)
    } finally {
      setSending(false)
    }
  }

  const clear = () => {
    const documentIDs = attachments.map((item) => item.id)
    setMessages([])
    setInput('')
    setAttachments([])
    setPending(null)
    clearAssistantSession()
    documentIDs.forEach((id) => {
      void deleteAssistantDocument(id).catch(() => undefined)
    })
  }

  return (
    <div className={cn('flex min-h-0 flex-1 flex-col', className)}>
      <Card
        className="relative flex min-h-0 flex-1 flex-col overflow-hidden"
        onDragEnter={onDragEnter}
        onDragOver={onDragOver}
        onDragLeave={onDragLeave}
        onDrop={onDrop}
      >
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
                        disabled={sending || Boolean(pending)}
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
                    <div className="min-w-0 flex-1">
                      {message.role === 'assistant' ? (
                        <AIMarkdown content={message.content} />
                      ) : (
                        <p className="whitespace-pre-wrap break-words text-sm leading-6">
                          {message.content}
                        </p>
                      )}
                    </div>
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
                  message.clarification &&
                  !message.clarificationAnswered ? (
                    <ClarificationChoices
                      clarification={message.clarification}
                      disabled={sending || uploading}
                      onChoose={(value) => void sendText(value)}
                    />
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
        {pending ? (
          <div className="shrink-0 border-t border-warning/40 bg-warning/5 px-4 py-3">
            <div className="rounded-lg border border-warning/40 bg-surface p-3">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="text-sm font-semibold text-fg">
                    {t('gateway.ai.assistant.approvalTitle')}
                  </p>
                  <p className="mt-1 text-[13px] text-muted">
                    {t('gateway.ai.assistant.approvalDescription')}
                  </p>
                </div>
                <span className="shrink-0 rounded border border-warning/40 px-1.5 py-0.5 text-[11px] font-medium text-warning">
                  AI
                </span>
              </div>
              <div className="mt-3 space-y-2">
                {pending.tools.map((item) => (
                  <div key={item.id} className="rounded border border-border bg-surface-2 p-2">
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-[13px] text-fg">{item.name}</span>
                      <span className="rounded bg-warning/10 px-1.5 py-0.5 text-[11px] text-warning">
                        {t(
                          item.risk === 'write'
                            ? 'gateway.ai.assistant.approvalRiskWrite'
                            : 'gateway.ai.assistant.approvalRiskAction',
                        )}
                      </span>
                    </div>
                    <pre className="mt-2 max-h-32 overflow-auto whitespace-pre-wrap break-all rounded bg-surface p-2 font-mono text-[11px] text-muted">
                      {JSON.stringify(item.arguments, null, 2)}
                    </pre>
                  </div>
                ))}
              </div>
              <div className="mt-3 flex justify-end gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={sending}
                  onClick={() => void resolvePending(false)}
                >
                  {t('gateway.ai.assistant.approvalDeny')}
                </Button>
                <Button
                  type="button"
                  size="sm"
                  disabled={sending}
                  onClick={() => void resolvePending(true)}
                >
                  {t('gateway.ai.assistant.approvalApprove')}
                </Button>
              </div>
            </div>
          </div>
        ) : null}

        <form
          className="shrink-0 border-t border-border p-3"
          onSubmit={(event) => {
            event.preventDefault()
            void sendText(input)
          }}
        >
          <div className="relative rounded-xl border border-border-strong bg-surface-2 px-3 transition-colors focus-within:border-primary focus-within:ring-2 focus-within:ring-primary/15">
            {mentionRange ? (
              <div
                role="listbox"
                aria-label={t('gateway.ai.assistant.mentionTitle')}
                className="absolute right-0 bottom-full left-0 z-20 mb-2 max-h-64 overflow-y-auto rounded-lg border border-border bg-surface p-1 shadow-2xl"
              >
                <div className="px-2 py-1.5 text-[11px] font-medium text-faint">
                  {t('gateway.ai.assistant.mentionTitle')}
                </div>
                {collections.isLoading ? (
                  <div className="px-2 py-2 text-[13px] text-muted">
                    {t('gateway.ai.assistant.mentionLoading')}
                  </div>
                ) : mentionItems.length === 0 ? (
                  <div className="px-2 py-2 text-[13px] text-muted">
                    {t('gateway.ai.assistant.mentionEmpty')}
                  </div>
                ) : (
                  mentionItems.map((item, index) => (
                    <button
                      key={item.id}
                      type="button"
                      role="option"
                      aria-selected={index === activeMentionIndex}
                      onMouseDown={(event) => event.preventDefault()}
                      onMouseEnter={() => setMentionIndex(index)}
                      onClick={() => selectMention(item)}
                      className={cn(
                        'flex w-full items-center gap-2 rounded px-2 py-2 text-left text-sm text-fg transition-colors',
                        index === activeMentionIndex
                          ? 'bg-surface-2'
                          : 'hover:bg-surface-2',
                      )}
                    >
                      <AtSign className="size-3.5 shrink-0 text-faint" />
                      <span className="min-w-0 flex-1 truncate font-medium">{item.name}</span>
                      <span className="max-w-40 shrink-0 truncate text-[11px] text-faint">
                        {item.driverType} · {item.driverModel}
                      </span>
                    </button>
                  ))
                )}
              </div>
            ) : null}
            <Textarea
              id={inputId}
              value={input}
              placeholder={t('gateway.ai.assistant.placeholder')}
              disabled={sending || Boolean(pending)}
              rows={3}
              className="min-h-[60px] resize-none border-0 bg-transparent px-0 py-0 text-sm focus:ring-0"
              onChange={(event) =>
                handleInputChange(event.target.value, event.target.selectionStart ?? 0)
              }
              onPaste={onPaste}
              onKeyDown={(event) => {
                if (mentionRange && mentionItems.length > 0) {
                  if (event.key === 'ArrowDown') {
                    event.preventDefault()
                    setMentionIndex((current) => (current + 1) % mentionItems.length)
                    return
                  }
                  if (event.key === 'ArrowUp') {
                    event.preventDefault()
                    setMentionIndex(
                      (current) => (current - 1 + mentionItems.length) % mentionItems.length,
                    )
                    return
                  }
                  if (event.key === 'Enter' || event.key === 'Tab') {
                    event.preventDefault()
                    selectMention(mentionItems[activeMentionIndex])
                    return
                  }
                }
                if (event.key === 'Escape' && mentionRange) {
                  event.preventDefault()
                  event.stopPropagation()
                  setMentionRange(null)
                  return
                }
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
                    key={item.id}
                    title={t('gateway.ai.assistant.documentMeta', {
                      size: formatBytes(item.textSize),
                      chunks: item.chunkCount,
                    })}
                    className="inline-flex items-center gap-1 rounded border border-border bg-surface px-2 py-0.5 text-[13px] text-muted"
                  >
                    <Paperclip className="size-3" />
                    <span className="max-w-48 truncate">{item.name}</span>
                    <span className="shrink-0 text-[11px] text-faint">
                      {formatBytes(item.textSize)}
                    </span>
                    <button
                      type="button"
                      aria-label={t('gateway.ai.assistant.removeFile')}
                      title={t('gateway.ai.assistant.removeFile')}
                      disabled={sending || uploading}
                      onClick={() => removeAttachment(item)}
                      className="text-faint transition-colors hover:text-fg disabled:pointer-events-none disabled:opacity-50"
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
                  accept={FILE_ACCEPT}
                  className="hidden"
                  disabled={uploading}
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
                  disabled={uploading}
                  onClick={() => fileRef.current?.click()}
                />
                <p className="truncate text-[13px] text-muted">
                  {uploading
                    ? t('gateway.ai.assistant.uploading')
                    : t('gateway.ai.assistant.composerHint')}
                </p>
              </div>
              <div className="flex items-center gap-2 pb-1">
                <Button
                  type="button"
                  variant="dangerSoft"
                  icon={<Trash2 className="size-3.5" />}
                  disabled={
                    (messages.length === 0 && attachments.length === 0) || sending || uploading
                  }
                  onClick={clear}
                >
                  {t('gateway.ai.assistant.clear')}
                </Button>
                <Button
                  type="submit"
                  variant="primary"
                  loading={sending}
                  icon={<Send className="size-3.5" />}
                  disabled={(!input.trim() && attachments.length === 0) || sending || uploading}
                >
                  {t('gateway.ai.assistant.send')}
                </Button>
              </div>
            </div>
          </div>
        </form>
        {dragActive && !uploading ? (
          <div className="pointer-events-none absolute inset-0 z-30 flex items-center justify-center rounded border-2 border-dashed border-primary bg-surface/90">
            <div className="flex flex-col items-center gap-1.5 text-primary">
              <Paperclip className="size-5" />
              <span className="text-[13px] font-medium">
                {t('gateway.ai.assistant.dropHint')}
              </span>
            </div>
          </div>
        ) : null}
      </Card>
    </div>
  )
}
