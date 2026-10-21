import { useSyncExternalStore } from 'react'
import type { OperationAction, OperationLogEntry, OperationResult } from '@/api/types'

const STORAGE_KEY = 'harns-gateway-operation-logs'
const MAX_ENTRIES = 500

function load(): OperationLogEntry[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter(isEntry)
  } catch {
    return []
  }
}

function isEntry(value: unknown): value is OperationLogEntry {
  if (!value || typeof value !== 'object') return false
  const item = value as Partial<OperationLogEntry>
  return (
    typeof item.id === 'string' &&
    typeof item.time === 'string' &&
    typeof item.action === 'string' &&
    typeof item.target === 'string' &&
    typeof item.message === 'string'
  )
}

function save(entries: OperationLogEntry[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(entries))
  } catch {
    // 隐私模式下写入失败不影响当前会话。
  }
}

let entries: OperationLogEntry[] = load()
const listeners = new Set<() => void>()

function emit(): void {
  save(entries)
  for (const listener of listeners) listener()
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

function getSnapshot(): OperationLogEntry[] {
  return entries
}

function createId(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) return crypto.randomUUID()
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

/** 记录一次网关写操作;失败时 message 保留后端错误文案。 */
export function appendOperationLog(input: {
  action: OperationAction
  result: OperationResult
  target: string
  message?: string
  time?: string
}): void {
  const entry: OperationLogEntry = {
    id: createId(),
    time: input.time ?? new Date().toISOString(),
    action: input.action,
    result: input.result,
    target: input.target,
    message: input.message ?? '',
  }
  entries = [entry, ...entries].slice(0, MAX_ENTRIES)
  emit()
}

export function clearOperationLogs(): void {
  entries = []
  emit()
}

/** 组件订阅操作日志;localStorage 在首次加载时已恢复。 */
export function useOperationLogs(): OperationLogEntry[] {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot)
}
