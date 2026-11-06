import type {
  ThingDataType,
  ThingDataTypeDetail,
  ThingModelFeature,
  ThingModelPropertyInput,
} from '@/api/types'
import { translate } from '@/i18n'
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
      if (min && !Number.isFinite(Number(min))) {
        push(translate('propertyDialog.validation.minNumber'), '.min')
      }
      if (max && !Number.isFinite(Number(max))) {
        push(translate('propertyDialog.validation.maxNumber'), '.max')
      }
      if (min && max && Number.isFinite(Number(min)) && Number.isFinite(Number(max))) {
        if (Number(min) > Number(max)) {
          push(translate('propertyDialog.validation.minGreaterMax'), '.min')
        }
      }
      break
    }
    case 'enum': {
      const values = detail.enumSpec?.values ?? []
      if (values.length === 0) {
        push(translate('propertyDialog.validation.enumRequired'), '.enumSpec.values')
      }
      const seen = new Set<number>()
      values.forEach((item, index) => {
        if (!Number.isInteger(item.value)) {
          push(
            translate('propertyDialog.validation.enumInteger'),
            `.enumSpec.values[${index}].value`,
          )
        } else if (seen.has(item.value)) {
          push(
            translate('propertyDialog.validation.enumDuplicate'),
            `.enumSpec.values[${index}].value`,
          )
        }
        seen.add(item.value)
      })
      break
    }
    case 'text': {
      const length = detail.length?.trim()
      if (!length) {
        push(translate('propertyDialog.validation.textLengthRequired'), '.length')
      } else if (!/^[1-9]\d*$/.test(length)) {
        push(translate('propertyDialog.validation.textLengthPositive'), '.length')
      }
      break
    }
    case 'array': {
      const size = detail.arraySpec?.size?.trim()
      if (size) {
        if (!/^[1-9]\d*$/.test(size) || Number(size) > 512) {
          push(translate('propertyDialog.validation.arraySize'), '.arraySpec.size')
        }
      }
      if (!detail.arraySpec?.element) {
        push(translate('propertyDialog.validation.arrayElement'), '.arraySpec.element')
      } else {
        validateDataTypeSpec(detail.arraySpec.element, `${path}.arraySpec.element`, issues)
      }
      break
    }
    case 'struct': {
      const fields = detail.structSpec?.fields ?? []
      if (fields.length === 0) {
        push(translate('propertyDialog.validation.structFields'), '.structSpec.fields')
      }
      const identifiers = new Set<string>()
      fields.forEach((field, index) => {
        const base = `${path}.structSpec.fields[${index}]`
        if (!IDENTIFIER_PATTERN.test(field.identifier.trim())) {
          issues.push({
            field: `${base}.identifier`,
            message: translate('propertyDialog.validation.structIdentifier'),
          })
        } else if (identifiers.has(field.identifier.trim())) {
          issues.push({
            field: `${base}.identifier`,
            message: translate('propertyDialog.validation.structIdentifierDuplicate'),
          })
        }
        identifiers.add(field.identifier.trim())
        if (!field.name.trim()) {
          issues.push({
            field: `${base}.name`,
            message: translate('propertyDialog.validation.structName'),
          })
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
      message: translate('propertyDialog.validation.identifierInvalid'),
    })
  } else if (identifier.length > 64) {
    issues.push({
      field: `${prefix}identifier`,
      message: translate('propertyDialog.validation.identifierMax'),
    })
  } else if (RESERVED_IDENTIFIERS.has(identifier)) {
    issues.push({
      field: `${prefix}identifier`,
      message: translate('propertyDialog.validation.identifierReserved', { identifier }),
    })
  }

  const name = input.name.trim()
  if (!name) {
    issues.push({
      field: `${prefix}name`,
      message: translate('propertyDialog.validation.nameRequired'),
    })
  } else if (!/^[A-Za-z]/.test(name)) {
    issues.push({
      field: `${prefix}name`,
      message: translate('propertyDialog.validation.nameStart'),
    })
  } else if (name.length > 64) {
    issues.push({
      field: `${prefix}name`,
      message: translate('propertyDialog.validation.nameMax'),
    })
  }

  if ((input.description ?? '').length > 255) {
    issues.push({
      field: `${prefix}description`,
      message: translate('propertyDialog.validation.descriptionMax'),
    })
  }

  if (options.requireRW !== false) {
    const property = input as ThingModelPropertyInput
    if (!['r', 'w', 'rw'].includes(property.rwFlag ?? 'r')) {
      issues.push({
        field: `${prefix}rwFlag`,
        message: translate('propertyDialog.validation.rwInvalid'),
      })
    }
    if ((property.sort ?? 0) < 0 || !Number.isInteger(property.sort ?? 0)) {
      issues.push({
        field: `${prefix}sort`,
        message: translate('propertyDialog.validation.sortInvalid'),
      })
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
      return detail.length
        ? translate('dataType.describeText', { length: detail.length })
        : translate('dataType.describeTextEmpty')
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
