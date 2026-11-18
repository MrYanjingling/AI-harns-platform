import type { CollectionVariable } from '@/api/types'

/** Format a variable value using its declared type and configured precision. */
export function formatVariableValue(
  variable: Pick<CollectionVariable, 'value' | 'dataType' | 'precision'> | null | undefined,
): string {
  if (!variable) return '—'
  const value = variable.value
  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'number') {
    const dataType = variable.dataType.trim().toLowerCase()
    const precision = variable.precision ?? 0
    if (['float32', 'float64', 'number'].includes(dataType) && precision > 0) {
      return value.toFixed(precision)
    }
  }
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}
