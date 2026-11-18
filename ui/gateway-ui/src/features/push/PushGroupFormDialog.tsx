import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  FormDialog,
  Field,
  Input,
  Pagination,
  SearchSelect,
  Table,
  TBody,
  TD,
  TH,
  THead,
  TR,
  showErrorToast,
  useI18n,
  type SearchSelectOption,
} from '@harns/core'
import { useCollections, useCreatePushGroup, useUpdatePushGroup } from '@/api/hooks'
import type { PushConnector, PushGroup, PushGroupPayload } from '@/api/types'
import { CollectStatusBadge } from '@/components/StatusBadge'
import { driverModelLabel, driverTypeLabel } from '@/lib/constants'

interface GroupForm {
  name: string
  connectorId: string
  dataTopic: string
  commandTopic: string
  pushIntervalMs: string
  maxCollectionsPerPushMessage: string
  allCollections: boolean
  collectionIds: string[]
}

function emptyGroup(connectorId = ''): GroupForm {
  return {
    name: '',
    connectorId,
    dataTopic: '',
    commandTopic: '',
    pushIntervalMs: '1000',
    maxCollectionsPerPushMessage: '50',
    allCollections: true,
    collectionIds: [],
  }
}

function groupToForm(group: PushGroup): GroupForm {
  return {
    name: group.name,
    connectorId: group.connectorId,
    dataTopic: group.dataTopic ?? '',
    commandTopic: group.commandTopic ?? '',
    pushIntervalMs: String(group.pushIntervalMs || 1000),
    maxCollectionsPerPushMessage: String(group.maxCollectionsPerPushMessage || 50),
    allCollections: group.allCollections,
    collectionIds: [...(group.collectionIds ?? [])],
  }
}

function groupPayload(form: GroupForm): PushGroupPayload {
  const payload: PushGroupPayload = {
    name: form.name.trim(),
    connectorId: form.connectorId,
    dataTopic: form.dataTopic.trim(),
    commandTopic: form.commandTopic.trim(),
    pushIntervalMs: Number(form.pushIntervalMs),
    allCollections: form.allCollections,
    collectionIds: form.allCollections ? [] : form.collectionIds,
  }
  const max = Number(form.maxCollectionsPerPushMessage)
  if (Number.isFinite(max) && max > 0) payload.maxCollectionsPerPushMessage = max
  return payload
}

function validateGroup(
  form: GroupForm,
  t: ReturnType<typeof useI18n>['t'],
): Record<string, string> {
  const errors: Record<string, string> = {}
  if (!form.name.trim()) errors.name = t('gateway.form.validation.nameRequired')
  if (!form.connectorId) errors.connectorId = t('gateway.push.validation.connectorRequired')
  if (!form.dataTopic.trim()) errors.dataTopic = t('gateway.push.validation.dataTopicRequired')
  const interval = Number(form.pushIntervalMs)
  if (!Number.isFinite(interval) || interval <= 0) {
    errors.pushIntervalMs = t('gateway.push.validation.intervalInvalid')
  }
  const max = Number(form.maxCollectionsPerPushMessage)
  if (
    form.maxCollectionsPerPushMessage.trim() !== '' &&
    (!Number.isFinite(max) || max <= 0)
  ) {
    errors.maxCollectionsPerPushMessage = t('gateway.push.validation.maxInvalid')
  }
  if (!form.allCollections && form.collectionIds.length === 0) {
    errors.collectionIds = t('gateway.push.validation.collectionsRequired')
  }
  return errors
}

interface PushGroupFormDialogProps {
  open: boolean
  group: PushGroup | null
  connectors: PushConnector[]
  defaultConnectorId?: string
  onClose: () => void
}

export function PushGroupFormDialog({
  open,
  group,
  connectors,
  defaultConnectorId,
  onClose,
}: PushGroupFormDialogProps) {
  const { t } = useI18n()
  const create = useCreatePushGroup()
  const update = useUpdatePushGroup()
  const [form, setForm] = useState<GroupForm>(emptyGroup())
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [collectionPage, setCollectionPage] = useState(1)
  const [collectionPageSize, setCollectionPageSize] = useState(5)
  const collections = useCollections({
    page: collectionPage,
    pageSize: collectionPageSize,
  })

  useEffect(() => {
    if (!open) return
    setForm(
      group
        ? groupToForm(group)
        : emptyGroup(defaultConnectorId || connectors[0]?.id || ''),
    )
    setErrors({})
    setCollectionPage(1)
  }, [open, group, defaultConnectorId])

  const connectorOptions = useMemo<SearchSelectOption[]>(
    () =>
      connectors.map((connector) => ({
        value: connector.id,
        label: `${connector.name} · ${connector.pushConfig?.brokerUrl ?? ''}`,
      })),
    [connectors],
  )

  const collectionList = collections.data?.list ?? []
  const collectionTotal = collections.data?.total ?? 0
  const pageIds = collectionList.map((item) => item.id)
  const pageAllSelected =
    pageIds.length > 0 && pageIds.every((id) => form.collectionIds.includes(id))

  const toggleCollection = (id: string) => {
    setForm((current) => ({
      ...current,
      collectionIds: current.collectionIds.includes(id)
        ? current.collectionIds.filter((item) => item !== id)
        : [...current.collectionIds, id],
    }))
  }

  const togglePage = () => {
    setForm((current) => {
      const selected = new Set(current.collectionIds)
      if (pageAllSelected) {
        for (const id of pageIds) selected.delete(id)
      } else {
        for (const id of pageIds) selected.add(id)
      }
      return { ...current, collectionIds: [...selected] }
    })
  }

  const submit = async () => {
    const nextErrors = validateGroup(form, t)
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length > 0) return
    try {
      if (group) {
        await update.mutateAsync({ id: group.id, payload: groupPayload(form) })
        toast.success(t('gateway.push.toast.groupUpdated'))
      } else {
        await create.mutateAsync(groupPayload(form))
        toast.success(t('gateway.push.toast.groupCreated'))
      }
      onClose()
    } catch (error) {
      showErrorToast(error)
    }
  }

  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={group ? t('gateway.push.group.editTitle') : t('gateway.push.group.createTitle')}
      footer={
        <>
          <Button variant="outline" onClick={onClose}>
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
      <Card>
        <CardHeader>
          <CardTitle>{t('gateway.form.section.basic')}</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-x-8 gap-y-4 sm:grid-cols-2">
          <Field
            label={t('common.field.name')}
            htmlFor="push-group-name"
            labelClassName="text-sm"
            labelWidthClassName="w-[96px]"
            inline
            required
            error={errors.name}
          >
            <Input
              id="push-group-name"
              value={form.name}
              className="w-full max-w-[240px]"
              autoFocus
              autoComplete="off"
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
          </Field>
          <Field
            label={t('gateway.push.columns.connector')}
            htmlFor="push-group-connector"
            labelClassName="text-sm"
            labelWidthClassName="w-[96px]"
            inline
            required
            error={errors.connectorId}
          >
            <SearchSelect
              id="push-group-connector"
              value={form.connectorId}
              options={connectorOptions}
              onChange={(connectorId) => setForm({ ...form, connectorId })}
              ariaLabel={t('gateway.push.columns.connector')}
              emptyText={t('common.state.empty')}
              searchable={false}
              className="w-full max-w-[240px]"
            />
          </Field>
          <Field
            label={t('gateway.push.field.dataTopic')}
            htmlFor="push-group-data-topic"
            labelClassName="text-sm"
            labelWidthClassName="w-[96px]"
            inline
            required
            error={errors.dataTopic}
          >
            <Input
              id="push-group-data-topic"
              value={form.dataTopic}
              className="w-full max-w-[240px]"
              placeholder="data/{gatewayId}/v1/#"
              autoComplete="off"
              onChange={(event) => setForm({ ...form, dataTopic: event.target.value })}
            />
          </Field>
          <Field
            label={t('gateway.push.field.commandTopic')}
            htmlFor="push-group-command-topic"
            labelClassName="text-sm"
            labelWidthClassName="w-[96px]"
            inline
          >
            <Input
              id="push-group-command-topic"
              value={form.commandTopic}
              className="w-full max-w-[240px]"
              placeholder="command/{gatewayId}/#"
              autoComplete="off"
              onChange={(event) => setForm({ ...form, commandTopic: event.target.value })}
            />
          </Field>
          <Field
            label={t('gateway.push.columns.interval')}
            htmlFor="push-group-interval"
            labelClassName="text-sm"
            labelWidthClassName="w-[96px]"
            inline
            error={errors.pushIntervalMs}
          >
            <div className="space-y-1">
              <Input
                id="push-group-interval"
                value={form.pushIntervalMs}
                className="w-full max-w-[240px]"
                inputMode="numeric"
                onChange={(event) => setForm({ ...form, pushIntervalMs: event.target.value })}
              />
              <p className="text-[13px] leading-relaxed text-muted">
                {t('gateway.push.group.intervalHint')}
              </p>
            </div>
          </Field>
          <Field
            label={t('gateway.push.columns.maxPerMessage')}
            htmlFor="push-group-max"
            labelClassName="text-sm"
            labelWidthClassName="w-[96px]"
            inline
            error={errors.maxCollectionsPerPushMessage}
          >
            <Input
              id="push-group-max"
              value={form.maxCollectionsPerPushMessage}
              className="w-full max-w-[240px]"
              inputMode="numeric"
              onChange={(event) =>
                setForm({ ...form, maxCollectionsPerPushMessage: event.target.value })
              }
            />
          </Field>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('gateway.push.group.scopeTitle')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex flex-wrap items-center gap-5">
            <label className="flex cursor-pointer items-center gap-2 text-sm text-fg">
              <input
                type="radio"
                name="push-group-scope"
                checked={form.allCollections}
                onChange={() => setForm({ ...form, allCollections: true })}
              />
              {t('gateway.push.scope.all')}
            </label>
            <label className="flex cursor-pointer items-center gap-2 text-sm text-fg">
              <input
                type="radio"
                name="push-group-scope"
                checked={!form.allCollections}
                onChange={() => {
                  setCollectionPage(1)
                  setForm({ ...form, allCollections: false })
                }}
              />
              {t('gateway.push.scope.select')}
            </label>
          </div>

          {!form.allCollections ? (
            <div className="space-y-2">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <p className="text-sm font-medium text-muted">
                  {t('gateway.push.group.selectCollections', {
                    count: form.collectionIds.length,
                  })}
                </p>
              </div>
              <div className="overflow-hidden rounded border border-border">
                {collections.isLoading ? (
                  <p className="px-3 py-6 text-center text-sm text-faint">
                    {t('common.state.loading')}
                  </p>
                ) : collections.isError ? (
                  <p className="px-3 py-6 text-center text-sm text-danger">
                    {t('common.state.loadFailed')}
                  </p>
                ) : collectionList.length === 0 ? (
                  <p className="px-3 py-6 text-center text-sm text-faint">
                    {t('gateway.push.group.noCollections')}
                  </p>
                ) : (
                  <Table>
                    <THead>
                      <TR>
                        <TH className="w-10">
                          <input
                            type="checkbox"
                            checked={pageAllSelected}
                            onChange={togglePage}
                            aria-label={t('common.action.selectAll')}
                          />
                        </TH>
                        <TH>{t('gateway.push.group.collectionName')}</TH>
                        <TH>{t('gateway.collection.columns.status')}</TH>
                        <TH>{t('gateway.collection.columns.driverType')}</TH>
                        <TH>{t('gateway.push.group.protocolType')}</TH>
                      </TR>
                    </THead>
                    <TBody>
                      {collectionList.map((collection) => (
                        <TR key={collection.id}>
                          <TD>
                            <input
                              type="checkbox"
                              checked={form.collectionIds.includes(collection.id)}
                              onChange={() => toggleCollection(collection.id)}
                              aria-label={collection.name}
                            />
                          </TD>
                          <TD className="max-w-[220px] truncate text-fg">{collection.name}</TD>
                          <TD>
                            <CollectStatusBadge
                              status={collection.collectStatus}
                              error={collection.lastError}
                            />
                          </TD>
                          <TD className="text-muted">
                            {driverTypeLabel(collection.driverType) || '—'}
                          </TD>
                          <TD className="text-muted">
                            {driverModelLabel(collection.driverModel) || '—'}
                          </TD>
                        </TR>
                      ))}
                    </TBody>
                  </Table>
                )}
              </div>
              <Pagination
                page={collectionPage}
                pageSize={collectionPageSize}
                total={collectionTotal}
                onPageChange={setCollectionPage}
                onPageSizeChange={(size) => {
                  setCollectionPageSize(size)
                  setCollectionPage(1)
                }}
                disabled={collections.isFetching}
              />
              <p className="min-h-4 text-xs leading-4 text-danger">{errors.collectionIds}</p>
            </div>
          ) : null}
        </CardContent>
      </Card>
    </FormDialog>
  )
}
