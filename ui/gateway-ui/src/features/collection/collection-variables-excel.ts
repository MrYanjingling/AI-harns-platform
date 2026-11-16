import type { Cell } from 'write-excel-file/browser'
import type { Translate } from '@harns/core'
import type { DriverSchema } from '@/api/types'
import type { Locale } from '@harns/core'
import { ACCESS_MODES } from '@/lib/constants'
import { createEmptyVariable, variableTypes, type VariableFormValues } from '@/lib/collection-form'
import { localized } from '@/lib/labels'

type HeaderKey =
  | 'name'
  | 'dataType'
  | 'address'
  | 'bits'
  | 'precision'
  | 'rate'
  | 'offset'
  | 'accessMode'

interface HeaderDefinition {
  key: HeaderKey
  label: string
  aliases: string[]
  width: number
}

function headerDefinitions(
  schema: DriverSchema,
  locale: Locale,
  t: Translate,
): HeaderDefinition[] {
  const addressLabel = schema.variableAddress
    ? localized(schema.variableAddress.labelZh, schema.variableAddress.labelEn, locale)
    : ''
  return [
    {
      key: 'name',
      label: t('gateway.variable.name'),
      aliases: ['变量名', 'name', 'variable name'],
      width: 20,
    },
    {
      key: 'dataType',
      label: t('gateway.variable.dataType'),
      aliases: ['数据类型', 'data type', 'datatype', 'type'],
      width: 18,
    },
    {
      key: 'address',
      label: t('gateway.variable.template.address'),
      aliases: ['变量地址', '地址', 'address', 'variable address', addressLabel],
      width: 20,
    },
    {
      key: 'bits',
      label: t('gateway.variable.bits'),
      aliases: ['位', 'bit', 'bits'],
      width: 10,
    },
    {
      key: 'precision',
      label: t('gateway.variable.precision'),
      aliases: ['精度', '小数位', 'precision'],
      width: 10,
    },
    {
      key: 'rate',
      label: t('gateway.variable.rate'),
      aliases: ['倍率', 'rate'],
      width: 10,
    },
    {
      key: 'offset',
      label: t('gateway.variable.offset'),
      aliases: ['偏移', 'offset'],
      width: 10,
    },
    {
      key: 'accessMode',
      label: t('gateway.variable.accessMode'),
      aliases: ['读写', '读写属性', 'access', 'access mode', 'accessmode'],
      width: 12,
    },
  ]
}

function normalizeHeader(value: unknown): string {
  return String(value ?? '').trim().toLowerCase()
}

function cellString(value: unknown): string {
  if (value === null || value === undefined) return ''
  return String(value).trim()
}

function accessModeAliases(t: Translate): Record<string, VariableFormValues['accessMode']> {
  const aliases: Record<string, VariableFormValues['accessMode']> = {}
  for (const mode of ACCESS_MODES) {
    aliases[normalizeHeader(mode.value)] = mode.value
    aliases[normalizeHeader(t(mode.labelKey))] = mode.value
  }
  aliases['只读'] = 'r'
  aliases['只写'] = 'w'
  aliases['读写'] = 'rw'
  aliases['read'] = 'r'
  aliases['write'] = 'w'
  aliases['read/write'] = 'rw'
  aliases['readwrite'] = 'rw'
  return aliases
}

export async function exportCollectionVariablesTemplate({
  schema,
  locale,
  t,
}: {
  schema: DriverSchema
  locale: Locale
  t: Translate
}): Promise<void> {
  const { default: writeXlsxFile } = await import('write-excel-file/browser')
  const headers = headerDefinitions(schema, locale, t)
  const types = variableTypes(schema)
  const exampleType = types[0]
  const exampleAddress = schema.variableAddress
    ? localized(
        schema.variableAddress.placeholderZh,
        schema.variableAddress.placeholderEn,
        locale,
      )
        .replace(/^(例如|e\.g\.)\s*/i, '')
        .trim()
    : ''
  const sampleValues = [
    'example_voltage',
    exampleType ? localized(exampleType.labelZh, exampleType.labelEn, locale) || exampleType.value : '',
    exampleAddress || 'example_address',
    '',
    '',
    '1',
    '0',
    t('gateway.variable.accessMode.r'),
  ]

  const headerCells: Cell[] = headers.map((header) => ({
    value: header.label,
    fontWeight: 'bold',
    backgroundColor: '#F1F5F9',
  }))
  const sampleCells: Cell[] = sampleValues.map((value) => ({ value }))

  await writeXlsxFile([headerCells, sampleCells], {
    sheet: t('gateway.variable.template.sheet'),
    columns: headers.map((header) => ({ width: header.width })),
  }).toFile(`${t('gateway.variable.template.fileName')}.xlsx`)
}

export async function importCollectionVariablesFile({
  file,
  schema,
  locale,
  t,
}: {
  file: File
  schema: DriverSchema
  locale: Locale
  t: Translate
}): Promise<VariableFormValues[]> {
  const { readSheet } = await import('read-excel-file/browser')
  const rows = await readSheet(file)
  if (rows.length === 0) {
    throw new Error(t('gateway.variable.importEmpty'))
  }

  const headers = headerDefinitions(schema, locale, t)
  const headerRow = rows[0] ?? []
  const indices = new Map<HeaderKey, number>()
  for (const definition of headers) {
    const aliases = new Set(
      [definition.label, ...definition.aliases]
        .map((alias) => normalizeHeader(alias))
        .filter(Boolean),
    )
    const index = headerRow.findIndex((cell) => aliases.has(normalizeHeader(cell)))
    if (index !== -1) indices.set(definition.key, index)
  }

  const missing = headers
    .filter((header) => ['name', 'dataType', 'address'].includes(header.key))
    .filter((header) => !indices.has(header.key))
    .map((header) => header.label)
  if (missing.length > 0) {
    throw new Error(
      t('gateway.variable.importMissingHeader', { headers: missing.join('、') }),
    )
  }

  const typeAliases = new Map<string, string>()
  for (const type of variableTypes(schema)) {
    typeAliases.set(normalizeHeader(type.value), type.value)
    typeAliases.set(normalizeHeader(type.labelZh), type.value)
    typeAliases.set(normalizeHeader(type.labelEn), type.value)
  }
  const accessAliases = accessModeAliases(t)

  const imported: VariableFormValues[] = []
  for (let index = 1; index < rows.length; index += 1) {
    const row = rows[index] ?? []
    if (row.every((cell) => cellString(cell) === '')) continue
    const value = (key: HeaderKey) => {
      const column = indices.get(key)
      return column === undefined ? '' : cellString(row[column])
    }
    const accessInput = value('accessMode')
    const accessMode = accessInput
      ? accessAliases[normalizeHeader(accessInput)]
      : 'r'
    if (!accessMode) {
      throw new Error(
        t('gateway.variable.importRowAccessInvalid', { row: index + 1 }),
      )
    }
    const dataTypeInput = value('dataType')
    imported.push({
      ...createEmptyVariable(schema),
      name: value('name'),
      dataType: typeAliases.get(normalizeHeader(dataTypeInput)) ?? dataTypeInput,
      address: value('address'),
      bits: value('bits'),
      precision: value('precision'),
      rate: value('rate'),
      offset: value('offset'),
      accessMode,
      isCalculated: false,
    })
  }

  if (imported.length === 0) {
    throw new Error(t('gateway.variable.importEmpty'))
  }
  return imported
}
