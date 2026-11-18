import { useQueryClient } from '@tanstack/react-query'
import { AutoRefreshButton } from '@harns/core'

export function OverviewBreadcrumbAction() {
  const queryClient = useQueryClient()

  return (
    <AutoRefreshButton
      onRefresh={() => queryClient.invalidateQueries({ queryKey: ['gateway'] })}
    />
  )
}
