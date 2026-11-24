import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import {
  Badge,
  Button,
  FormDialog,
  Skeleton,
  Table,
  Tabs,
  TBody,
  TD,
  Textarea,
  TH,
  THead,
  TR,
  showErrorToast,
  useI18n,
} from '@harns/core'
import { useDriverSchema, useUpdateDriver } from '@/api/hooks'
import type { Driver } from '@/api/types'
import { sortedFields } from '@/lib/collection-form'
import { driverModelLabel, driverTypeLabel } from '@/lib/constants'
import { localized } from '@/lib/labels'
import { parseDriverSchema } from './driver-utils'

export function DriverSchemaDialog({
  driver,
  onClose,
}: {
  driver: Driver | null
  onClose: () => void
}) {
  const { t, locale } = useI18n()
  const schemaQuery = useDriverSchema(driver?.model)
  const schema = schemaQuery.data?.schema ?? driver?.schema
  const update = useUpdateDriver()
  const [view, setView] = useState<'table' | 'json'>('table')
  const [editing, setEditing] = useState(false)
  const [schemaText, setSchemaText] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    setView('table')
    setEditing(false)
    setError('')
  }, [driver?.id])

  const startEditing = () => {
    if (!schema || driver?.builtinDriver) return
    setSchemaText(JSON.stringify(schema, null, 2))
    setError('')
    setEditing(true)
  }

  const saveSchema = async () => {
    if (!driver || driver.builtinDriver) return
    const nextSchema = parseDriverSchema(schemaText)
    if (!nextSchema) {
      setError(t('gateway.drivers.validation.schemaInvalid'))
      return
    }
    try {
      const updated = await update.mutateAsync({
        id: driver.id,
        payload: {
          driverType: driver.driverType,
          model: driver.model,
          pluginName: driver.pluginName,
          remark: driver.remark,
          schema: nextSchema,
        },
      })
      setSchemaText(JSON.stringify(updated.schema ?? nextSchema, null, 2))
      setEditing(false)
      setError('')
      toast.success(t('gateway.drivers.toast.schemaUpdated'))
      onClose()
    } catch (err) {
      showErrorToast(err)
    }
  }

  return (
    <FormDialog
      open={driver !== null}
      onClose={onClose}
      title={
        driver
          ? `${driverTypeLabel(driver.driverType)} · ${driverModelLabel(driver.model)}`
          : ''
      }
      footer={
        <Button variant="outline" onClick={onClose}>
          {t('common.action.close')}
        </Button>
      }
    >
      {!schema ? (
        <div className="space-y-2">
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-24 w-full" />
        </div>
      ) : (
        <div className="space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <Tabs
              items={[
                { value: 'table', label: t('gateway.drivers.schema.viewTable') },
                { value: 'json', label: t('gateway.drivers.schema.viewJson') },
              ]}
              value={view}
              onChange={setView}
            />
            {view === 'json' ? (
              <div className="ml-auto flex items-center gap-2">
                {editing ? (
                  <>
                    <Button
                      variant="outline"
                      className="h-8"
                      onClick={() => {
                        setEditing(false)
                        setError('')
                      }}
                    >
                      {t('common.action.cancel')}
                    </Button>
                    <Button
                      variant="primary"
                      className="h-8"
                      loading={update.isPending}
                      onClick={() => void saveSchema()}
                    >
                      {t('common.action.save')}
                    </Button>
                  </>
                ) : !driver?.builtinDriver ? (
                  <Button variant="outline" className="h-8" onClick={startEditing}>
                    {t('gateway.drivers.schema.edit')}
                  </Button>
                ) : null}
              </div>
            ) : null}
          </div>
          {view === 'json' ? (
            <div className="space-y-2">
              {editing ? (
                <Textarea
                  value={schemaText}
                  spellCheck={false}
                  className="min-h-[420px] font-mono text-xs select-text"
                  aria-invalid={Boolean(error)}
                  onChange={(event) => {
                    setSchemaText(event.target.value)
                    setError('')
                  }}
                />
              ) : (
                <pre className="max-h-[60dvh] cursor-text select-text overflow-auto rounded border border-border bg-surface p-3 font-mono text-xs leading-relaxed whitespace-pre-wrap break-all text-fg">
                  {JSON.stringify(schema, null, 2)}
                </pre>
              )}
              <p className="app-dialog-error">{error}</p>
            </div>
          ) : (
        <div className="space-y-4">
          <section>
            <h3 className="mb-1.5 text-sm font-medium text-muted">
              {t('gateway.drivers.schema.fields')}
            </h3>
            {sortedFields(schema).length === 0 ? (
              <p className="text-sm text-faint">{t('gateway.drivers.schema.empty')}</p>
            ) : (
              <Table>
                <THead>
                  <TR>
                    <TH>{t('gateway.drivers.schema.name')}</TH>
                    <TH>{t('gateway.drivers.schema.label')}</TH>
                    <TH>{t('gateway.drivers.schema.type')}</TH>
                    <TH>{t('gateway.drivers.schema.required')}</TH>
                    <TH>{t('gateway.drivers.schema.regex')}</TH>
                  </TR>
                </THead>
                <TBody>
                  {sortedFields(schema).map((field) => (
                    <TR key={field.name}>
                      <TD className="max-w-[220px] truncate text-fg">{field.name}</TD>
                      <TD className="truncate">
                        {localized(field.labelZh, field.labelEn, locale)}
                      </TD>
                      <TD>
                        <Badge tone={field.type === 'select' ? 'info' : 'neutral'}>
                          {field.type}
                        </Badge>
                        {field.options?.length ? (
                          <span className="ml-1 text-muted">
                            {field.options.map((option) => option.value).join(' / ')}
                          </span>
                        ) : null}
                      </TD>
                      <TD>{field.required ? t('common.state.yes') : '—'}</TD>
                      <TD className="max-w-56 truncate text-muted">
                        {field.regex || '—'}
                      </TD>
                    </TR>
                  ))}
                </TBody>
              </Table>
            )}
          </section>

          <section>
            <h3 className="mb-1.5 text-sm font-medium text-muted">
              {t('gateway.drivers.schema.variableTypes')}
            </h3>
            {(schema.variableDataTypes ?? []).length === 0 ? (
              <p className="text-sm text-faint">{t('gateway.drivers.schema.empty')}</p>
            ) : (
              <div className="flex flex-wrap gap-1.5">
                {(schema.variableDataTypes ?? []).map((type) => (
                  <Badge key={type.value} tone="neutral" className="font-mono">
                    {type.value}
                    <span className="ml-1 font-sans font-normal text-faint">
                      {localized(type.labelZh, type.labelEn, locale)}
                    </span>
                  </Badge>
                ))}
              </div>
            )}
          </section>

          {schema.variableAddress ? (
            <section>
              <h3 className="mb-1.5 text-sm font-medium text-muted">
                {t('gateway.drivers.schema.variableAddress')}
              </h3>
              <div className="rounded border border-border bg-surface-2 px-3 py-2 text-sm">
                <p className="font-mono text-fg">
                  {localized(
                    schema.variableAddress.labelZh,
                    schema.variableAddress.labelEn,
                    locale,
                  )}
                </p>
                <p className="mt-1 font-mono text-xs text-faint">
                  {localized(
                    schema.variableAddress.placeholderZh,
                    schema.variableAddress.placeholderEn,
                    locale,
                  ) || '—'}
                </p>
                <p className="mt-1 text-xs text-muted">
                  {localized(
                    schema.variableAddress.descriptionZh,
                    schema.variableAddress.descriptionEn,
                    locale,
                  )}
                </p>
              </div>
            </section>
          ) : null}
        </div>
          )}
        </div>
      )}
    </FormDialog>
  )
}
