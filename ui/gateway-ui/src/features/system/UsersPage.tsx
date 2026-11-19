import { useState } from 'react'
import { KeyRound, Pencil, Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  ConfirmDialog,
  ErrorState,
  ListTableCard,
  SearchInput,
  Pagination,
  TableSkeleton,
  formatDateTime,
  showErrorToast,
  useI18n,
} from '@harns/core'
import { TableActions } from '@harns/core'
import {
  useDeleteUser,
  useResetUserPassword,
  useUsers,
} from '@/api/hooks'
import type { GatewayUser } from '@/api/types'
import {
  DataTableBody,
  DataTableCell,
  DataTableHeadCell,
  DataTableHead,
  DataTableRow,
  DataTable,
} from '@harns/core'
import { ColumnResizeHandle, useColumnResize } from '@harns/core'
import { TableActionButton } from '@harns/core'
import { UserFormDialog } from './UserFormDialog'

export function UsersPage() {
  const { t } = useI18n()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<GatewayUser | undefined>(undefined)
  const [resetTarget, setResetTarget] = useState<GatewayUser | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<GatewayUser | null>(null)

  const users = useUsers({ page, pageSize, keyword: keyword || undefined })
  const resetMutation = useResetUserPassword()
  const deleteMutation = useDeleteUser()
  const list = users.data?.list ?? []
  const total = users.data?.total ?? 0
  const { widths, totalWidth, resize } = useColumnResize({
    order: 80,
    user: 280,
    remark: 460,
    updatedAt: 260,
    actions: 140,
  })

  const applySearch = () => {
    const next = keywordInput.trim()
    const unchanged = page === 1 && next === keyword
    setKeyword(next)
    setPage(1)
    if (unchanged) void users.refetch()
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

  const openEdit = (user: GatewayUser) => {
    setEditing(user)
    setFormOpen(true)
  }

  const confirmReset = async () => {
    if (!resetTarget) return
    try {
      await resetMutation.mutateAsync(resetTarget.id)
      toast.success(t('gateway.users.resetToast', { name: resetTarget.username }))
      setResetTarget(null)
    } catch (error) {
      showErrorToast(error)
    }
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteMutation.mutateAsync(deleteTarget.id)
      toast.success(t('gateway.users.deletedToast', { name: deleteTarget.username }))
      setDeleteTarget(null)
    } catch (error) {
      showErrorToast(error)
    }
  }

  return (
    <div>
      <div className="app-page-toolbar">
        <div className="flex w-[260px]">
          <SearchInput
            wrapperClassName="min-w-0 flex-1"
            inputClassName="rounded-r-none"
            value={keywordInput}
            placeholder={t('gateway.users.searchPlaceholder')}
            onChange={(event) => setKeywordInput(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') applySearch()
            }}
          />
          <Button variant="primary" className="h-9 rounded-l-none px-4" onClick={applySearch}>
            {t('common.action.search')}
          </Button>
        </div>
        <Button variant="ghost" className="h-9" onClick={resetSearch}>
          {t('common.action.reset')}
        </Button>
        <Button
          variant="primary"
          className="ml-auto h-9"
          icon={<Plus className="size-3.5" />}
          onClick={openCreate}
        >
          {t('gateway.users.create')}
        </Button>
      </div>

      <ListTableCard className="h-[calc(100dvh-212px)] min-h-[420px]"
        pagination={
          <Pagination
            page={page}
            pageSize={pageSize}
            total={total}
            onPageChange={setPage}
            onPageSizeChange={(size) => {
              setPageSize(size)
              setPage(1)
            }}
            disabled={users.isFetching || users.isLoading}
          />
        }
      >
          {users.isLoading ? (
            <TableSkeleton rows={6} cols={5} />
          ) : users.isError ? (
            <div className="p-5">
              <ErrorState error={users.error} onRetry={users.refetch} compact />
            </div>
          ) : list.length === 0 ? null : (
            <DataTable
              style={{
                tableLayout: 'fixed',
                width: `max(100%, ${totalWidth}px)`,
              }}
            >
              <colgroup>
                <col style={{ width: widths.order }} />
                <col style={{ width: widths.user }} />
                <col style={{ width: widths.remark }} />
                <col style={{ width: widths.updatedAt }} />
                <col style={{ width: widths.actions }} />
              </colgroup>
              <DataTableHead>
                <tr>
                  <DataTableHeadCell className="text-center">
                    {t('common.field.order')}
                    <ColumnResizeHandle column="order" width={widths.order} onResize={resize} />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.users.table.user')}
                    <ColumnResizeHandle column="user" width={widths.user} onResize={resize} />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('common.field.remark')}
                    <ColumnResizeHandle column="remark" width={widths.remark} onResize={resize} />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('common.field.updatedAt')}
                    <ColumnResizeHandle
                      column="updatedAt"
                      width={widths.updatedAt}
                      onResize={resize}
                    />
                  </DataTableHeadCell>
                  <DataTableHeadCell className="text-left">
                    {t('common.field.actions')}
                    <ColumnResizeHandle
                      column="actions"
                      width={widths.actions}
                      onResize={resize}
                    />
                  </DataTableHeadCell>
                </tr>
              </DataTableHead>
              <DataTableBody>
                {list.map((user, index) => (
                  <DataTableRow key={user.id}>
                    <DataTableCell className="text-center">
                      {(page - 1) * pageSize + index + 1}
                    </DataTableCell>
                    <DataTableCell>
                      <span>{user.username}</span>
                    </DataTableCell>
                    <DataTableCell>
                      <span>{user.remark || '—'}</span>
                    </DataTableCell>
                    <DataTableCell>
                      <span className="whitespace-nowrap">{formatDateTime(user.updatedTime)}</span>
                    </DataTableCell>
                    <DataTableCell>
                      <TableActions>
                        <TableActionButton
                          label={t('common.action.edit')}
                          icon={<Pencil />}
                          onClick={() => openEdit(user)}
                        />
                        <TableActionButton
                          label={t('gateway.users.resetPassword')}
                          icon={<KeyRound />}
                          onClick={() => setResetTarget(user)}
                        />
                        <TableActionButton
                          label={t('common.action.delete')}
                          icon={<Trash2 />}
                          disabled={user.isAdmin}
                          onClick={() => setDeleteTarget(user)}
                        />
                      </TableActions>
                    </DataTableCell>
                  </DataTableRow>
                ))}
              </DataTableBody>
            </DataTable>
          )}
      </ListTableCard>

      <UserFormDialog open={formOpen} user={editing} onClose={() => setFormOpen(false)} />

      <ConfirmDialog
        open={Boolean(resetTarget)}
        title={t('gateway.users.resetTitle')}
        description={
          resetTarget ? t('gateway.users.resetDescription', { name: resetTarget.username }) : undefined
        }
        confirmText={t('gateway.users.resetConfirm')}
        loading={resetMutation.isPending}
        onConfirm={confirmReset}
        onClose={() => setResetTarget(null)}
      />

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title={t('gateway.users.deleteTitle')}
        description={
          deleteTarget
            ? t('gateway.users.deleteDescription', { name: deleteTarget.username })
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
