import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Boxes, Pencil, Plus, RadioTower, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import {
  useDeleteThingModel,
  useDeleteThingModelProperties,
  useThingModel,
} from '@/api/hooks'
import type { ThingModelProperty } from '@/api/types'
import { useI18n, type MessageKey } from '@harns/core'
import { formatDateTime } from '@harns/core'
import { errorMessage } from '@harns/core'
import { Badge } from '@harns/core'
import { Button } from '@harns/core'
import { Card } from '@harns/core'
import { ConfirmDialog } from '@harns/core'
import { EmptyState } from '@harns/core'
import { TableSkeleton } from '@harns/core'
import { Tabs } from '@harns/core'
import { TBody, TD, TH, THead, TR, Table } from '@harns/core'
import { ErrorState } from '@harns/core'
import { PageHeader } from '@harns/core'
import { PageLoading } from '@harns/core'
import { EditThingModelDialog } from './EditThingModelDialog'
import { PropertyDialog } from './PropertyDialog'
import { TslPanel } from './TslPanel'
import { describeDataType } from './data-type'

type TabValue = 'properties' | 'features' | 'tsl'

const TABS: Array<{ value: TabValue; labelKey: MessageKey }> = [
  { value: 'properties', labelKey: 'thingModelDetail.tab.properties' },
  { value: 'features', labelKey: 'thingModelDetail.tab.features' },
  { value: 'tsl', labelKey: 'thingModelDetail.tab.tsl' },
]

export function ThingModelDetailPage() {
  const { t } = useI18n()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const model = useThingModel(id)
  const deleteMutation = useDeleteThingModel()
  const deleteProperties = useDeleteThingModelProperties()

  const [tab, setTab] = useState<TabValue>('properties')
  const [editOpen, setEditOpen] = useState(false)
  const [propertyDialog, setPropertyDialog] = useState<{
    open: boolean
    property?: ThingModelProperty
  }>({ open: false })
  const [deleteModelOpen, setDeleteModelOpen] = useState(false)
  const [deleteProperty, setDeleteProperty] = useState<ThingModelProperty | null>(null)

  if (model.isLoading) return <PageLoading label={t('thingModelDetail.loading')} />
  if (model.isError) {
    return <ErrorState error={model.error} onRetry={model.refetch} />
  }
  if (!model.data) {
    return (
      <ErrorState
        error={new Error(t('thingModelDetail.notFound'))}
        onRetry={model.refetch}
      />
    )
  }

  const detail = model.data
  const properties = detail.properties ?? []
  const features = detail.features ?? []

  const confirmDeleteModel = async () => {
    try {
      await deleteMutation.mutateAsync(detail.id)
      toast.success(t('thingModelDetail.modelDeleted'))
      navigate('/platform/thing-models', { replace: true })
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const confirmDeleteProperty = async () => {
    if (!deleteProperty) return
    try {
      await deleteProperties.mutateAsync({
        id: detail.id,
        identifiers: [deleteProperty.identifier],
      })
      toast.success(t('thingModelDetail.propertyDeleted'))
      setDeleteProperty(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <div>
      <PageHeader
        title={detail.name}
        description={detail.description || t('thingModelDetail.noDescription')}
        backTo="/platform/thing-models"
        backLabel={t('thingModelDetail.back')}
        actions={
          <>
            <Button
              variant="outline"
              icon={<RadioTower className="size-4" />}
              onClick={() => navigate(`/platform/things/new?thingModelId=${detail.id}`)}
            >
              {t('thingModelDetail.createThing')}
            </Button>
            <Button
              variant="outline"
              icon={<Pencil className="size-4" />}
              onClick={() => setEditOpen(true)}
            >
              {t('common.action.edit')}
            </Button>
            <Button
              variant="ghost"
              icon={<Trash2 className="size-4 text-danger" />}
              onClick={() => setDeleteModelOpen(true)}
            >
              {t('common.action.delete')}
            </Button>
          </>
        }
      />

      <Card className="mb-4 grid gap-4 p-5 sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <p className="text-[11px] text-faint">{t('thingModelDetail.key')}</p>
          <p className="mt-1 font-mono text-sm">{detail.thingModelKey}</p>
        </div>
        <div>
          <p className="text-[11px] text-faint">{t('thingModelDetail.propsFeatures')}</p>
          <p className="mt-1 text-sm tabular">
            {t('thingModelDetail.counts', {
              properties: properties.length,
              features: features.length,
            })}
          </p>
        </div>
        <div>
          <p className="text-[11px] text-faint">{t('common.field.createdAt')}</p>
          <p className="mt-1 text-sm">{formatDateTime(detail.createdTime)}</p>
        </div>
        <div>
          <p className="text-[11px] text-faint">{t('common.field.updatedAt')}</p>
          <p className="mt-1 text-sm">{formatDateTime(detail.updatedTime)}</p>
        </div>
      </Card>

      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <Tabs
          items={TABS.map((item) => ({ value: item.value, label: t(item.labelKey) }))}
          value={tab}
          onChange={setTab}
        />
        {tab === 'properties' ? (
          <Button
            variant="primary"
            icon={<Plus className="size-4" />}
            onClick={() => setPropertyDialog({ open: true })}
          >
            {t('thingModelDetail.addProperty')}
          </Button>
        ) : null}
      </div>

      {tab === 'properties' ? (
        <Card className="overflow-hidden">
          {deleteProperties.isPending && properties.length === 0 ? (
            <TableSkeleton rows={4} cols={5} />
          ) : properties.length === 0 ? (
            <EmptyState
              icon={<Boxes className="size-5" />}
              title={t('thingModelDetail.noProperties')}
              description={t('thingModelDetail.noPropertiesDescription')}
              action={
                <Button
                  variant="primary"
                  icon={<Plus className="size-4" />}
                  onClick={() => setPropertyDialog({ open: true })}
                >
                  {t('thingModelDetail.addProperty')}
                </Button>
              }
            />
          ) : (
            <Table>
              <THead>
                <tr>
                  <TH>{t('thingModelDetail.table.property')}</TH>
                  <TH>{t('thingModelDetail.table.identifier')}</TH>
                  <TH>{t('thingModelDetail.table.dataType')}</TH>
                  <TH>{t('thingModelDetail.table.rw')}</TH>
                  <TH>{t('thingModelDetail.table.required')}</TH>
                  <TH>{t('thingModelDetail.table.sort')}</TH>
                  <TH className="text-right">{t('common.field.actions')}</TH>
                </tr>
              </THead>
              <TBody>
                {properties.map((property) => (
                  <TR key={property.id || property.identifier}>
                    <TD>
                      <p className="font-medium">{property.name}</p>
                      {property.description ? (
                        <p className="max-w-64 truncate text-[11px] text-faint" title={property.description}>
                          {property.description}
                        </p>
                      ) : null}
                    </TD>
                    <TD className="font-mono text-xs text-muted">{property.identifier}</TD>
                    <TD className="max-w-72 truncate text-xs text-muted" title={describeDataType(property.dataType)}>
                      {describeDataType(property.dataType)}
                    </TD>
                    <TD>
                      <Badge tone={property.rwFlag === 'rw' ? 'primary' : 'neutral'}>
                        {property.rwFlag ?? 'r'}
                      </Badge>
                    </TD>
                    <TD className="text-xs text-muted">
                      {property.required ? t('common.state.yes') : t('common.state.no')}
                    </TD>
                    <TD className="text-xs text-muted tabular">{property.sort ?? 0}</TD>
                    <TD>
                      <div className="flex justify-end gap-1">
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => setPropertyDialog({ open: true, property })}
                        >
                          {t('common.action.edit')}
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label={t('thingModelDetail.deletePropertyAria', {
                            name: property.name,
                          })}
                          onClick={() => setDeleteProperty(property)}
                        >
                          <Trash2 className="size-3.5 text-danger" />
                        </Button>
                      </div>
                    </TD>
                  </TR>
                ))}
              </TBody>
            </Table>
          )}
        </Card>
      ) : null}

      {tab === 'features' ? (
        <Card className="overflow-hidden">
          {features.length === 0 ? (
            <EmptyState
              title={t('thingModelDetail.noFeatures')}
              description={t('thingModelDetail.noFeaturesDescription')}
              action={
                <Button variant="outline" onClick={() => setTab('tsl')}>
                  {t('thingModelDetail.gotoTsl')}
                </Button>
              }
            />
          ) : (
            <Table>
              <THead>
                <tr>
                  <TH>{t('thingModelDetail.table.feature')}</TH>
                  <TH>{t('thingModelDetail.table.identifier')}</TH>
                  <TH>{t('thingModelDetail.table.dataType')}</TH>
                  <TH>{t('thingModelDetail.table.description')}</TH>
                  <TH>{t('thingModelDetail.table.sort')}</TH>
                </tr>
              </THead>
              <TBody>
                {features.map((feature) => (
                  <TR key={feature.identifier}>
                    <TD className="font-medium">{feature.name}</TD>
                    <TD className="font-mono text-xs text-muted">{feature.identifier}</TD>
                    <TD className="text-xs text-muted">{describeDataType(feature.dataType)}</TD>
                    <TD className="max-w-64 truncate text-xs text-muted" title={feature.description}>
                      {feature.description || '—'}
                    </TD>
                    <TD className="text-xs text-muted tabular">{feature.sort ?? 0}</TD>
                  </TR>
                ))}
              </TBody>
            </Table>
          )}
        </Card>
      ) : null}

      {tab === 'tsl' ? <TslPanel model={detail} /> : null}

      <EditThingModelDialog open={editOpen} onClose={() => setEditOpen(false)} model={detail} />

      <PropertyDialog
        open={propertyDialog.open}
        property={propertyDialog.property}
        modelId={detail.id}
        onClose={() => setPropertyDialog({ open: false })}
      />

      <ConfirmDialog
        open={deleteModelOpen}
        title={t('thingModelDetail.deleteModelTitle')}
        description={t('thingModelDetail.deleteModelDescription', {
          name: detail.name,
          count: properties.length,
        })}
        confirmText={t('common.action.delete')}
        danger
        loading={deleteMutation.isPending}
        onConfirm={confirmDeleteModel}
        onClose={() => setDeleteModelOpen(false)}
      />

      <ConfirmDialog
        open={Boolean(deleteProperty)}
        title={t('thingModelDetail.deletePropertyTitle')}
        description={
          deleteProperty
            ? t('thingModelDetail.deletePropertyDescription', {
                name: deleteProperty.name,
                identifier: deleteProperty.identifier,
              })
            : undefined
        }
        confirmText={t('common.action.delete')}
        danger
        loading={deleteProperties.isPending}
        onConfirm={confirmDeleteProperty}
        onClose={() => setDeleteProperty(null)}
      />
    </div>
  )
}
