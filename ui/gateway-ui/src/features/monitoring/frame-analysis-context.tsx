import { createContext, useContext, type ReactNode } from 'react'

export interface FrameAnalysisContextValue {
  open: (collectionId: string) => void
  update: (collectionId: string) => void
  close: () => void
}

const FrameAnalysisContext = createContext<FrameAnalysisContextValue | null>(null)

export function FrameAnalysisProvider({
  value,
  children,
}: {
  value: FrameAnalysisContextValue
  children: ReactNode
}) {
  return (
    <FrameAnalysisContext.Provider value={value}>
      {children}
    </FrameAnalysisContext.Provider>
  )
}

export function useFrameAnalysis(): FrameAnalysisContextValue {
  const context = useContext(FrameAnalysisContext)
  if (!context) {
    throw new Error('FrameAnalysisProvider is missing')
  }
  return context
}
