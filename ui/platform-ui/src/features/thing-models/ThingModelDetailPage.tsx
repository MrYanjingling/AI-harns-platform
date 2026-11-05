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
import { formatDateTime } from '@/lib/format'
import { errorMessage } from '@/lib/http'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState } from '@/components/ui/EmptyState'
import { TableSkeleton } from '@/components/ui/Skeleton'
import { Tabs } from '@/components/ui/Tabs'
import { TBody, TD, TH, THead, TR, Table } from '@/components/ui/Table'
import { ErrorState } from '@/components/shared/ErrorState'
import { PageHeader } from '@/components/shared/PageHeader'
import { PageLoading } from '@/components/ui/Spinner'
import { EditThingModelDialog } from './EditThingModelDialog'
import { PropertyDialog } from './PropertyDialog'
import { TslPanel } from './TslPanel'
import { describeDataType } from './data-type'

type TabValue = 'properties' | 'features' | 'tsl'

const TABS: Array<{ value: TabValue; label: string }> = [
  { value: 'properties', label: '属性' },
  { value: 'features', label: '特征' },
  { value: 'tsl', label: 'TSL 导入导出' },
]

export function ThingModelDetailPage() {
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

  if (model.isLoading) return <PageLoading label="正在载入物模型…" />
  if (model.isError) {
    return <ErrorState error={model.error} onRetry={model.refetch} />
  }
  if (!model.data) return <ErrorState error={new Error('物模型不存在')} onRetry={model.refetch} />

  const detail = model.data
  const properties = detail.properties ?? []
  const features = detail.features ?? []

  const confirmDeleteModel = async () => {
    try {
      await deleteMutation.mutateAsync(detail.id)
      toast.success('物模型已删除')
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
      toast.success('属性已删除')
      setDeleteProperty(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <div>
      <PageHeader
        title={detail.name}
        description={detail.description || '未填写描述'}
        backTo="/platform/thing-models"
        backLabel="返回物模型列表"
        actions={
          <>
            <Button
              variant="outline"
              icon={<RadioTower className="size-4" />}
              onClick={() => navigate(`/platform/things/new?thingModelId=${detail.id}`)}
            >
              基于此模型创建设备
            </Button>
            <Button
              variant="outline"
              icon={<Pencil className="size-4" />}
              onClick={() => setEditOpen(true)}
            >
              编辑
            </Button>
            <Button
              variant="ghost"
              icon={<Trash2 className="size-4 text-danger" />}
              onClick={() => setDeleteModelOpen(true)}
            >
              删除
            </Button>
          </>
        }
      />

      <Card className="mb-4 grid gap-4 p-5 sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <p className="text-[11px] text-faint">物模型 Key</p>
          <p className="mt-1 font-mono text-sm">{detail.thingModelKey}</p>
        </div>
        <div>
          <p className="text-[11px] text-faint">属性 / 特征</p>
          <p className="mt-1 text-sm tabular">
            {properties.length} 个属性 · {features.length} 个特征
          </p>
        </div>
        <div>
          <p className="text-[11px] text-faint">创建时间</p>
          <p className="mt-1 text-sm">{formatDateTime(detail.createdTime)}</p>
        </div>
        <div>
          <p className="text-[11px] text-faint">更新时间</p>
          <p className="mt-1 text-sm">{formatDateTime(detail.updatedTime)}</p>
        </div>
      </Card>

      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <Tabs items={TABS} value={tab} onChange={setTab} />
        {tab === 'properties' ? (
          <Button
            variant="primary"
            icon={<Plus className="size-4" />}
            onClick={() => setPropertyDialog({ open: true })}
          >
            添加属性
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
              title="还没有属性"
              description="添加属性后,设备即可按标识符上报动态数据。"
              action={
                <Button
                  variant="primary"
                  icon={<Plus className="size-4" />}
                  onClick={() => setPropertyDialog({ open: true })}
                >
                  添加属性
                </Button>
              }
            />
          ) : (
            <Table>
              <THead>
                <tr>
                  <TH>属性</TH>
                  <TH>标识符</TH>
                  <TH>数据类型</TH>
                  <TH>读写</TH>
                  <TH>必填</TH>
                  <TH>排序</TH>
                  <TH className="text-right">操作</TH>
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
                    <TD className="text-xs text-muted">{property.required ? '是' : '否'}</TD>
                    <TD className="text-xs text-muted tabular">{property.sort ?? 0}</TD>
                    <TD>
                      <div className="flex justify-end gap-1">
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => setPropertyDialog({ open: true, property })}
                        >
                          编辑
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label={`删除属性 ${property.name}`}
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
              title="没有特征定义"
              description="特征定义只能在创建物模型或导入 TSL 时维护;设备可为其填写静态值。"
              action={
                <Button variant="outline" onClick={() => setTab('tsl')}>
                  前往 TSL 导入
                </Button>
              }
            />
          ) : (
            <Table>
              <THead>
                <tr>
                  <TH>特征</TH>
                  <TH>标识符</TH>
                  <TH>数据类型</TH>
                  <TH>描述</TH>
                  <TH>排序</TH>
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
        title="删除物模型"
        description={`确定删除「${detail.name}」及其 ${properties.length} 个属性吗?该操作不可恢复。`}
        confirmText="删除"
        danger
        loading={deleteMutation.isPending}
        onConfirm={confirmDeleteModel}
        onClose={() => setDeleteModelOpen(false)}
      />

      <ConfirmDialog
        open={Boolean(deleteProperty)}
        title="删除属性"
        description={
          deleteProperty
            ? `确定删除属性「${deleteProperty.name}」(${deleteProperty.identifier})吗?`
            : undefined
        }
        confirmText="删除"
        danger
        loading={deleteProperties.isPending}
        onConfirm={confirmDeleteProperty}
        onClose={() => setDeleteProperty(null)}
      />
    </div>
  )
}
