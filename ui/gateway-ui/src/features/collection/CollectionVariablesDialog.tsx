import { useEffect, useMemo, useRef, useState, type ChangeEvent } from 'react'
import { Play, Plus, RefreshCw, RotateCw, Trash2, Upload } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  ConfirmDialog,
  FormDialog,
  ErrorState,
  Pagination,
  Skeleton,
  cn,
  showErrorToast,
  useI18n,
} from '@harns/core'
import {
  useAddCollectionVariables,
  useAllCollectionVariables,
  useCollection,
  useCollectionRuntimeAction,
  useDeleteCollectionVariables,
  useDriverSchema,
  useUpdateCollectionVariables,
} from '@/api/hooks'
import type { Collection } from '@/api/types'
import { CollectStatusBadge } from '@/components/StatusBadge'
import {
  canRestartCollection,
  canStartCollection,
  canStopCollection,
} from '@/lib/collection-actions'
import {
  createEmptyCalculatedVariable,
  createEmptyVariable,
  toVariableInputs,
  validateVariableRows,
  type VariableFormValues,
} from '@/lib/collection-form'
import { importCollectionVariablesFile } from './collection-variables-excel'
import { CalculatedVariablesEditor } from './CalculatedVariablesEditor'
import { VariablesEditor } from './VariablesEditor'

interface CollectionVariablesDialogProps {
  collection: Collection | null
  onClose: () => void
}

type VariableTab = 'collection' | 'calculated'

function formatValue(value: unknown): string {
  if (value === null || value === undefined || value === '') return ''
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function comparable(row: VariableFormValues): string {
  return JSON.stringify({
    dataType: row.dataType,
    address: row.address,
    bits: row.bits,
    precision: row.precision ?? '',
    rate: row.rate,
    offset: row.offset ?? '',
    accessMode: row.accessMode,
    expression: row.expression ?? '',
    isCalculated: Boolean(row.isCalculated),
  })
}

function addCrossListDuplicateErrors(
  existing: VariableFormValues[],
  rows: VariableFormValues[],
  errors: Record<string, string>,
  message: string,
): void {
  const names = new Set(existing.map((variable) => variable.name.trim()).filter(Boolean))
  rows.forEach((variable, index) => {
    const name = variable.name.trim()
    if (name && names.has(name) && !errors[`variables.${index}.name`]) {
      errors[`variables.${index}.name`] = message
    }
  })
}

export function CollectionVariablesDialog({ collection, onClose }: CollectionVariablesDialogProps) {
  const { t, locale } = useI18n()
  const variablesQuery = useAllCollectionVariables(collection?.id)
  const detailQuery = useCollection(collection?.id)
  const schema = useDriverSchema(collection?.driverModel)
  const runtimeAction = useCollectionRuntimeAction()
  const addVariables = useAddCollectionVariables()
  const updateVariables = useUpdateCollectionVariables()
  const deleteVariables = useDeleteCollectionVariables()
  const [loadedFor, setLoadedFor] = useState<string | null>(null)
  const [normalRows, setNormalRows] = useState<VariableFormValues[]>([])
  const [calculatedRows, setCalculatedRows] = useState<VariableFormValues[]>([])
  const [normalErrors, setNormalErrors] = useState<Record<string, string>>({})
  const [calculatedErrors, setCalculatedErrors] = useState<Record<string, string>>({})
  const [normalSelected, setNormalSelected] = useState<Set<number>>(new Set())
  const [calculatedSelected, setCalculatedSelected] = useState<Set<number>>(new Set())
  const [originalRows, setOriginalRows] = useState<Map<string, VariableFormValues>>(new Map())
  const [tab, setTab] = useState<VariableTab>('collection')
  const [importing, setImporting] = useState(false)
  const [refreshing, setRefreshing] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [singleDelete, setSingleDelete] = useState<{
    kind: VariableTab
    index: number
    name: string
  } | null>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const currentCollection = detailQuery.data ?? collection
  const collectStatus = currentCollection?.collectStatus ?? ''
  const canStop = currentCollection ? canStopCollection(currentCollection) : false
  const canStart = currentCollection ? canStartCollection(currentCollection) : false
  const canRestart = currentCollection ? canRestartCollection(currentCollection) : false
  const hideValues =
    collectStatus === 'collectingError' ||
    collectStatus === 'unconnected' ||
    collectStatus === 'error'

  useEffect(() => {
    if (!collection) return
    setLoadedFor(null)
    setNormalRows([])
    setCalculatedRows([])
    setNormalErrors({})
    setCalculatedErrors({})
    setNormalSelected(new Set())
    setCalculatedSelected(new Set())
    setOriginalRows(new Map())
    setTab('collection')
    setPage(1)
  }, [collection?.id])

  useEffect(() => {
    if (!collection || !variablesQuery.data || !schema.data || loadedFor === collection.id) return
    const rows: VariableFormValues[] = variablesQuery.data.map((variable) => ({
      name: variable.name,
      dataType: variable.dataType,
      address: variable.address,
      bits: variable.bits != null ? String(variable.bits) : '',
      precision: variable.precision != null ? String(variable.precision) : '',
      rate: variable.rate != null ? String(variable.rate) : '',
      offset: variable.offset != null ? String(variable.offset) : '',
      accessMode: variable.accessMode,
      isCalculated: Boolean(variable.isCalculated),
      expression: variable.expression ?? '',
      value: hideValues ? '' : formatValue(variable.value),
      isNew: false,
    }))
    const original = new Map<string, VariableFormValues>()
    for (const row of rows) original.set(row.name, { ...row })
    setOriginalRows(original)
    setNormalRows(rows.filter((row) => !row.isCalculated))
    setCalculatedRows(rows.filter((row) => row.isCalculated))
    setLoadedFor(collection.id)
  }, [collection?.id, variablesQuery.data, schema.data, loadedFor, hideValues])

  const driverSchema = schema.data?.schema
  const activeRows = tab === 'collection' ? normalRows : calculatedRows
  const activeSelected = tab === 'collection' ? normalSelected : calculatedSelected
  const activeErrors = tab === 'collection' ? normalErrors : calculatedErrors
  const lockedNames = useMemo(() => new Set(originalRows.keys()), [originalRows])
  const pageCount = Math.max(1, Math.ceil(activeRows.length / pageSize))
  const safePage = Math.min(page, pageCount)
  const pageStart = (safePage - 1) * pageSize
  const pageRows = activeRows.slice(pageStart, pageStart + pageSize)
  const allSelected = pageRows.length > 0 && activeSelected.size === pageRows.length
  const submitting = addVariables.isPending || updateVariables.isPending || deleteVariables.isPending
  const loading = variablesQuery.isLoading || schema.isLoading || !loadedFor

  const refresh = async () => {
    setRefreshing(true)
    try {
      await Promise.all([variablesQuery.refetch(), detailQuery.refetch()])
      setLoadedFor(null)
      setNormalErrors({})
      setCalculatedErrors({})
      setNormalSelected(new Set())
      setCalculatedSelected(new Set())
    } finally {
      setRefreshing(false)
    }
  }

  const runRuntimeAction = async (action: 'start' | 'stop' | 'restart') => {
    if (!collection) return
    try {
      await runtimeAction.mutateAsync({
        id: collection.id,
        name: collection.name,
        action,
      })
      toast.success(
        action === 'start'
          ? t('gateway.collection.toast.started')
          : action === 'stop'
            ? t('gateway.collection.toast.stopped')
            : t('gateway.collection.toast.restarted'),
      )
      await detailQuery.refetch()
    } catch (error) {
      showErrorToast(error)
    }
  }

  const validate = (): boolean => {
    if (!driverSchema) return false
    const nextNormalErrors = validateVariableRows(normalRows, driverSchema, t)
    const nextCalculatedErrors = validateVariableRows(calculatedRows, driverSchema, t)
    const duplicate = t('gateway.form.validation.variableNameDuplicate')
    addCrossListDuplicateErrors(normalRows, calculatedRows, nextCalculatedErrors, duplicate)
    addCrossListDuplicateErrors(calculatedRows, normalRows, nextNormalErrors, duplicate)
    setNormalErrors(nextNormalErrors)
    setCalculatedErrors(nextCalculatedErrors)
    return (
      Object.keys(nextNormalErrors).length === 0 &&
      Object.keys(nextCalculatedErrors).length === 0
    )
  }

  const addRows = (kind: VariableTab, count: number) => {
    if (!driverSchema) return
    const total = (kind === 'collection' ? normalRows.length : calculatedRows.length) + count
    if (kind === 'collection') {
      setNormalRows((rows) => [
        ...rows,
        ...Array.from({ length: count }, () => ({
          ...createEmptyVariable(driverSchema),
          isNew: true,
          value: '',
        })),
      ])
      setNormalSelected(new Set())
    } else {
      setCalculatedRows((rows) => [
        ...rows,
        ...Array.from({ length: count }, () => ({
          ...createEmptyCalculatedVariable(driverSchema),
          isNew: true,
          value: '',
        })),
      ])
      setCalculatedSelected(new Set())
    }
    setPage(Math.max(1, Math.ceil(total / pageSize)))
  }

  const removeRows = (kind: VariableTab, indexes: Set<number>) => {
    if (kind === 'collection') {
      setNormalRows((rows) => rows.filter((_, index) => !indexes.has(index)))
      setNormalSelected(new Set())
    } else {
      setCalculatedRows((rows) => rows.filter((_, index) => !indexes.has(index)))
      setCalculatedSelected(new Set())
    }
  }

  const removeRow = (kind: VariableTab, index: number) => {
    const rows = kind === 'collection' ? normalRows : calculatedRows
    const row = rows[index]
    if (!row) return
    if (!row.isNew) {
      setSingleDelete({ kind, index, name: row.name })
      return
    }
    removeRows(kind, new Set([index]))
  }

  const confirmSingleDelete = async () => {
    if (!collection || !singleDelete) return
    try {
      await deleteVariables.mutateAsync({
        id: collection.id,
        name: collection.name,
        names: [singleDelete.name],
      })
      setOriginalRows((current) => {
        const next = new Map(current)
        next.delete(singleDelete.name)
        return next
      })
      removeRows(singleDelete.kind, new Set([singleDelete.index]))
      setSingleDelete(null)
    } catch (error) {
      showErrorToast(error)
    }
  }

  const toggleSelect = (index: number) => {
    const setter = tab === 'collection' ? setNormalSelected : setCalculatedSelected
    setter((current) => {
      const next = new Set(current)
      if (next.has(index)) next.delete(index)
      else next.add(index)
      return next
    })
  }

  const toggleAll = () => {
    const setter = tab === 'collection' ? setNormalSelected : setCalculatedSelected
    setter(
      allSelected ? new Set() : new Set(pageRows.map((_, index) => index)),
    )
  }

  const confirmBatchDelete = async () => {
    if (!collection) return
    const indexes = new Set(
      [...activeSelected]
        .map((index) => pageStart + index)
        .filter((index) => index < activeRows.length),
    )
    const rows = [...indexes].map((index) => activeRows[index]).filter(Boolean)
    const savedNames = rows.filter((row) => !row.isNew).map((row) => row.name)
    try {
      if (savedNames.length > 0) {
        await deleteVariables.mutateAsync({
          id: collection.id,
          name: collection.name,
          names: savedNames,
        })
        setOriginalRows((current) => {
          const next = new Map(current)
          for (const name of savedNames) next.delete(name)
          return next
        })
      }
      removeRows(tab, indexes)
      setDeleteOpen(false)
    } catch (error) {
      showErrorToast(error)
    }
  }

  const requestBatchDelete = () => {
    const indexes = new Set(
      [...activeSelected]
        .map((index) => pageStart + index)
        .filter((index) => index < activeRows.length),
    )
    const rows = [...indexes].map((index) => activeRows[index]).filter(Boolean)
    if (rows.length === 0) return
    if (rows.every((row) => row.isNew)) {
      removeRows(tab, indexes)
      return
    }
    setDeleteOpen(true)
  }

  const importVariables = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.currentTarget.files?.[0]
    event.currentTarget.value = ''
    if (!file || !driverSchema) return
    if (!file.name.toLowerCase().endsWith('.xlsx')) {
      showErrorToast(new Error(t('gateway.variable.importInvalidFile')))
      return
    }
    setImporting(true)
    try {
      const rows = await importCollectionVariablesFile({ file, schema: driverSchema, locale, t })
      setNormalRows((current) => [
        ...current,
        ...rows.map((row) => ({ ...row, isNew: true, value: '' })),
      ])
      setNormalSelected(new Set())
      setTab('collection')
      setPage(Math.max(1, Math.ceil((normalRows.length + rows.length) / pageSize)))
      toast.success(t('gateway.variable.imported', { count: rows.length }))
    } catch (error) {
      showErrorToast(error)
    } finally {
      setImporting(false)
    }
  }

  const save = async () => {
    if (!collection || !driverSchema || !validate()) return
    const currentRows = [...normalRows, ...calculatedRows]
    const updated = currentRows.filter((row) => {
      if (row.isNew) return false
      const original = originalRows.get(row.name)
      return original ? comparable(original) !== comparable(row) : false
    })
    const added = currentRows.filter((row) => row.isNew)
    try {
      if (updated.length > 0) {
        await updateVariables.mutateAsync({
          id: collection.id,
          name: collection.name,
          variables: toVariableInputs(updated),
        })
      }
      if (added.length > 0) {
        await addVariables.mutateAsync({
          id: collection.id,
          name: collection.name,
          variables: toVariableInputs(added),
        })
      }
      const savedRows = currentRows.map((row) => ({ ...row, isNew: false }))
      setNormalRows(savedRows.filter((row) => !row.isCalculated))
      setCalculatedRows(savedRows.filter((row) => row.isCalculated))
      setOriginalRows(new Map(savedRows.map((row) => [row.name, { ...row }])))
      setNormalErrors({})
      setCalculatedErrors({})
      setNormalSelected(new Set())
      setCalculatedSelected(new Set())
      toast.success(t('gateway.variable.saved'))
    } catch (error) {
      showErrorToast(error)
    }
  }

  return (
    <FormDialog
      open={Boolean(collection)}
      onClose={onClose}
      title={
        currentCollection ? (
          <span className="flex flex-wrap items-center gap-2">
            {t('gateway.variable.manageTitle', { name: currentCollection.name })}
            <CollectStatusBadge
              status={currentCollection.collectStatus}
              error={currentCollection.lastError}
            />
            {canStop ? (
              <Button
                variant="outline"
                size="sm"
                disabled={!canRestart}
                loading={runtimeAction.isPending}
                icon={<RotateCw className="size-3.5" />}
                onClick={() => void runRuntimeAction('restart')}
              >
                {t('common.action.restart')}
              </Button>
            ) : (
              <Button
                variant="outline"
                size="sm"
                disabled={!canStart}
                loading={runtimeAction.isPending}
                icon={<Play className="size-3.5" />}
                onClick={() => void runRuntimeAction('start')}
              >
                {t('common.action.start')}
              </Button>
            )}
          </span>
        ) : (
          t('gateway.variable.manageTitleFallback')
        )
      }
      width="wide"
      footer={
        <>
          <Button variant="outline" onClick={onClose}>
            {t('common.action.cancel')}
          </Button>
          <Button
            variant="primary"
            loading={submitting}
            disabled={loading}
            onClick={() => void save()}
          >
            {t('common.action.save')}
          </Button>
        </>
      }
    >
      {variablesQuery.isError ? (
        <ErrorState error={variablesQuery.error} onRetry={() => void variablesQuery.refetch()} />
      ) : schema.isError ? (
        <ErrorState error={schema.error} onRetry={() => void schema.refetch()} />
      ) : loading || !collection ? (
        <div className="space-y-3">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-40 w-full" />
        </div>
      ) : (
        <>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <div className="inline-flex items-center gap-1 rounded border border-border bg-surface-2 p-1">
                <button
                  type="button"
                  onClick={() => { setTab('collection'); setPage(1) }}
                  className={
                    tab === 'collection'
                      ? 'rounded bg-surface px-2.5 py-1 text-xs font-medium text-fg shadow-sm'
                      : 'rounded px-2.5 py-1 text-xs text-muted hover:text-fg'
                  }
                >
                  {t('gateway.form.tab.collectionVariables')} ({normalRows.length})
                </button>
                <button
                  type="button"
                  onClick={() => { setTab('calculated'); setPage(1) }}
                  className={
                    tab === 'calculated'
                      ? 'rounded bg-surface px-2.5 py-1 text-xs font-medium text-fg shadow-sm'
                      : 'rounded px-2.5 py-1 text-xs text-muted hover:text-fg'
                  }
                >
                  {t('gateway.form.tab.calculatedVariables')} ({calculatedRows.length})
                </button>
              </div>
              <button
                type="button"
                title={t('gateway.variable.refresh')}
                aria-label={t('gateway.variable.refresh')}
                onClick={() => void refresh()}
                className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-surface-2 hover:text-fg"
              >
                <RefreshCw className={cn('size-3.5', refreshing && 'animate-spin')} />
              </button>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {tab === 'collection' ? (
                <>
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".xlsx"
                    className="hidden"
                    onChange={(event) => void importVariables(event)}
                  />
                  <Button
                    variant="outline"
                    size="sm"
                    loading={importing}
                    icon={<Upload className="size-3.5" />}
                    onClick={() => fileInputRef.current?.click()}
                  >
                    {t('gateway.variable.import')}
                  </Button>
                </>
              ) : null}
              <Button
                variant="outline"
                size="sm"
                icon={<Plus className="size-3.5" />}
                onClick={() => addRows(tab, 5)}
              >
                {t('gateway.variable.batchAdd')}
              </Button>
              <Button
                variant="dangerSoft"
                size="sm"
                icon={<Trash2 className="size-3.5" />}
                disabled={activeSelected.size === 0}
                onClick={requestBatchDelete}
              >
                {t('gateway.variable.batchDelete')}
              </Button>
            </div>
          </div>

          {driverSchema ? (
            tab === 'collection' ? (
              <VariablesEditor
                schema={driverSchema}
                variables={pageRows}
                errors={normalErrors}
                onAdd={() => addRows('collection', 1)}
                onRemove={(index) => void removeRow('collection', pageStart + index)}
                onChange={(index, patch) =>
                  setNormalRows((rows) =>
                    rows.map((row, rowIndex) =>
                      rowIndex === pageStart + index ? { ...row, ...patch } : row,
                    ),
                  )
                }
                selectable
                selected={normalSelected}
                allSelected={tab === 'collection' && allSelected}
                onToggleSelect={toggleSelect}
                onToggleAll={toggleAll}
                showValue
                lockedNames={lockedNames}
              />
            ) : (
              <CalculatedVariablesEditor
                schema={driverSchema}
                variables={pageRows}
                errors={calculatedErrors}
                onAdd={() => addRows('calculated', 1)}
                onRemove={(index) => void removeRow('calculated', pageStart + index)}
                onChange={(index, patch) =>
                  setCalculatedRows((rows) =>
                    rows.map((row, rowIndex) =>
                      rowIndex === pageStart + index ? { ...row, ...patch } : row,
                    ),
                  )
                }
                selectable
                selected={calculatedSelected}
                allSelected={tab === 'calculated' && allSelected}
                onToggleSelect={toggleSelect}
                onToggleAll={toggleAll}
                showValue
                lockedNames={lockedNames}
              />
            )
          ) : null}

          <Pagination
            page={safePage}
            pageSize={pageSize}
            total={activeRows.length}
            onPageChange={(nextPage) => {
              setPage(nextPage)
              if (tab === 'collection') setNormalSelected(new Set())
              else setCalculatedSelected(new Set())
            }}
            onPageSizeChange={(size) => {
              setPageSize(size)
              setPage(1)
              setNormalSelected(new Set())
              setCalculatedSelected(new Set())
            }}
            disabled={submitting}
          />

          <p className="app-dialog-error">{activeErrors.variables}</p>
        </>
      )}

      <ConfirmDialog
        open={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() => void confirmBatchDelete()}
        loading={deleteVariables.isPending}
        danger
        title={t('gateway.variable.batchDeleteTitle')}
        description={t('gateway.variable.batchDeleteDescription', {
          count: activeSelected.size,
        })}
        confirmText={t('common.action.delete')}
      />

      <ConfirmDialog
        open={Boolean(singleDelete)}
        onClose={() => setSingleDelete(null)}
        onConfirm={() => void confirmSingleDelete()}
        loading={deleteVariables.isPending}
        danger
        title={t('gateway.variable.deleteSavedTitle')}
        description={t('gateway.variable.deleteSavedDescription', {
          name: singleDelete?.name ?? '',
        })}
        confirmText={t('common.action.delete')}
      />
    </FormDialog>
  )
}
