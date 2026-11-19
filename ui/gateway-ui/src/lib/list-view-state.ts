import { useCallback, useEffect, useRef, useState, type Dispatch, type SetStateAction } from 'react'
import { useLocation, useNavigationType } from 'react-router-dom'
import { PAGE_SIZES } from '@harns/core'

/** 列表页视图缓存的 key，按页面隔离。 */
export const listViewKeys = {
  collections: 'gateway.list.collections',
  connectors: 'gateway.list.connectors',
  pushGroups: 'gateway.list.pushGroups',
  monitoringPoints: 'gateway.list.monitoringPoints',
  drivers: 'gateway.list.drivers',
  users: 'gateway.list.users',
} as const

interface ListViewLocationState {
  restoreListView?: boolean
}

const memory = new Map<string, unknown>()

/**
 * 是否恢复上次列表视图：
 * - 浏览器返回（POP）恢复；
 * - 详情页返回按钮通过 location.state 标记恢复；
 * - 从侧边栏菜单进入（PUSH/REPLACE）重置。
 */
export function useListViewRestore(): boolean {
  const navigationType = useNavigationType()
  const location = useLocation()
  const state = location.state as ListViewLocationState | null
  return useRef(navigationType === 'POP' || state?.restoreListView === true).current
}

/** 仅内存的 useState；restore 为 false 时清理旧值并回到默认值。 */
export function usePersistedState<T>(
  key: string,
  initial: T,
  restore: boolean,
): [T, Dispatch<SetStateAction<T>>] {
  const [value, setValue] = useState<T>(() => {
    if (restore && memory.has(key)) return memory.get(key) as T
    memory.delete(key)
    return initial
  })
  const valueRef = useRef(value)

  const setPersisted = useCallback<Dispatch<SetStateAction<T>>>(
    (next) => {
      const resolved =
        typeof next === 'function' ? (next as (prev: T) => T)(valueRef.current) : next
      valueRef.current = resolved
      memory.set(key, resolved)
      setValue(resolved)
    },
    [key],
  )

  return [value, setPersisted]
}

/** 列表页页码与每页条数。 */
export function useListPagination(key: string, restore: boolean) {
  const [page, setPage] = usePersistedState(`${key}:page`, 1, restore)
  const [pageSize, setPageSize] = usePersistedState(`${key}:pageSize`, 10, restore)

  const setPageSafe = useCallback(
    (next: number) => {
      setPage(Math.max(1, Math.trunc(next) || 1))
    },
    [setPage],
  )
  const setPageSizeSafe = useCallback(
    (next: number) => {
      setPageSize((PAGE_SIZES as readonly number[]).includes(next) ? next : 10)
    },
    [setPageSize],
  )

  return { page, pageSize, setPage: setPageSafe, setPageSize: setPageSizeSafe }
}

/** 数据量减少导致页码越界时回到最后一页。 */
export function useListPaginationClamp(
  page: number,
  pageSize: number,
  total: number,
  setPage: (page: number) => void,
) {
  useEffect(() => {
    if (total <= 0) return
    const lastPage = Math.max(1, Math.ceil(total / pageSize))
    if (page > lastPage) setPage(lastPage)
  }, [page, pageSize, total, setPage])
}
