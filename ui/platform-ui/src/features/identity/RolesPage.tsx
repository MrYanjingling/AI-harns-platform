import { useState } from 'react'
import { Plus, Search, ShieldCheck, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { useDeleteRole, useRoles } from '@/api/hooks'
import type { RoleInfo } from '@/api/types'
import { useI18n } from '@harns/core'
import { formatDateTime } from '@harns/core'
import { errorMessage } from '@harns/core'
import { Badge } from '@harns/core'
import { Button } from '@harns/core'
import { Card } from '@harns/core'
import { ConfirmDialog } from '@harns/core'
import { EmptyState } from '@harns/core'
import { Field, Input } from '@harns/core'
import { TableSkeleton } from '@harns/core'
import { TBody, TD, TH, THead, TR, Table } from '@harns/core'
import { ErrorState } from '@harns/core'
import { PageHeader } from '@harns/core'
import { Pagination } from '@harns/core'
import { PlatformStatusBadge } from '@/components/shared/PlatformStatusBadge'
import { RoleFormDialog } from './RoleFormDialog'

export function RolesPage() {
  const { t } = useI18n()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<RoleInfo | undefined>(undefined)
  const [deleteTarget, setDeleteTarget] = useState<RoleInfo | null>(null)

  const roles = useRoles({ page, pageSize, keyword: keyword || undefined })
  const deleteMutation = useDeleteRole()

  const list = roles.data?.list ?? []
  const total = roles.data?.total ?? 0

  const applySearch = () => {
    setKeyword(keywordInput.trim())
    setPage(1)
  }

  const resetSearch = () => {
    setKeywordInput('')
    setKeyword('')
    setPage(1)
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteMutation.mutateAsync(deleteTarget.id)
      toast.success(t('identity.roles.deletedToast', { name: deleteTarget.name }))
      setDeleteTarget(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <div>
      <PageHeader
        title={t('identity.roles.title')}
        description={t('identity.roles.description')}
        actions={
          <Button
            variant="primary"
            icon={<Plus className="size-4" />}
            onClick={() => {
              setEditing(undefined)
              setFormOpen(true)
            }}
          >
            {t('identity.roles.create')}
          </Button>
        }
      />

      <Card className="mb-4 flex flex-wrap items-end gap-3 p-4">
        <Field label={t('common.action.search')} className="min-w-56 flex-1">
          <div className="relative">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
            <Input
              value={keywordInput}
              placeholder={t('identity.roles.searchPlaceholder')}
              className="pl-9"
              onChange={(event) => setKeywordInput(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') applySearch()
              }}
            />
          </div>
        </Field>
        <Button variant="outline" onClick={applySearch}>
          {t('common.action.query')}
        </Button>
        <Button variant="ghost" onClick={resetSearch}>
          {t('common.action.reset')}
        </Button>
      </Card>

      <Card className="overflow-hidden">
        {roles.isLoading ? (
          <TableSkeleton rows={6} cols={5} />
        ) : roles.isError ? (
          <div className="p-5">
            <ErrorState error={roles.error} onRetry={roles.refetch} compact />
          </div>
        ) : list.length === 0 ? (
          <EmptyState
            icon={<ShieldCheck className="size-5" />}
            title={keyword ? t('identity.roles.emptyMatch') : t('identity.roles.emptyNone')}
            description={
              keyword
                ? t('identity.roles.emptyMatchDescription')
                : t('identity.roles.emptyNoneDescription')
            }
            action={
              keyword ? (
                <Button variant="outline" onClick={resetSearch}>
                  {t('identity.users.clearSearch')}
                </Button>
              ) : (
                <Button
                  variant="primary"
                  onClick={() => {
                    setEditing(undefined)
                    setFormOpen(true)
                  }}
                >
                  {t('identity.roles.create')}
                </Button>
              )
            }
          />
        ) : (
          <>
            <Table>
              <THead>
                <tr>
                  <TH>{t('identity.roles.table.role')}</TH>
                  <TH>{t('common.field.code')}</TH>
                  <TH>{t('common.field.description')}</TH>
                  <TH>{t('common.field.status')}</TH>
                  <TH>{t('common.field.updatedAt')}</TH>
                  <TH className="text-right">{t('common.field.actions')}</TH>
                </tr>
              </THead>
              <TBody>
                {list.map((role) => (
                  <TR key={role.id}>
                    <TD>
                      <p className="font-medium">{role.name}</p>
                      {role.code === 'admin' ? (
                        <Badge tone="violet" className="mt-1">
                          {t('identity.roles.builtin')}
                        </Badge>
                      ) : null}
                    </TD>
                    <TD className="font-mono text-xs text-muted">{role.code}</TD>
                    <TD className="max-w-64 truncate text-xs text-muted" title={role.description}>
                      {role.description || '—'}
                    </TD>
                    <TD>
                      <PlatformStatusBadge status={role.status} />
                    </TD>
                    <TD className="text-xs text-muted">{formatDateTime(role.updatedTime)}</TD>
                    <TD>
                      <div className="flex justify-end gap-1">
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => {
                            setEditing(role)
                            setFormOpen(true)
                          }}
                        >
                          {t('common.action.edit')}
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label={t('identity.roles.deleteAria', { name: role.name })}
                          onClick={() => setDeleteTarget(role)}
                        >
                          <Trash2 className="size-3.5 text-danger" />
                        </Button>
                      </div>
                    </TD>
                  </TR>
                ))}
              </TBody>
            </Table>
            <Pagination
              page={page}
              pageSize={pageSize}
              total={total}
              onPageChange={setPage}
              onPageSizeChange={(size) => {
                setPageSize(size)
                setPage(1)
              }}
              disabled={roles.isFetching}
            />
          </>
        )}
      </Card>

      {formOpen ? (
        <RoleFormDialog open={formOpen} role={editing} onClose={() => setFormOpen(false)} />
      ) : null}

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title={t('identity.roles.deleteTitle')}
        description={
          deleteTarget
            ? t('identity.roles.deleteDescription', { name: deleteTarget.name })
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
