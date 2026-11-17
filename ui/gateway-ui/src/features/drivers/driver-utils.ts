import type { DriverSchema } from '@/api/types'

export const EMPTY_DRIVER_SCHEMA_TEXT = JSON.stringify(
  {
    fields: [
      {
        name: 'location',
        labelZh: '设备地址',
        labelEn: 'Address',
        regex: '^.+$',
        sort: 1,
        type: 'input',
        required: true,
      },
    ],
    variableAddress: {
      labelZh: '变量地址',
      labelEn: 'Variable address',
      placeholderZh: '例如 40001',
      placeholderEn: 'e.g. 40001',
    },
    variableDataTypes: [
      {
        value: 'uint16',
        labelZh: '无符号 16 位',
        labelEn: 'Unsigned 16-bit',
        addressRegex: '^\\d+$',
      },
    ],
  },
  null,
  2,
)

export function parseDriverSchema(text: string): DriverSchema | undefined {
  try {
    const parsed: unknown = JSON.parse(text)
    if (!parsed || typeof parsed !== 'object') return undefined
    const schema = parsed as DriverSchema
    if (!Array.isArray(schema.fields)) return undefined
    return schema
  } catch {
    return undefined
  }
}
