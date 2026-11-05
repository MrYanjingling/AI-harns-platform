import { useState, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Eye, EyeOff, ImageIcon, Pencil, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { useDeleteThings, useThing, useThingModel, useUpdateThing } from '@/api/hooks'
import type { ThingDetail, ThingModelFeature, UpdateThingRequest } from '@/api/types'
import { formatDateTime, formatRelative, maskSecret } from '@/lib/format'
import { errorMessage } from '@/lib/http'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { CopyButton } from '@/components/ui/CopyButton'
import { Dialog } from '@/components/ui/Dialog'
import { EmptyState } from '@/components/ui/EmptyState'
import { Switch } from '@/components/ui/Switch'
import { TBody, TD, TH, THead, TR, Table } from '@/components/ui/Table'
import { PageLoading } from '@/components/ui/Spinner'
import { ErrorState } from '@/components/shared/ErrorState'
import { PageHeader } from '@/components/shared/PageHeader'
import { describeDataType } from '@/features/thing-models/data-type'
import { EditThingDialog } from './EditThingDialog'
import { FeatureValuesEditor } from './FeatureValuesEditor'
import { ImageUploadField } from './ImageUploadField'

export function ThingDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const thingQuery = useThing(id)
  const modelQuery = useThingModel(thingQuery.data?.thingModelId)
  const updateMutation = useUpdateThing()
  const deleteMutation = useDeleteThings()

  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [featureOpen, setFeatureOpen] = useState(false)
  const [featureDraft, setFeatureDraft] = useState<Record<string, unknown>>({})

  if (thingQuery.isLoading) return <PageLoading label="正在载入设备…" />
  if (thingQuery.isError) {
    return <ErrorState error={thingQuery.error} onRetry={thingQuery.refetch} />
  }
  if (!thingQuery.data) {
    return <ErrorState error={new Error('设备不存在')} onRetry={thingQuery.refetch} />
  }

  const thing = thingQuery.data
  const online = thing.enable === 1 && thing.onlineStatus === 1
  const features = modelQuery.data?.features ?? []

  const update = async (payload: UpdateThingRequest) => {
    try {
      await updateMutation.mutateAsync({ id: thing.id, payload })
      return true
    } catch (error) {
      toast.error(errorMessage(error))
      return false
    }
  }

  const toggleEnable = async (next: boolean) => {
    const ok = await update({ enable: next ? 1 : 0 })
    if (ok) toast.success(next ? '设备已启用' : '设备已禁用')
  }

  const changeImage = async (key: string) => {
    const ok = await update({ image: key })
    if (ok) toast.success(key ? '设备图片已更新' : '设备图片已移除')
  }

  const openFeatures = () => {
    setFeatureDraft({ ...(thing.features ?? {}) })
    setFeatureOpen(true)
  }

  const saveFeatures = async () => {
    const ok = await update({ features: featureDraft })
    if (ok) {
      toast.success('特征值已更新')
      setFeatureOpen(false)
    }
  }

  const confirmDelete = async () => {
    try {
      await deleteMutation.mutateAsync([thing.id])
      toast.success('设备已删除')
      navigate('/platform/things', { replace: true })
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <div>
      <PageHeader
        title={thing.deviceName}
        description={thing.deviceDesc || '未填写设备描述'}
        backTo="/platform/things"
        backLabel="返回设备列表"
        actions={
          <>
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
              onClick={() => setDeleteOpen(true)}
            >
              删除
            </Button>
          </>
        }
      />

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="p-5 lg:col-span-3">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex flex-wrap items-center gap-3">
              {thing.enable === 1 ? (
                <Badge tone="success" dot>
                  已启用
                </Badge>
              ) : (
                <Badge tone="neutral" dot>
                  已禁用
                </Badge>
              )}
              {online ? (
                <Badge tone="info" dot>
                  在线
                </Badge>
              ) : (
                <Badge tone="neutral" dot>
                  离线
                </Badge>
              )}
              <span className="text-xs text-muted">
                最后上报 {thing.lastActiveTime ? formatRelative(thing.lastActiveTime) : '—'}
              </span>
            </div>
            <label className="flex items-center gap-2 text-xs text-muted">
              <span>启用设备</span>
              <Switch
                checked={thing.enable === 1}
                onChange={(next) => void toggleEnable(next)}
                disabled={updateMutation.isPending}
                label="启用设备"
              />
            </label>
          </div>
        </Card>

        <Card>
          <CardHeader>
            <div>
              <CardTitle>基本信息</CardTitle>
              <CardDescription>设备身份与所属物模型。</CardDescription>
            </div>
          </CardHeader>
          <CardContent className="space-y-3 text-xs">
            <InfoRow label="物模型">
              {modelQuery.data ? (
                <Link
                  to={`/platform/thing-models/${modelQuery.data.id}`}
                  className="text-primary hover:underline"
                >
                  {modelQuery.data.name}
                </Link>
              ) : (
                <span className="font-mono text-[11px]">{thing.thingModelId}</span>
              )}
            </InfoRow>
            <InfoRow label="Device Key">
              <CodeValue value={thing.deviceKey} />
            </InfoRow>
            <InfoRow label="上行 Topic">
              <CodeValue value={thing.upTopic} />
            </InfoRow>
            <InfoRow label="下行 Topic">
              <CodeValue value={thing.downTopic} />
            </InfoRow>
            <InfoRow label="创建时间">{formatDateTime(thing.createdTime)}</InfoRow>
            <InfoRow label="更新时间">{formatDateTime(thing.updatedTime)}</InfoRow>
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader>
            <div>
              <CardTitle>接入凭证</CardTitle>
              <CardDescription>
                MQTT 使用 clientId / username / password 认证;HTTP 上报使用 Token。
              </CardDescription>
            </div>
          </CardHeader>
          <CardContent className="space-y-1">
            <SecretRow label="MQTT Client ID" value={thing.mqttClientId} secret={false} />
            <SecretRow label="MQTT Username" value={thing.mqttUsername} secret={false} />
            <SecretRow label="MQTT Password" value={thing.mqttPassword} secret />
            <SecretRow label="Device Secret" value={thing.deviceSecret} secret />
            <SecretRow label="HTTP Token" value={thing.token} secret />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div>
              <CardTitle>设备图片</CardTitle>
              <CardDescription>用于列表与可视化展示。</CardDescription>
            </div>
            <ImageIcon className="size-4 text-faint" />
          </CardHeader>
          <CardContent>
            <ImageUploadField value={thing.image} onChange={(key) => void changeImage(key)} />
          </CardContent>
        </Card>

        <Card className="lg:col-span-3">
          <CardHeader>
            <div>
              <CardTitle>静态特征值</CardTitle>
              <CardDescription>
                特征定义来自物模型 {modelQuery.data ? `「${modelQuery.data.name}」` : ''};此处维护设备的具体取值。
              </CardDescription>
            </div>
            <Button
              size="sm"
              variant="outline"
              disabled={features.length === 0}
              onClick={openFeatures}
            >
              编辑特征值
            </Button>
          </CardHeader>
          <CardContent>
            <FeatureTable thing={thing} modelFeatures={features} />
          </CardContent>
        </Card>
      </div>

      <EditThingDialog open={editOpen} onClose={() => setEditOpen(false)} thing={thing} />

      <Dialog
        open={featureOpen}
        onClose={() => setFeatureOpen(false)}
        title="编辑特征值"
        description="只提交有值的特征;清空某项即从设备上移除该特征值。"
        size="lg"
        footer={
          <>
            <Button variant="ghost" onClick={() => setFeatureOpen(false)}>
              取消
            </Button>
            <Button variant="primary" onClick={saveFeatures} loading={updateMutation.isPending}>
              保存
            </Button>
          </>
        }
      >
        <div className="max-h-[60vh] overflow-y-auto pr-1">
          <FeatureValuesEditor
            features={features}
            value={featureDraft}
            onChange={setFeatureDraft}
          />
        </div>
      </Dialog>

      <ConfirmDialog
        open={deleteOpen}
        title="删除设备"
        description={`确定删除「${thing.deviceName}」吗?删除后设备凭证立即失效,该操作不可恢复。`}
        confirmText="删除"
        danger
        loading={deleteMutation.isPending}
        onConfirm={confirmDelete}
        onClose={() => setDeleteOpen(false)}
      />
    </div>
  )
}

function InfoRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-3">
      <span className="shrink-0 text-faint">{label}</span>
      <span className="min-w-0 text-right">{children}</span>
    </div>
  )
}

function CodeValue({ value }: { value: string }) {
  return (
    <span className="inline-flex max-w-64 items-center gap-1">
      <span className="truncate font-mono text-[11px]" title={value}>
        {value}
      </span>
      <CopyButton value={value} />
    </span>
  )
}

function SecretRow({
  label,
  value,
  secret,
}: {
  label: string
  value: string
  secret: boolean
}) {
  const [visible, setVisible] = useState(false)
  return (
    <div className="flex items-center justify-between gap-3 border-b border-border/60 py-2 last:border-0">
      <span className="shrink-0 text-xs text-faint">{label}</span>
      <span className="flex min-w-0 items-center gap-1">
        <span className="truncate font-mono text-[11px]" title={visible ? value : undefined}>
          {secret && !visible ? maskSecret(value) : value || '—'}
        </span>
        {secret ? (
          <Button
            size="icon"
            variant="ghost"
            aria-label={visible ? '隐藏' : '显示'}
            onClick={() => setVisible((current) => !current)}
          >
            {visible ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
          </Button>
        ) : null}
        <CopyButton value={value} />
      </span>
    </div>
  )
}

function FeatureTable({
  thing,
  modelFeatures,
}: {
  thing: ThingDetail
  modelFeatures: ThingModelFeature[]
}) {
  const values = thing.features ?? {}
  const defined = new Set(modelFeatures.map((feature) => feature.identifier))
  const extras = Object.keys(values).filter((key) => !defined.has(key))

  if (modelFeatures.length === 0 && extras.length === 0) {
    return (
      <EmptyState
        title="没有特征值"
        description="所属物模型未定义特征,或设备尚未填写特征值。"
      />
    )
  }

  return (
    <Table>
      <THead>
        <tr>
          <TH>特征</TH>
          <TH>标识符</TH>
          <TH>数据类型</TH>
          <TH>当前值</TH>
        </tr>
      </THead>
      <TBody>
        {modelFeatures.map((feature) => (
          <TR key={feature.identifier}>
            <TD className="font-medium">{feature.name}</TD>
            <TD className="font-mono text-xs text-muted">{feature.identifier}</TD>
            <TD className="text-xs text-muted">{describeDataType(feature.dataType)}</TD>
            <TD className="max-w-72 truncate font-mono text-[11px]" title={renderValue(values[feature.identifier])}>
              {renderValue(values[feature.identifier])}
            </TD>
          </TR>
        ))}
        {extras.map((key) => (
          <TR key={key}>
            <TD className="text-muted">未定义特征</TD>
            <TD className="font-mono text-xs text-muted">{key}</TD>
            <TD className="text-xs text-faint">—</TD>
            <TD className="max-w-72 truncate font-mono text-[11px]" title={renderValue(values[key])}>
              {renderValue(values[key])}
            </TD>
          </TR>
        ))}
      </TBody>
    </Table>
  )
}

function renderValue(value: unknown): string {
  if (value === undefined || value === null || value === '') return '—'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}
