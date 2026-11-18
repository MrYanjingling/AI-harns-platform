import { useRef, useState } from 'react'

export type ColumnWidths = Record<string, number>

export function useColumnResize(initialWidths: ColumnWidths) {
  const [widths, setWidths] = useState(initialWidths)
  const resize = (column: string, width: number) => {
    setWidths((current) => ({
      ...current,
      [column]: Math.max(48, Math.min(480, Math.round(width))),
    }))
  }
  const totalWidth = Object.values(widths).reduce((sum, width) => sum + width, 0)
  return { widths, totalWidth, resize }
}

export function ColumnResizeHandle({
  column,
  width,
  onResize,
}: {
  column: string
  width: number
  onResize: (column: string, width: number) => void
}) {
  const start = useRef<{ x: number; width: number } | null>(null)

  return (
    <span
      role="separator"
      aria-orientation="vertical"
      onPointerDown={(event) => {
        event.preventDefault()
        event.currentTarget.setPointerCapture(event.pointerId)
        start.current = { x: event.clientX, width }
        document.body.style.userSelect = 'none'
      }}
      onPointerMove={(event) => {
        if (!start.current) return
        onResize(column, start.current.width + event.clientX - start.current.x)
      }}
      onPointerUp={(event) => {
        start.current = null
        document.body.style.userSelect = ''
        event.currentTarget.releasePointerCapture(event.pointerId)
      }}
      onPointerCancel={() => {
        start.current = null
        document.body.style.userSelect = ''
      }}
      className="absolute inset-y-0 right-0 z-10 w-1.5 -translate-x-px cursor-col-resize touch-none select-none rounded-full hover:bg-primary/30 active:bg-primary/50"
    />
  )
}
