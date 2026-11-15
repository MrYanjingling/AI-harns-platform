import { useState } from 'react'
import { Plus, Search, UserRound } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  Card,
  ConfirmDialog,
  EmptyState,
  ErrorState,
  Input,
  Pagination,
  TableSkeleton,
  formatDateTime,
  showErrorToast,
  useI18n,
} from '@harns/core'
import {
  useDeleteUser,
  useResetUserPassword,
  useUsers,
} from '@/api/hooks'
import type { GatewayUser } from '@/api/types'
import {
  GatewayTBody,
  GatewayTD,
  GatewayTH,
  GatewayTHead,
  GatewayTR,
  GatewayTable,
} from '@/components/GatewayTable'
import { ColumnResizeHandle, useColumnResize } from '@/components/useColumnResize'
import { UserActionsMenu } from './UserActionsMenu'
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
    actions: 120,
  })

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
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="flex w-[260px]">
          <div className="relative min-w-0 flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
            <Input
              value={keywordInput}
              placeholder={t('gateway.users.searchPlaceholder')}
              className="h-9 rounded-r-none pl-9 text-sm"
              onChange={(event) => setKeywordInput(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') applySearch()
              }}
            />
          </div>
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

      <Card className="flex h-[calc(100dvh-212px)] min-h-[420px] flex-col overflow-hidden">
        <div className="min-h-0 flex-1 overflow-auto">
          {users.isLoading ? (
            <TableSkeleton rows={6} cols={5} />
          ) : users.isError ? (
            <div className="p-5">
              <ErrorState error={users.error} onRetry={users.refetch} compact />
            </div>
          ) : list.length === 0 ? (
            <EmptyState
              icon={<UserRound className="size-5" />}
              title={keyword ? t('gateway.users.emptyMatch') : t('gateway.users.emptyNone')}
              description={
                keyword
                  ? t('gateway.users.emptyMatchDescription')
                  : t('gateway.users.emptyNoneDescription')
              }
              action={
                keyword ? (
                  <Button variant="outline" onClick={resetSearch}>
                    {t('gateway.users.clearSearch')}
                  </Button>
                ) : (
                  <Button variant="primary" onClick={openCreate}>
                    {t('gateway.users.create')}
                  </Button>
                )
              }
            />
          ) : (
            <GatewayTable
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
              <GatewayTHead>
                <tr>
                  <GatewayTH className="text-center">
                    {t('common.field.order')}
                    <ColumnResizeHandle column="order" width={widths.order} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.users.table.user')}
                    <ColumnResizeHandle column="user" width={widths.user} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('common.field.remark')}
                    <ColumnResizeHandle column="remark" width={widths.remark} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('common.field.updatedAt')}
                    <ColumnResizeHandle
                      column="updatedAt"
                      width={widths.updatedAt}
                      onResize={resize}
                    />
                  </GatewayTH>
                  <GatewayTH className="text-right">
                    {t('common.field.actions')}
                    <ColumnResizeHandle
                      column="actions"
                      width={widths.actions}
                      onResize={resize}
                    />
                  </GatewayTH>
                </tr>
              </GatewayTHead>
              <GatewayTBody>
                {list.map((user, index) => (
                  <GatewayTR key={user.id}>
                    <GatewayTD className="text-center text-muted">
                      {(page - 1) * pageSize + index + 1}
                    </GatewayTD>
                    <GatewayTD>
                      <span className="text-fg">{user.username}</span>
                    </GatewayTD>
                    <GatewayTD>
                      <span className="text-fg">{user.remark || '—'}</span>
                    </GatewayTD>
                    <GatewayTD>
                      <span className="whitespace-nowrap text-muted">{formatDateTime(user.updatedTime)}</span>
                    </GatewayTD>
                    <GatewayTD className="text-right">
                      <div className="flex justify-end">
                        <UserActionsMenu
                          user={user}
                          onEdit={() => openEdit(user)}
                          onReset={() => setResetTarget(user)}
                          onDelete={() => setDeleteTarget(user)}
                        />
                      </div>
                    </GatewayTD>
                  </GatewayTR>
                ))}
              </GatewayTBody>
            </GatewayTable>
          )}
        </div>
        <div className="shrink-0">
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
        </div>
      </Card>

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
