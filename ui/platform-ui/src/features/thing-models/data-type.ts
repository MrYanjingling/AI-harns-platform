import type {
  ThingDataType,
  ThingDataTypeDetail,
  ThingModelFeature,
  ThingModelPropertyInput,
} from '@/api/types'
import { NUMERIC_DATA_TYPES, THING_DATA_TYPES } from '@/lib/constants'

export interface FieldIssue {
  field: string
  message: string
}

export const IDENTIFIER_PATTERN = /^[A-Za-z][A-Za-z0-9_]*$/
const RESERVED_IDENTIFIERS = new Set(['time', 'device_id'])

export function dataTypeLabel(type: ThingDataType): string {
  return THING_DATA_TYPES.find((item) => item.value === type)?.label ?? type
}

export function isNumericDataType(type: ThingDataType): boolean {
  return NUMERIC_DATA_TYPES.includes(type)
}

/** 切换类型时只保留该类型允许的字段,与后端「宽松忽略」行为一致。 */
export function normalizeDataType(detail: ThingDataTypeDetail): ThingDataTypeDetail {
  switch (detail.type) {
    case 'int':
    case 'long':
    case 'float':
    case 'double': {
      const next: ThingDataTypeDetail = { type: detail.type }
      if (detail.unit?.trim()) next.unit = detail.unit.trim()
      if (detail.unitName?.trim()) next.unitName = detail.unitName.trim()
      if (detail.min?.trim()) next.min = detail.min.trim()
      if (detail.max?.trim()) next.max = detail.max.trim()
      return next
    }
    case 'bool': {
      const next: ThingDataTypeDetail = { type: detail.type }
      if (detail.trueValue?.trim()) next.trueValue = detail.trueValue.trim()
      if (detail.falseValue?.trim()) next.falseValue = detail.falseValue.trim()
      return next
    }
    case 'enum':
      return {
        type: detail.type,
        enumSpec: {
          values: (detail.enumSpec?.values ?? [])
            .filter((item) => Number.isFinite(item.value))
            .map((item) => ({ value: Math.trunc(item.value), desc: item.desc ?? '' })),
        },
      }
    case 'text': {
      const next: ThingDataTypeDetail = { type: detail.type }
      if (detail.length?.trim()) next.length = detail.length.trim()
      return next
    }
    case 'time':
      return { type: detail.type }
    case 'array':
      return {
        type: detail.type,
        arraySpec: {
          ...(detail.arraySpec?.size?.trim() ? { size: detail.arraySpec.size.trim() } : {}),
          element: normalizeDataType(detail.arraySpec?.element ?? { type: 'double' }),
        },
      }
    case 'struct':
      return {
        type: detail.type,
        structSpec: {
          fields: (detail.structSpec?.fields ?? []).map((field) => ({
            identifier: field.identifier ?? '',
            name: field.name ?? '',
            dataType: normalizeDataType(field.dataType ?? { type: 'double' }),
          })),
        },
      }
    default:
      return { type: detail.type }
  }
}

export function emptyDataType(type: ThingDataType = 'double'): ThingDataTypeDetail {
  switch (type) {
    case 'enum':
      return { type, enumSpec: { values: [{ value: 0, desc: '' }] } }
    case 'array':
      return { type, arraySpec: { element: { type: 'double' } } }
    case 'struct':
      return {
        type,
        structSpec: { fields: [{ identifier: '', name: '', dataType: { type: 'double' } }] },
      }
    default:
      return { type }
  }
}

function validateDataTypeSpec(
  detail: ThingDataTypeDetail,
  path: string,
  issues: FieldIssue[],
): void {
  const push = (message: string, suffix = '') =>
    issues.push({ field: `${path}${suffix}`, message })

  switch (detail.type) {
    case 'int':
    case 'long':
    case 'float':
    case 'double': {
      const min = detail.min?.trim()
      const max = detail.max?.trim()
      if (min && !Number.isFinite(Number(min))) push('最小值必须是数字', '.min')
      if (max && !Number.isFinite(Number(max))) push('最大值必须是数字', '.max')
      if (min && max && Number.isFinite(Number(min)) && Number.isFinite(Number(max))) {
        if (Number(min) > Number(max)) push('最小值不能大于最大值', '.min')
      }
      break
    }
    case 'enum': {
      const values = detail.enumSpec?.values ?? []
      if (values.length === 0) push('枚举至少需要一个取值', '.enumSpec.values')
      const seen = new Set<number>()
      values.forEach((item, index) => {
        if (!Number.isInteger(item.value)) {
          push('枚举取值必须是整数', `.enumSpec.values[${index}].value`)
        } else if (seen.has(item.value)) {
          push('枚举取值不能重复', `.enumSpec.values[${index}].value`)
        }
        seen.add(item.value)
      })
      break
    }
    case 'text': {
      const length = detail.length?.trim()
      if (!length) {
        push('文本长度不能为空', '.length')
      } else if (!/^[1-9]\d*$/.test(length)) {
        push('文本长度必须是正整数', '.length')
      }
      break
    }
    case 'array': {
      const size = detail.arraySpec?.size?.trim()
      if (size) {
        if (!/^[1-9]\d*$/.test(size) || Number(size) > 512) {
          push('数组元素个数必须是 1~512 的整数', '.arraySpec.size')
        }
      }
      if (!detail.arraySpec?.element) {
        push('数组元素类型不能为空', '.arraySpec.element')
      } else {
        validateDataTypeSpec(detail.arraySpec.element, `${path}.arraySpec.element`, issues)
      }
      break
    }
    case 'struct': {
      const fields = detail.structSpec?.fields ?? []
      if (fields.length === 0) push('结构体至少需要一个字段', '.structSpec.fields')
      const identifiers = new Set<string>()
      fields.forEach((field, index) => {
        const base = `${path}.structSpec.fields[${index}]`
        if (!IDENTIFIER_PATTERN.test(field.identifier.trim())) {
          issues.push({
            field: `${base}.identifier`,
            message: '字段标识符只能由字母、数字和下划线组成,且必须以字母开头',
          })
        } else if (identifiers.has(field.identifier.trim())) {
          issues.push({ field: `${base}.identifier`, message: '字段标识符不能重复' })
        }
        identifiers.add(field.identifier.trim())
        if (!field.name.trim()) {
          issues.push({ field: `${base}.name`, message: '字段名称不能为空' })
        }
        validateDataTypeSpec(field.dataType, `${base}.dataType`, issues)
      })
      break
    }
    default:
      break
  }
}

/** 校验属性或特征入参,返回与后端一致的字段路径。 */
export function validatePropertyInput(
  input: ThingModelPropertyInput | ThingModelFeature,
  options: { prefix?: string; requireRW?: boolean } = {},
): FieldIssue[] {
  const issues: FieldIssue[] = []
  const prefix = options.prefix ?? ''

  const identifier = input.identifier.trim()
  if (!IDENTIFIER_PATTERN.test(identifier)) {
    issues.push({
      field: `${prefix}identifier`,
      message: '标识符只能由字母、数字和下划线组成,且必须以字母开头',
    })
  } else if (identifier.length > 64) {
    issues.push({ field: `${prefix}identifier`, message: '标识符不能超过 64 个字符' })
  } else if (RESERVED_IDENTIFIERS.has(identifier)) {
    issues.push({ field: `${prefix}identifier`, message: `标识符不能使用保留字 ${identifier}` })
  }

  const name = input.name.trim()
  if (!name) {
    issues.push({ field: `${prefix}name`, message: '名称不能为空' })
  } else if (!/^[A-Za-z]/.test(name)) {
    issues.push({ field: `${prefix}name`, message: '名称必须以字母开头' })
  } else if (name.length > 64) {
    issues.push({ field: `${prefix}name`, message: '名称不能超过 64 个字符' })
  }

  if ((input.description ?? '').length > 255) {
    issues.push({ field: `${prefix}description`, message: '描述不能超过 255 个字符' })
  }

  if (options.requireRW !== false) {
    const property = input as ThingModelPropertyInput
    if (!['r', 'w', 'rw'].includes(property.rwFlag ?? 'r')) {
      issues.push({ field: `${prefix}rwFlag`, message: '读写标记必须是 r、w 或 rw' })
    }
    if ((property.sort ?? 0) < 0 || !Number.isInteger(property.sort ?? 0)) {
      issues.push({ field: `${prefix}sort`, message: '排序必须是非负整数' })
    }
  }

  validateDataTypeSpec(input.dataType, `${prefix}dataType`, issues)
  return issues
}

/** 简洁展示 dataType,用于表格与详情。 */
export function describeDataType(detail: ThingDataTypeDetail): string {
  const unit = detail.unit ? ` · ${detail.unit}` : ''
  switch (detail.type) {
    case 'int':
    case 'long':
    case 'float':
    case 'double': {
      const range =
        detail.min && detail.max
          ? ` [${detail.min}, ${detail.max}]`
          : detail.min
            ? ` ≥ ${detail.min}`
            : detail.max
              ? ` ≤ ${detail.max}`
              : ''
      return `${detail.type}${unit}${range}`
    }
    case 'bool':
      return `bool (${detail.trueValue ?? 'true'} / ${detail.falseValue ?? 'false'})`
    case 'enum':
      return `enum (${(detail.enumSpec?.values ?? [])
        .map((item) => `${item.desc || item.value}=${item.value}`)
        .join(', ')})`
    case 'text':
      return `text (长度 ${detail.length ?? '—'})`
    case 'array': {
      const size = detail.arraySpec?.size ? `[${detail.arraySpec.size}]` : '[]'
      return `array${size}<${describeDataType(detail.arraySpec?.element ?? { type: 'double' })}>`
    }
    case 'struct':
      return `struct{${(detail.structSpec?.fields ?? [])
        .map((field) => `${field.name || field.identifier}:${field.dataType.type}`)
        .join(', ')}}`
    default:
      return detail.type
  }
}
