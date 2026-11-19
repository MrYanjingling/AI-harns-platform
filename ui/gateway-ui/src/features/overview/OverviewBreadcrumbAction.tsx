import { useQueryClient } from '@tanstack/react-query'
import { AutoRefreshButton } from '@harns/core'
import { queryKeys } from '@/api/query-keys'

const overviewKeys = [
  queryKeys.gateway.meta,
  queryKeys.gateway.cpu,
  queryKeys.gateway.mem,
  queryKeys.gateway.disk,
  queryKeys.gateway.license,
  queryKeys.gateway.systemInfo,
]

export function OverviewBreadcrumbAction() {
  const queryClient = useQueryClient()

  return (
    <AutoRefreshButton
      onRefresh={() => {
        for (const queryKey of overviewKeys) {
          void queryClient.refetchQueries({ queryKey, type: 'all' })
        }
        void queryClient.refetchQueries({
          queryKey: ['gateway', 'collections'],
          type: 'all',
        })
        void queryClient.refetchQueries({
          queryKey: ['gateway', 'push'],
          type: 'all',
        })
      }}
    />
  )
}
