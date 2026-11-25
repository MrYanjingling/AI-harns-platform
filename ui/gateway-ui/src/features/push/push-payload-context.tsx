import { createContext, useContext, type ReactNode } from 'react'

export interface PushPayloadContextValue {
  open: (groupId: string) => void
  close: () => void
}

const PushPayloadContext = createContext<PushPayloadContextValue | null>(null)

export function PushPayloadProvider({
  value,
  children,
}: {
  value: PushPayloadContextValue
  children: ReactNode
}) {
  return <PushPayloadContext.Provider value={value}>{children}</PushPayloadContext.Provider>
}

export function usePushPayload(): PushPayloadContextValue {
  const context = useContext(PushPayloadContext)
  if (!context) throw new Error('PushPayloadProvider is missing')
  return context
}
