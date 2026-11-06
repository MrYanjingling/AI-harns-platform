import { useState } from 'react'
import { KeyRound, Plus, Search, Trash2, UserRound } from 'lucide-react'
import { toast } from 'sonner'
import { useDeleteUser, useResetUserPassword, useUsers } from '@/api/hooks'
import type { UserInfo } from '@/api/types'
import { useI18n } from '@/i18n'
import { formatDateTime } from '@/lib/format'
import { errorMessage } from '@/lib/http'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState } from '@/components/ui/EmptyState'
import { Field, Input } from '@/components/ui/Field'
import { TableSkeleton } from '@/components/ui/Skeleton'
import { TBody, TD, TH, THead, TR, Table } from '@/components/ui/Table'
import { ErrorState } from '@/components/shared/ErrorState'
import { PageHeader } from '@/components/shared/PageHeader'
import { Pagination } from '@/components/shared/Pagination'
import { PlatformStatusBadge } from '@/components/shared/PlatformStatusBadge'
import { UserFormDialog } from './UserFormDialog'

export function UsersPage() {
  const { t } = useI18n()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<UserInfo | undefined>(undefined)
  const [resetTarget, setResetTarget] = useState<UserInfo | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<UserInfo | null>(null)

  const users = useUsers({ page, pageSize, keyword: keyword || undefined })
  const resetMutation = useResetUserPassword()
  const deleteMutation = useDeleteUser()

  const list = users.data?.list ?? []
  const total = users.data?.total ?? 0

  const applySearch = () => {
    setKeyword(keywordInput.trim())
    setPage(1)
  }

  const resetSearch = () => {
    setKeywordInput('')
    setKeyword('')
    setPage(1)
  }

  const openCreate = () => {
    setEditing(undefined)
    setFormOpen(true)
  }

  const openEdit = (user: UserInfo) => {
    setEditing(user)
    setFormOpen(true)
  }

  const confirmReset = async () => {
    if (!resetTarget) return
    try {
      await resetMutation.mutateAsync(resetTarget.id)
      toast.success(t('identity.users.resetToast', { name: resetTarget.userName }))
      setResetTarget(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteMutation.mutateAsync(deleteTarget.id)
      toast.success(t('identity.users.deletedToast', { name: deleteTarget.userName }))
      setDeleteTarget(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <div>
      <PageHeader
        title={t('identity.users.title')}
        description={t('identity.users.description')}
        actions={
          <Button variant="primary" icon={<Plus className="size-4" />} onClick={openCreate}>
            {t('identity.users.create')}
          </Button>
        }
      />

      <Card className="mb-4 flex flex-wrap items-end gap-3 p-4">
        <Field label={t('common.action.search')} className="min-w-56 flex-1">
          <div className="relative">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
            <Input
              value={keywordInput}
              placeholder={t('identity.users.searchPlaceholder')}
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
        {users.isLoading ? (
          <TableSkeleton rows={6} cols={5} />
        ) : users.isError ? (
          <div className="p-5">
            <ErrorState error={users.error} onRetry={users.refetch} compact />
          </div>
        ) : list.length === 0 ? (
          <EmptyState
            icon={<UserRound className="size-5" />}
            title={keyword ? t('identity.users.emptyMatch') : t('identity.users.emptyNone')}
            description={
              keyword
                ? t('identity.users.emptyMatchDescription')
                : t('identity.users.emptyNoneDescription')
            }
            action={
              keyword ? (
                <Button variant="outline" onClick={resetSearch}>
                  {t('identity.users.clearSearch')}
                </Button>
              ) : (
                <Button variant="primary" onClick={openCreate}>
                  {t('identity.users.create')}
                </Button>
              )
            }
          />
        ) : (
          <>
            <Table>
              <THead>
                <tr>
                  <TH>{t('identity.users.table.user')}</TH>
                  <TH>{t('identity.users.table.userNo')}</TH>
                  <TH>{t('identity.users.table.roles')}</TH>
                  <TH>{t('common.field.status')}</TH>
                  <TH>{t('identity.users.table.lastLogin')}</TH>
                  <TH className="text-right">{t('common.field.actions')}</TH>
                </tr>
              </THead>
              <TBody>
                {list.map((user) => (
                  <TR key={user.id}>
                    <TD>
                      <p className="font-medium">{user.nickName}</p>
                      <p className="font-mono text-[11px] text-faint">{user.userName}</p>
                    </TD>
                    <TD className="text-xs text-muted">{user.userNo}</TD>
                    <TD className="max-w-56">
                      <div className="flex flex-wrap gap-1">
                        {(user.roles ?? []).length > 0 ? (
                          user.roles.map((role) => (
                            <Badge key={role.id} tone="primary">
                              {role.name}
                            </Badge>
                          ))
                        ) : (
                          <span className="text-xs text-faint">
                            {t('identity.unassigned')}
                          </span>
                        )}
                      </div>
                    </TD>
                    <TD>
                      <PlatformStatusBadge status={user.status} />
                    </TD>
                    <TD className="text-xs text-muted">
                      {user.lastLoginTime
                        ? formatDateTime(user.lastLoginTime)
                        : t('identity.users.neverLoggedIn')}
                    </TD>
                    <TD>
                      <div className="flex justify-end gap-1">
                        <Button size="sm" variant="ghost" onClick={() => openEdit(user)}>
                          {t('common.action.edit')}
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label={t('identity.users.resetPasswordAria', { name: user.userName })}
                          title={t('identity.users.resetPassword')}
                          onClick={() => setResetTarget(user)}
                        >
                          <KeyRound className="size-3.5" />
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label={t('identity.users.deleteAria', { name: user.userName })}
                          onClick={() => setDeleteTarget(user)}
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
              disabled={users.isFetching}
            />
          </>
        )}
      </Card>

      {formOpen ? (
        <UserFormDialog open={formOpen} user={editing} onClose={() => setFormOpen(false)} />
      ) : null}

      <ConfirmDialog
        open={Boolean(resetTarget)}
        title={t('identity.users.resetTitle')}
        description={
          resetTarget
            ? t('identity.users.resetDescription', { name: resetTarget.userName })
            : undefined
        }
        confirmText={t('identity.users.resetConfirm')}
        loading={resetMutation.isPending}
        onConfirm={confirmReset}
        onClose={() => setResetTarget(null)}
      />

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title={t('identity.users.deleteTitle')}
        description={
          deleteTarget
            ? t('identity.users.deleteDescription', { name: deleteTarget.userName })
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
