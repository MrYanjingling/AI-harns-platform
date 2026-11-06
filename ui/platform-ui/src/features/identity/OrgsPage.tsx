import { useMemo, useState } from 'react'
import { Network, Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { useDeleteOrg, useOrgTree } from '@/api/hooks'
import type { OrgInfo } from '@/api/types'
import { useI18n } from '@/i18n'
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
  const { t } = useI18n()
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
      toast.success(t('identity.orgs.deletedToast', { name: deleteTarget.name }))
      setDeleteTarget(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <div>
      <PageHeader
        title={t('identity.orgs.title')}
        description={t('identity.orgs.description')}
        actions={
          <Button
            variant="primary"
            icon={<Plus className="size-4" />}
            onClick={() => setFormState({ open: true })}
          >
            {t('identity.orgs.createRoot')}
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
            title={t('identity.orgs.emptyTitle')}
            description={t('identity.orgs.emptyDescription')}
            action={
              <Button
                variant="primary"
                icon={<Plus className="size-4" />}
                onClick={() => setFormState({ open: true })}
              >
                {t('identity.orgs.createRoot')}
              </Button>
            }
          />
        ) : (
          <Table>
            <THead>
              <tr>
                <TH>{t('identity.orgs.table.org')}</TH>
                <TH>{t('common.field.code')}</TH>
                <TH>{t('identity.orgs.table.order')}</TH>
                <TH>{t('common.field.status')}</TH>
                <TH>{t('common.field.description')}</TH>
                <TH className="text-right">{t('common.field.actions')}</TH>
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
                            {t('identity.orgs.childCount', { count: org.children?.length ?? 0 })}
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
                          {t('identity.orgs.addChild')}
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => setFormState({ open: true, org })}
                        >
                          {t('common.action.edit')}
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label={t('identity.orgs.deleteAria', { name: org.name })}
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
        title={t('identity.orgs.deleteTitle')}
        description={
          deleteTarget
            ? t('identity.orgs.deleteDescription', { name: deleteTarget.name })
            : undefined
        }
        confirmText={t('common.action.delete')}
        danger
        loading={deleteMutation.isPending}
        onConfirm={confirmDelete}
        onClose={() => setDeleteTarget(null)}
      />
    </div>
  )
}
