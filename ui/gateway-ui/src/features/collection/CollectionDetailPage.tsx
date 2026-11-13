import { useMemo, useState, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  Pencil,
  Play,
  Plus,
  RotateCw,
  SlidersHorizontal,
  Square,
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
  CopyButton,
  Dialog,
  EmptyState,
  ErrorState,
  Field,
  Input,
  PageHeader,
  PageLoading,
  Skeleton,
  Switch,
  Table,
  TableSkeleton,
  TBody,
  TD,
  TH,
  THead,
  TR,
  errorMessage,
  formatDateTime,
  useI18n,
} from '@harns/core'
import {
  useAddCollectionVariables,
  useCollection,
  useCollectionRuntimeAction,
  useCollectionVariables,
  useDeleteCollection,
  useDeleteCollectionVariables,
  useDeliverCollectionAction,
  useDriverSchema,
  useSetCollectionEnable,
} from '@/api/hooks'
import type { DriverSchema } from '@/api/types'
import { CollectStatusBadge } from '@/components/StatusBadge'
import { ProtocolLabel } from '@/components/ProtocolLabel'
import { VariablesEditor } from './VariablesEditor'
import {
  createEmptyVariable,
  sortedFields,
  toVariableInputs,
  validateVariableRows,
  type VariableFormValues,
} from '@/lib/collection-form'
import { formatCycle } from '@/lib/constants'
import { localized } from '@/lib/labels'

type RuntimeAction = 'start' | 'stop' | 'restart'

function InfoItem({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="min-w-0">
      <p className="text-[11px] text-faint">{label}</p>
      <div className="mt-0.5 text-[13px] text-fg">{children}</div>
    </div>
  )
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

export function CollectionDetailPage() {
  const { t, locale } = useI18n()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const detail = useCollection(id)
  const schema = useDriverSchema(detail.data?.driverModel)
  const variables = useCollectionVariables(id, { page: 1, pageSize: 100 })
  const deleteCollection = useDeleteCollection()
  const runtimeAction = useCollectionRuntimeAction()
  const setEnable = useSetCollectionEnable()
  const addVariables = useAddCollectionVariables()
  const deleteVariables = useDeleteCollectionVariables()
  const deliver = useDeliverCollectionAction()

  const [deleteOpen, setDeleteOpen] = useState(false)
  const [addOpen, setAddOpen] = useState(false)
  const [controlOpen, setControlOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null)
  const [variableRows, setVariableRows] = useState<VariableFormValues[]>([])
  const [variableErrors, setVariableErrors] = useState<Record<string, string>>({})
  const [controlValues, setControlValues] = useState<Record<string, string>>({})

  const collection = detail.data
  const schemaData = schema.data?.schema
  const writableVariables = useMemo(
    () => (variables.data?.list ?? []).filter((item) => item.accessMode !== 'r' && !item.isCalculated),
    [variables.data],
  )

  const runAction = async (action: RuntimeAction) => {
    if (!collection) return
    try {
      await runtimeAction.mutateAsync({ id: collection.id, name: collection.name, action })
      toast.success(
        action === 'start'
          ? t('gateway.collection.toast.started')
          : action === 'stop'
            ? t('gateway.collection.toast.stopped')
            : t('gateway.collection.toast.restarted'),
      )
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const toggleEnable = async (enable: boolean) => {
    if (!collection) return
    try {
      await setEnable.mutateAsync({ id: collection.id, name: collection.name, enable })
      toast.success(
        enable ? t('gateway.collection.toast.enabled') : t('gateway.collection.toast.disabled'),
      )
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const openAddVariables = () => {
    if (!schemaData) return
    setVariableRows([createEmptyVariable(schemaData)])
    setVariableErrors({})
    setAddOpen(true)
  }

  const submitVariables = async () => {
    if (!collection || !schemaData) return
    const errors = validateVariableRows(variableRows, schemaData, t)
    setVariableErrors(errors)
    if (Object.keys(errors).length > 0) return
    try {
      await addVariables.mutateAsync({
        id: collection.id,
        name: collection.name,
        variables: toVariableInputs(variableRows),
      })
      toast.success(t('gateway.detail.variables.added'))
      setAddOpen(false)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const confirmDeleteVariable = async () => {
    if (!collection || !deleteTarget) return
    try {
      await deleteVariables.mutateAsync({
        id: collection.id,
        name: collection.name,
        names: [deleteTarget],
      })
      toast.success(t('gateway.detail.variables.deleted'))
      setDeleteTarget(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const openControl = () => {
    setControlValues({})
    setControlOpen(true)
  }

  const submitControl = async () => {
    if (!collection) return
    const actions = writableVariables
      .filter((item) => (controlValues[item.name] ?? '').trim() !== '')
      .map((item) => ({ name: item.name, value: controlValues[item.name].trim() }))
    if (actions.length === 0) return
    try {
      await deliver.mutateAsync({ id: collection.id, name: collection.name, actions })
      toast.success(t('gateway.detail.control.toast'))
      setControlOpen(false)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const confirmDelete = async () => {
    if (!collection) return
    try {
      await deleteCollection.mutateAsync({ id: collection.id, name: collection.name })
      toast.success(t('gateway.collection.toast.deleted'))
      navigate('/collections')
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  if (detail.isLoading) return <PageLoading />
  if (detail.isError) {
    return <ErrorState error={detail.error} onRetry={() => void detail.refetch()} />
  }
  if (!collection) {
    return (
      <EmptyState
        title={t('gateway.detail.notFound')}
        action={
          <Link
            to="/collections"
            className="inline-flex h-8 items-center rounded bg-primary px-3 text-sm font-medium text-primary-fg"
          >
            {t('gateway.form.back')}
          </Link>
        }
      />
    )
  }

  const variableList = variables.data?.list ?? []

  return (
    <div className="space-y-3">
      <PageHeader
        title={collection.name}
        description={collection.id}
        backTo="/collections"
        backLabel={t('gateway.detail.back')}
        actions={
          <>
            <Link
              to={`/collections/${collection.id}/edit`}
              className="inline-flex h-8 items-center gap-1.5 rounded border border-border-strong bg-surface px-3 text-sm text-fg transition-colors hover:bg-surface-2"
            >
              <Pencil className="size-3.5" />
              {t('common.action.edit')}
            </Link>
            {writableVariables.length > 0 ? (
              <Button
                variant="outline"
                onClick={openControl}
                disabled={collection.collectStatus !== 'collecting'}
                icon={<SlidersHorizontal className="size-3.5" />}
              >
                {t('gateway.detail.control.button')}
              </Button>
            ) : null}
            {collection.collectStatus === 'collecting' ? (
              <Button
                variant="outline"
                onClick={() => void runAction('stop')}
                icon={<Square className="size-3.5" />}
              >
                {t('common.action.stop')}
              </Button>
            ) : (
              <Button
                variant="outline"
                onClick={() => void runAction('start')}
                icon={<Play className="size-3.5" />}
              >
                {t('common.action.start')}
              </Button>
            )}
            <Button
              variant="outline"
              onClick={() => void runAction('restart')}
              icon={<RotateCw className="size-3.5" />}
            >
              {t('common.action.restart')}
            </Button>
            <Button
              variant="dangerSoft"
              onClick={() => setDeleteOpen(true)}
              icon={<Trash2 className="size-3.5" />}
            >
              {t('common.action.delete')}
            </Button>
          </>
        }
      />

      <Card>
        <CardHeader>
          <div>
            <CardTitle>{t('gateway.detail.section.identity')}</CardTitle>
            <CardDescription>{t('gateway.detail.identityHint')}</CardDescription>
          </div>
          <CollectStatusBadge status={collection.collectStatus} />
        </CardHeader>
        <CardContent className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <InfoItem label={t('gateway.collection.columns.protocol')}>
            <ProtocolLabel type={collection.driverType} model={collection.driverModel} />
          </InfoItem>
          <InfoItem label={t('gateway.collection.columns.cycle')}>
            <span className="tabular">{formatCycle(collection.collectorCycleMs)}</span>
          </InfoItem>
          <InfoItem label={t('gateway.detail.field.version')}>
            <span className="font-mono text-xs">{collection.version}</span>
          </InfoItem>
          <InfoItem label={t('gateway.collection.columns.modTime')}>
            {formatDateTime(collection.modTime)}
          </InfoItem>
          <InfoItem label={t('gateway.detail.field.id')}>
            <span className="flex items-center gap-1 font-mono text-[11px]">
              {collection.id}
              <CopyButton value={collection.id} />
            </span>
          </InfoItem>
          <InfoItem label={t('gateway.collection.columns.enable')}>
            <div className="flex items-center gap-2">
              <Switch
                checked={collection.enable}
                disabled={setEnable.isPending || (collection.enable && collection.collectStatus !== 'stopped')}
                label={t('gateway.collection.columns.enable')}
                onChange={(enable) => void toggleEnable(enable)}
              />
              <span className="text-[11px] text-muted">
                {collection.enable ? t('gateway.detail.enable.on') : t('gateway.detail.enable.off')}
              </span>
            </div>
          </InfoItem>
          <InfoItem label={t('gateway.detail.field.variableCount')}>
            <Badge tone="neutral">{variableList.length}</Badge>
          </InfoItem>
          <InfoItem label={t('gateway.detail.field.runtimeHint')}>
            <span className="text-[11px] text-muted">{t('gateway.detail.enable.hint')}</span>
          </InfoItem>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div>
            <CardTitle>{t('gateway.detail.section.protocol')}</CardTitle>
            <CardDescription>{t('gateway.detail.protocolHint')}</CardDescription>
          </div>
        </CardHeader>
        <CardContent>
          {schema.isLoading ? (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              {Array.from({ length: 4 }).map((_, index) => (
                <Skeleton key={index} className="h-10 w-full" />
              ))}
            </div>
          ) : schemaData ? (
            <ProtocolReadonly schema={schemaData} config={collection.protocolConfig} locale={locale} />
          ) : (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              {Object.entries(collection.protocolConfig ?? {}).map(([key, value]) => (
                <InfoItem key={key} label={key}>
                  <span className="font-mono text-xs">{formatValue(value)}</span>
                </InfoItem>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Card className="overflow-hidden">
        <CardHeader>
          <div>
            <CardTitle>{t('gateway.detail.section.variables')}</CardTitle>
            <CardDescription>{t('gateway.detail.variablesHint')}</CardDescription>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={openAddVariables}
            disabled={!schemaData}
            icon={<Plus className="size-3.5" />}
          >
            {t('gateway.variable.add')}
          </Button>
        </CardHeader>
        <CardContent className="px-0 pb-0">
          {variables.isLoading ? (
            <TableSkeleton rows={4} cols={7} />
          ) : variables.isError ? (
            <div className="p-3">
              <ErrorState error={variables.error} onRetry={() => void variables.refetch()} />
            </div>
          ) : variableList.length === 0 ? (
            <EmptyState
              title={t('gateway.detail.variables.empty')}
              description={t('gateway.detail.variables.emptyHint')}
              className="py-8"
            />
          ) : (
            <Table>
              <THead>
                <TR>
                  <TH>{t('gateway.variable.name')}</TH>
                  <TH>{t('gateway.variable.dataType')}</TH>
                  <TH>{t('gateway.variable.address')}</TH>
                  <TH>{t('gateway.variable.bits')}</TH>
                  <TH>{t('gateway.variable.rate')}</TH>
                  <TH>{t('gateway.variable.accessMode')}</TH>
                  <TH>{t('gateway.variable.value')}</TH>
                  <TH className="w-10 text-right">{t('common.field.actions')}</TH>
                </TR>
              </THead>
              <TBody>
                {variableList.map((variable) => (
                  <TR key={variable.name}>
                    <TD className="font-medium">{variable.name}</TD>
                    <TD className="font-mono text-[11px] text-muted">
                      {variable.dataType}
                      {variable.isCalculated ? (
                        <Badge tone="info" className="ml-1.5">
                          {t('gateway.variable.calculated')}
                        </Badge>
                      ) : null}
                    </TD>
                    <TD className="font-mono text-[11px] text-muted">{variable.address || '—'}</TD>
                    <TD className="tabular text-xs">{variable.bits ?? '—'}</TD>
                    <TD className="tabular text-xs">{variable.rate ?? '—'}</TD>
                    <TD>
                      <Badge tone={variable.accessMode === 'r' ? 'neutral' : 'primary'}>
                        {variable.accessMode}
                      </Badge>
                    </TD>
                    <TD className="max-w-40 truncate font-mono text-[11px] text-fg">
                      {formatValue(variable.value)}
                    </TD>
                    <TD className="text-right">
                      <button
                        type="button"
                        onClick={() => setDeleteTarget(variable.name)}
                        className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-danger/10 hover:text-danger"
                        title={t('common.action.delete')}
                        aria-label={t('common.action.delete')}
                      >
                        <Trash2 className="size-3.5" />
                      </button>
                    </TD>
                  </TR>
                ))}
              </TBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog
        open={addOpen}
        onClose={() => setAddOpen(false)}
        title={t('gateway.detail.variables.addTitle')}
        description={t('gateway.detail.variables.addDescription')}
        size="xl"
        footer={
          <>
            <Button variant="ghost" onClick={() => setAddOpen(false)}>
              {t('common.action.cancel')}
            </Button>
            <Button
              variant="primary"
              loading={addVariables.isPending}
              onClick={() => void submitVariables()}
            >
              {t('common.action.add')}
            </Button>
          </>
        }
      >
        {schemaData ? (
          <VariablesEditor
            schema={schemaData}
            variables={variableRows}
            errors={variableErrors}
            onAdd={() => setVariableRows((rows) => [...rows, createEmptyVariable(schemaData)])}
            onRemove={(index) =>
              setVariableRows((rows) => rows.filter((_, rowIndex) => rowIndex !== index))
            }
            onChange={(index, patch) =>
              setVariableRows((rows) =>
                rows.map((row, rowIndex) =>
                  rowIndex === index ? { ...row, ...patch } : row,
                ),
              )
            }
          />
        ) : null}
      </Dialog>

      <Dialog
        open={controlOpen}
        onClose={() => setControlOpen(false)}
        title={t('gateway.detail.control.title')}
        description={t('gateway.detail.control.description')}
        footer={
          <>
            <Button variant="ghost" onClick={() => setControlOpen(false)}>
              {t('common.action.cancel')}
            </Button>
            <Button
              variant="primary"
              loading={deliver.isPending}
              disabled={writableVariables.every(
                (item) => (controlValues[item.name] ?? '').trim() === '',
              )}
              onClick={() => void submitControl()}
            >
              {t('gateway.detail.control.submit')}
            </Button>
          </>
        }
      >
        {writableVariables.length === 0 ? (
          <p className="text-sm text-muted">{t('gateway.detail.control.empty')}</p>
        ) : (
          <div className="space-y-2.5">
            {writableVariables.map((variable) => (
              <Field
                key={variable.name}
                label={`${variable.name} · ${variable.dataType}`}
                htmlFor={`control-${variable.name}`}
                hint={t('gateway.detail.control.current', {
                  value: formatValue(variable.value),
                })}
              >
                <Input
                  id={`control-${variable.name}`}
                  value={controlValues[variable.name] ?? ''}
                  placeholder={t('gateway.detail.control.valuePlaceholder')}
                  autoComplete="off"
                  onChange={(event) =>
                    setControlValues((current) => ({
                      ...current,
                      [variable.name]: event.target.value,
                    }))
                  }
                />
              </Field>
            ))}
          </div>
        )}
      </Dialog>

      <ConfirmDialog
        open={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() => void confirmDelete()}
        loading={deleteCollection.isPending}
        danger
        title={t('gateway.collection.confirm.deleteTitle')}
        description={t('gateway.collection.confirm.deleteDescription', {
          name: collection.name,
        })}
        confirmText={t('common.action.delete')}
      />

      <ConfirmDialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => void confirmDeleteVariable()}
        loading={deleteVariables.isPending}
        danger
        title={t('gateway.detail.variables.deleteTitle')}
        description={t('gateway.detail.variables.deleteDescription', {
          name: deleteTarget ?? '',
        })}
        confirmText={t('common.action.delete')}
      />
    </div>
  )
}

function ProtocolReadonly({
  schema,
  config,
  locale,
}: {
  schema: DriverSchema
  config: Record<string, unknown>
  locale: 'zh-CN' | 'en-US'
}) {
  const { t } = useI18n()
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      {sortedFields(schema).map((field) => {
        const value = config?.[field.name]
        const label = localized(field.labelZh, field.labelEn, locale)
        const sensitive = /password|secret/i.test(field.name) && value
        return (
          <InfoItem key={field.name} label={label}>
            <span className="break-all font-mono text-xs">
              {sensitive ? '••••••' : formatValue(value)}
            </span>
          </InfoItem>
        )
      })}
      {sortedFields(schema).length === 0 ? (
        <p className="text-xs text-faint">{t('gateway.detail.protocolEmpty')}</p>
      ) : null}
    </div>
  )
}
