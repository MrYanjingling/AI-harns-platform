import type { ReactNode } from 'react'

export function Toolbar({ children }: { children: ReactNode }) {
  return <div className="flex flex-wrap items-center justify-end gap-2 py-4">{children}</div>
}
