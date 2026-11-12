import { useMemo, useState } from 'react'
import {
  Braces,
  Cpu,
  Layers,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  ConfirmDialog,
  Dialog,
  EmptyState,
  ErrorState,
  Field,
  Input,
  PageHeader,
  Skeleton,
  Table,
  TBody,
  TD,
  TH,
  THead,
  TR,
  Textarea,
  errorMessage,
  formatDateTime,
  useI18n,
} from '@harns/core'
import {
  useCreateDriver,
  useDeleteDriver,
  useDriverGroups,
  useDriverSchema,
  useUpdateDriver,
} from '@/api/hooks'
import type { Driver, DriverPayload, DriverSchema } from '@/api/types'
import { driverModelLabel, driverTypeLabel } from '@/lib/constants'
import { sortedFields } from '@/lib/collection-form'
import { localized } from '@/lib/labels'

interface DriverForm {
  driverType: string
  model: string
  pluginName: string
  schemaText: string
}

const EMPTY_DRIVER: DriverForm = {
  driverType: '',
  model: '',
  pluginName: '',
  schemaText: JSON.stringify(
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
  ),
}

function driverToForm(driver: Driver): DriverForm {
  return {
    driverType: driver.driverType,
    model: driver.model,
    pluginName: driver.pluginName,
    schemaText: JSON.stringify(driver.schema, null, 2),
  }
}

function parseSchema(text: string): DriverSchema | undefined {
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

export function DriversPage() {
  const { t } = useI18n()
  const groups = useDriverGroups()
  const [keyword, setKeyword] = useState('')
  const [schemaTarget, setSchemaTarget] = useState<Driver | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<Driver | null>(null)
  const [form, setForm] = useState<DriverForm>(EMPTY_DRIVER)
  const [formError, setFormError] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<Driver | null>(null)
  const create = useCreateDriver()
  const update = useUpdateDriver()
  const remove = useDeleteDriver()

  const filteredGroups = useMemo(() => {
    const query = keyword.trim().toLowerCase()
    const items = groups.data ?? []
    if (!query) return items
    return items
      .map((group) => ({
        ...group,
        models: group.models.filter((driver) =>
          [driver.model, driver.driverType, driver.pluginName]
            .join(' ')
            .toLowerCase()
            .includes(query),
        ),
      }))
      .filter((group) => group.models.length > 0)
  }, [groups.data, keyword])

  const openCreate = () => {
    setEditing(null)
    setForm(EMPTY_DRIVER)
    setFormError('')
    setFormOpen(true)
  }

  const openEdit = (driver: Driver) => {
    setEditing(driver)
    setForm(driverToForm(driver))
    setFormError('')
    setFormOpen(true)
  }

  const closeForm = () => {
    setFormOpen(false)
    setFormError('')
  }

  const submit = async () => {
    if (!form.driverType.trim() || !form.model.trim()) {
      setFormError(t('gateway.drivers.validation.identityRequired'))
      return
    }
    const schema = parseSchema(form.schemaText)
    if (!schema) {
      setFormError(t('gateway.drivers.validation.schemaInvalid'))
      return
    }
    const payload: DriverPayload = {
      driverType: form.driverType.trim(),
      model: form.model.trim(),
      pluginName: form.pluginName.trim(),
      schema,
    }
    try {
      if (editing) {
        await update.mutateAsync({ id: editing.id, payload })
        toast.success(t('gateway.drivers.toast.updated'))
      } else {
        await create.mutateAsync(payload)
        toast.success(t('gateway.drivers.toast.created'))
      }
      closeForm()
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await remove.mutateAsync({
        id: deleteTarget.id,
        title: `${deleteTarget.driverType}/${deleteTarget.model}`,
      })
      toast.success(t('gateway.drivers.toast.deleted'))
      setDeleteTarget(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const total = useMemo(
    () => (groups.data ?? []).reduce((sum, group) => sum + group.models.length, 0),
    [groups.data],
  )

  return (
    <div className="space-y-3">
      <PageHeader
        title={t('gateway.drivers.title')}
        description={t('gateway.drivers.description')}
        actions={
          <>
            <div className="relative">
              <Search className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-faint" />
              <Input
                value={keyword}
                onChange={(event) => setKeyword(event.target.value)}
                placeholder={t('gateway.drivers.searchPlaceholder')}
                aria-label={t('common.action.search')}
                className="w-56 pl-8"
              />
            </div>
            <Button
              variant="outline"
              loading={groups.isFetching}
              onClick={() => void groups.refetch()}
              icon={<RefreshCw className="size-3.5" />}
            >
              {t('common.action.refresh')}
            </Button>
            <Button variant="primary" onClick={openCreate} icon={<Plus className="size-3.5" />}>
              {t('gateway.drivers.create')}
            </Button>
          </>
        }
      />

      {groups.isLoading ? (
        <div className="grid gap-3 lg:grid-cols-2">
          {Array.from({ length: 2 }).map((_, index) => (
            <Card key={index}>
              <CardHeader>
                <Skeleton className="h-5 w-32" />
              </CardHeader>
              <CardContent className="space-y-2">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : groups.isError ? (
        <ErrorState error={groups.error} onRetry={() => void groups.refetch()} />
      ) : filteredGroups.length === 0 ? (
        <Card>
          <EmptyState
            icon={<Layers className="size-5" />}
            title={
              keyword
                ? t('gateway.drivers.empty.filteredTitle')
                : t('gateway.drivers.empty.title')
            }
            description={t('gateway.drivers.empty.description')}
            action={
              <Button variant="primary" onClick={openCreate} icon={<Plus className="size-3.5" />}>
                {t('gateway.drivers.create')}
              </Button>
            }
          />
        </Card>
      ) : (
        <div className="grid items-start gap-3 lg:grid-cols-2">
          {filteredGroups.map((group) => (
            <Card key={group.driverType}>
              <CardHeader>
                <div>
                  <CardTitle className="flex items-center gap-1.5">
                    <Cpu className="size-4 text-faint" />
                    {driverTypeLabel(group.driverType)}
                  </CardTitle>
                  <CardDescription>
                    {t('gateway.drivers.groupDescription', { count: group.models.length })}
                  </CardDescription>
                </div>
                <Badge tone="neutral">{group.models.length}</Badge>
              </CardHeader>
              <CardContent className="space-y-2">
                {group.models.map((driver) => (
                  <div
                    key={driver.id}
                    className="rounded border border-border px-3 py-2 transition-colors hover:border-border-strong"
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div className="min-w-0">
                        <div className="flex items-center gap-1.5">
                          <span className="truncate text-[13px] font-medium text-fg">
                            {driverModelLabel(driver.model)}
                          </span>
                          {driver.builtinDriver ? (
                            <Badge tone="info">{t('gateway.drivers.builtin')}</Badge>
                          ) : (
                            <Badge tone="violet">{t('gateway.drivers.custom')}</Badge>
                          )}
                        </div>
                        <p className="mt-0.5 truncate font-mono text-[10px] text-faint">
                          {driver.driverType}/{driver.model} · {driver.pluginName || '—'}
                        </p>
                      </div>
                      <div className="flex shrink-0 items-center gap-0.5">
                        <button
                          type="button"
                          onClick={() => setSchemaTarget(driver)}
                          className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-surface-2 hover:text-fg"
                          title={t('gateway.drivers.viewSchema')}
                          aria-label={t('gateway.drivers.viewSchema')}
                        >
                          <Braces className="size-3.5" />
                        </button>
                        {!driver.builtinDriver ? (
                          <>
                            <button
                              type="button"
                              onClick={() => openEdit(driver)}
                              className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-surface-2 hover:text-fg"
                              title={t('common.action.edit')}
                              aria-label={t('common.action.edit')}
                            >
                              <Pencil className="size-3.5" />
                            </button>
                            <button
                              type="button"
                              onClick={() => setDeleteTarget(driver)}
                              className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-danger/10 hover:text-danger"
                              title={t('common.action.delete')}
                              aria-label={t('common.action.delete')}
                            >
                              <Trash2 className="size-3.5" />
                            </button>
                          </>
                        ) : null}
                      </div>
                    </div>
                    <p className="mt-1 text-[10px] text-faint">
                      {t('gateway.drivers.createdAt')} {formatDateTime(driver.createdTime)} ·{' '}
                      {t('gateway.drivers.fieldCount', {
                        count: driver.schema?.fields?.length ?? 0,
                      })}
                    </p>
                  </div>
                ))}
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <p className="px-0.5 text-[11px] text-faint">
        {t('gateway.drivers.total', { count: total })}
      </p>

      <SchemaDialog driver={schemaTarget} onClose={() => setSchemaTarget(null)} />

      <Dialog
        open={formOpen}
        onClose={closeForm}
        title={editing ? t('gateway.drivers.editTitle') : t('gateway.drivers.createTitle')}
        description={t('gateway.drivers.formDescription')}
        size="xl"
        footer={
          <>
            <Button variant="ghost" onClick={closeForm}>
              {t('common.action.cancel')}
            </Button>
            <Button
              variant="primary"
              loading={create.isPending || update.isPending}
              onClick={() => void submit()}
            >
              {t('common.action.save')}
            </Button>
          </>
        }
      >
        <div className="space-y-3">
          <div className="grid gap-3 sm:grid-cols-3">
            <Field label={t('gateway.drivers.field.driverType')} required>
              <Input
                value={form.driverType}
                disabled={Boolean(editing)}
                placeholder="modbus"
                autoComplete="off"
                onChange={(event) => setForm({ ...form, driverType: event.target.value })}
              />
            </Field>
            <Field label={t('gateway.drivers.field.model')} required>
              <Input
                value={form.model}
                disabled={Boolean(editing)}
                placeholder="myModbusTcp"
                autoComplete="off"
                onChange={(event) => setForm({ ...form, model: event.target.value })}
              />
            </Field>
            <Field label={t('gateway.drivers.field.pluginName')}>
              <Input
                value={form.pluginName}
                placeholder="my-driver-plugin"
                autoComplete="off"
                onChange={(event) => setForm({ ...form, pluginName: event.target.value })}
              />
            </Field>
          </div>
          <Field
            label={t('gateway.drivers.field.schema')}
            hint={t('gateway.drivers.field.schemaHint')}
            error={formError}
          >
            <Textarea
              value={form.schemaText}
              spellCheck={false}
              className="min-h-72 font-mono text-xs"
              onChange={(event) => setForm({ ...form, schemaText: event.target.value })}
            />
          </Field>
        </div>
      </Dialog>

      <ConfirmDialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => void confirmDelete()}
        loading={remove.isPending}
        danger
        title={t('gateway.drivers.deleteTitle')}
        description={t('gateway.drivers.deleteDescription', {
          name: deleteTarget
            ? `${deleteTarget.driverType}/${deleteTarget.model}`
            : '',
        })}
        confirmText={t('common.action.delete')}
      />
    </div>
  )
}

function SchemaDialog({ driver, onClose }: { driver: Driver | null; onClose: () => void }) {
  const { t, locale } = useI18n()
  const schemaQuery = useDriverSchema(driver?.model)
  const schema = schemaQuery.data?.schema ?? driver?.schema

  return (
    <Dialog
      open={driver !== null}
      onClose={onClose}
      title={
        driver
          ? `${driverTypeLabel(driver.driverType)} · ${driverModelLabel(driver.model)}`
          : ''
      }
      description={t('gateway.drivers.schemaDescription')}
      size="xl"
      footer={
        <Button variant="ghost" onClick={onClose}>
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
        <div className="space-y-4">
          <section>
            <h3 className="mb-1.5 text-xs font-medium text-muted">
              {t('gateway.drivers.schema.fields')}
            </h3>
            {sortedFields(schema).length === 0 ? (
              <p className="text-xs text-faint">{t('gateway.drivers.schema.empty')}</p>
            ) : (
              <div className="overflow-hidden rounded border border-border">
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
                        <TD className="font-mono text-[11px]">{field.name}</TD>
                        <TD className="text-xs">
                          {localized(field.labelZh, field.labelEn, locale)}
                        </TD>
                        <TD>
                          <Badge tone={field.type === 'select' ? 'info' : 'neutral'}>
                            {field.type}
                          </Badge>
                          {field.options?.length ? (
                            <span className="ml-1 text-[10px] text-faint">
                              {field.options.map((option) => option.value).join(' / ')}
                            </span>
                          ) : null}
                        </TD>
                        <TD className="text-xs">{field.required ? t('common.state.yes') : '—'}</TD>
                        <TD className="max-w-56 truncate font-mono text-[10px] text-faint">
                          {field.regex || '—'}
                        </TD>
                      </TR>
                    ))}
                  </TBody>
                </Table>
              </div>
            )}
          </section>

          <section>
            <h3 className="mb-1.5 text-xs font-medium text-muted">
              {t('gateway.drivers.schema.variableTypes')}
            </h3>
            {(schema.variableDataTypes ?? []).length === 0 ? (
              <p className="text-xs text-faint">{t('gateway.drivers.schema.empty')}</p>
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
              <h3 className="mb-1.5 text-xs font-medium text-muted">
                {t('gateway.drivers.schema.variableAddress')}
              </h3>
              <div className="rounded border border-border bg-surface-2 px-3 py-2 text-xs">
                <p className="font-mono text-fg">
                  {localized(
                    schema.variableAddress.labelZh,
                    schema.variableAddress.labelEn,
                    locale,
                  )}
                </p>
                <p className="mt-1 font-mono text-[10px] text-faint">
                  {localized(
                    schema.variableAddress.placeholderZh,
                    schema.variableAddress.placeholderEn,
                    locale,
                  ) || '—'}
                </p>
                <p className="mt-1 text-[11px] text-muted">
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
    </Dialog>
  )
}
