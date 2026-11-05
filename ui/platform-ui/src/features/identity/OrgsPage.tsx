import { useMemo, useState } from 'react'
import { Network, Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { useDeleteOrg, useOrgTree } from '@/api/hooks'
import type { OrgInfo } from '@/api/types'
import { errorMessage } from '@/lib/http'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState } from '@/components/ui/EmptyState'
import { TableSkeleton } from '@/components/ui/Skeleton'
import { TBody, TD, TH, THead, TR, Table } from '@/components/ui/Table'
import { ErrorState } from '@/components/shared/ErrorState'
import { PageHeader } from '@/components/shared/PageHeader'
import { PlatformStatusBadge } from '@/components/shared/PlatformStatusBadge'
import { OrgFormDialog } from './OrgFormDialog'
import { findOrg, flattenOrgs } from './org-utils'

export function OrgsPage() {
  const orgTree = useOrgTree()
  const deleteMutation = useDeleteOrg()
  const [formState, setFormState] = useState<{
    open: boolean
    org?: OrgInfo
    parentId?: string
  }>({ open: false })
  const [deleteTarget, setDeleteTarget] = useState<OrgInfo | null>(null)

  const tree = orgTree.data ?? []
  const rows = useMemo(() => flattenOrgs(tree), [tree])

  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteMutation.mutateAsync(deleteTarget.id)
      toast.success(`组织「${deleteTarget.name}」已删除`)
      setDeleteTarget(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <div>
      <PageHeader
        title="组织管理"
        description="按树状结构维护部门与层级;存在子组织时不能删除,需先删除子组织。"
        actions={
          <Button
            variant="primary"
            icon={<Plus className="size-4" />}
            onClick={() => setFormState({ open: true })}
          >
            新建根组织
          </Button>
        }
      />

      <Card className="overflow-hidden">
        {orgTree.isLoading ? (
          <TableSkeleton rows={6} cols={5} />
        ) : orgTree.isError ? (
          <div className="p-5">
            <ErrorState error={orgTree.error} onRetry={orgTree.refetch} compact />
          </div>
        ) : rows.length === 0 ? (
          <EmptyState
            icon={<Network className="size-5" />}
            title="还没有组织"
            description="创建根组织后,可继续添加下级部门。"
            action={
              <Button
                variant="primary"
                icon={<Plus className="size-4" />}
                onClick={() => setFormState({ open: true })}
              >
                新建根组织
              </Button>
            }
          />
        ) : (
          <Table>
            <THead>
              <tr>
                <TH>组织</TH>
                <TH>编码</TH>
                <TH>排序</TH>
                <TH>状态</TH>
                <TH>描述</TH>
                <TH className="text-right">操作</TH>
              </tr>
            </THead>
            <TBody>
              {rows.map((row) => {
                const org = findOrg(tree, row.id)
                if (!org) return null
                return (
                  <TR key={org.id}>
                    <TD>
                      <div style={{ paddingLeft: `${row.depth * 20}px` }}>
                        <span className={row.depth > 0 ? 'text-muted' : 'font-medium'}>
                          {row.depth > 0 ? '└ ' : ''}
                          {org.name}
                        </span>
                        {(org.children?.length ?? 0) > 0 ? (
                          <span className="ml-1.5 text-[10px] text-faint">
                            {org.children?.length} 个子组织
                          </span>
                        ) : null}
                      </div>
                    </TD>
                    <TD className="font-mono text-xs text-muted">{org.code}</TD>
                    <TD className="text-xs text-muted tabular">{org.sort ?? 0}</TD>
                    <TD>
                      <PlatformStatusBadge status={org.status} />
                    </TD>
                    <TD className="max-w-56 truncate text-xs text-muted" title={org.description}>
                      {org.description || '—'}
                    </TD>
                    <TD>
                      <div className="flex justify-end gap-1">
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() =>
                            setFormState({ open: true, parentId: org.id })
                          }
                        >
                          添加下级
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => setFormState({ open: true, org })}
                        >
                          编辑
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label={`删除 ${org.name}`}
                          onClick={() => setDeleteTarget(org)}
                        >
                          <Trash2 className="size-3.5 text-danger" />
                        </Button>
                      </div>
                    </TD>
                  </TR>
                )
              })}
            </TBody>
          </Table>
        )}
      </Card>

      {formState.open ? (
        <OrgFormDialog
          open={formState.open}
          tree={tree}
          org={formState.org}
          defaultParentId={formState.parentId}
          onClose={() => setFormState({ open: false })}
        />
      ) : null}

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="删除组织"
        description={
          deleteTarget
            ? `确定删除组织「${deleteTarget.name}」吗?用户绑定会一并清理;若仍有子组织,后端会拒绝删除。`
            : undefined
        }
        confirmText="删除"
        danger
        loading={deleteMutation.isPending}
        onConfirm={confirmDelete}
        onClose={() => setDeleteTarget(null)}
      />
    </div>
  )
}
