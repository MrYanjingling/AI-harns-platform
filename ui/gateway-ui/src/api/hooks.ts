import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { QueryClient } from '@tanstack/react-query'
import { errorMessage } from '@harns/core'
import {
  addCollectionVariables,
  createCollection,
  createDriver,
  createPushConnector,
  createPushGroup,
  deleteCollection,
  deleteCollectionVariables,
  deleteDriver,
  deletePushConnector,
  deletePushGroup,
  deliverCollectionAction,
  getCollection,
  getDriverSchema,
  getGatewayCpu,
  getGatewayDisk,
  getGatewayMem,
  getGatewayMeta,
  listCollectionVariables,
  listCollections,
  listDriverGroups,
  listDriversFlat,
  listPushConnectors,
  listPushGroups,
  restartCollection,
  setCollectionEnable,
  startCollection,
  startPushConnector,
  stopCollection,
  stopPushConnector,
  updateCollection,
  updateDriver,
  updatePushConnector,
  updatePushGroup,
} from './gateway'
import { queryKeys } from './query-keys'
import type {
  CollectionAction,
  CollectionPayload,
  CollectionVariableInput,
  DriverPayload,
  PushConnectorPayload,
  PushGroupPayload,
} from './types'
import { appendOperationLog } from '@/lib/operation-log'

const COLLECTION_POLL_MS = 10_000
const COLLECTION_DETAIL_POLL_MS = 5_000
const PUSH_POLL_MS = 10_000
const RESOURCE_POLL_MS = 10_000

export interface CollectionQueryParams {
  page: number
  pageSize: number
  keyword?: string
  driverType?: string
  driverModel?: string
}

export interface PageParams {
  page: number
  pageSize: number
  keyword?: string
}

/* ---------------- 采集器 ---------------- */

export function useCollections(params: CollectionQueryParams) {
  return useQuery({
    queryKey: queryKeys.gateway.collections(params),
    queryFn: () => listCollections(params),
    refetchInterval: COLLECTION_POLL_MS,
    placeholderData: keepPreviousData,
  })
}

export function useCollection(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.gateway.collection(id ?? ''),
    queryFn: () => getCollection(id as string),
    enabled: Boolean(id),
    refetchInterval: COLLECTION_DETAIL_POLL_MS,
  })
}

export function useCollectionVariables(id: string | undefined, params: PageParams) {
  return useQuery({
    queryKey: queryKeys.gateway.variables(id ?? '', params),
    queryFn: () => listCollectionVariables(id as string, params),
    enabled: Boolean(id),
    refetchInterval: COLLECTION_DETAIL_POLL_MS,
    placeholderData: keepPreviousData,
  })
}

function invalidateCollection(queryClient: QueryClient, id?: string): void {
  void queryClient.invalidateQueries({ queryKey: ['gateway', 'collections'] })
  if (id) void queryClient.invalidateQueries({ queryKey: queryKeys.gateway.collection(id) })
}

export function useCreateCollection() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CollectionPayload) => createCollection(payload),
    onSuccess: (collection, payload) => {
      appendOperationLog({ action: 'create', result: 'success', target: payload.name })
      invalidateCollection(queryClient, collection.id)
    },
    onError: (error, payload) => {
      appendOperationLog({
        action: 'create',
        result: 'failure',
        target: payload.name,
        message: errorMessage(error),
      })
    },
  })
}

export function useUpdateCollection() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; name: string; payload: CollectionPayload }) =>
      updateCollection(vars.id, vars.payload),
    onSuccess: (_data, vars) => {
      appendOperationLog({ action: 'update', result: 'success', target: vars.name })
      invalidateCollection(queryClient, vars.id)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: 'update',
        result: 'failure',
        target: vars.name,
        message: errorMessage(error),
      })
    },
  })
}

export function useDeleteCollection() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; name: string }) => deleteCollection(vars.id),
    onSuccess: (_data, vars) => {
      appendOperationLog({ action: 'delete', result: 'success', target: vars.name })
      invalidateCollection(queryClient, vars.id)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: 'delete',
        result: 'failure',
        target: vars.name,
        message: errorMessage(error),
      })
    },
  })
}

type CollectionRuntimeAction = 'start' | 'stop' | 'restart'

export function useCollectionRuntimeAction() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (vars: {
      id: string
      name: string
      action: CollectionRuntimeAction
    }) => {
      if (vars.action === 'start') await startCollection(vars.id)
      else if (vars.action === 'stop') await stopCollection(vars.id)
      else await restartCollection(vars.id)
    },
    onSuccess: (_data, vars) => {
      appendOperationLog({ action: vars.action, result: 'success', target: vars.name })
      // 状态切换是异步的,稍后再取一次最新状态。
      window.setTimeout(() => invalidateCollection(queryClient, vars.id), 600)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: vars.action,
        result: 'failure',
        target: vars.name,
        message: errorMessage(error),
      })
    },
  })
}

export function useSetCollectionEnable() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; name: string; enable: boolean }) =>
      setCollectionEnable(vars.id, vars.enable),
    onSuccess: (_data, vars) => {
      appendOperationLog({
        action: 'enable',
        result: 'success',
        target: `${vars.name} → ${vars.enable ? 'on' : 'off'}`,
      })
      invalidateCollection(queryClient, vars.id)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: 'enable',
        result: 'failure',
        target: vars.name,
        message: errorMessage(error),
      })
    },
  })
}

export function useAddCollectionVariables() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: {
      id: string
      name: string
      variables: CollectionVariableInput[]
    }) => addCollectionVariables(vars.id, vars.variables),
    onSuccess: (_data, vars) => {
      appendOperationLog({
        action: 'create',
        result: 'success',
        target: `${vars.name} · ${vars.variables.length} variables`,
      })
      invalidateCollection(queryClient, vars.id)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: 'create',
        result: 'failure',
        target: vars.name,
        message: errorMessage(error),
      })
    },
  })
}

export function useDeleteCollectionVariables() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; name: string; names: string[] }) =>
      deleteCollectionVariables(vars.id, vars.names),
    onSuccess: (_data, vars) => {
      appendOperationLog({
        action: 'delete',
        result: 'success',
        target: `${vars.name} · ${vars.names.join(', ')}`,
      })
      invalidateCollection(queryClient, vars.id)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: 'delete',
        result: 'failure',
        target: vars.name,
        message: errorMessage(error),
      })
    },
  })
}

export function useDeliverCollectionAction() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; name: string; actions: CollectionAction[] }) =>
      deliverCollectionAction(vars.id, vars.actions),
    onSuccess: (_data, vars) => {
      appendOperationLog({
        action: 'control',
        result: 'success',
        target: `${vars.name} · ${vars.actions.map((item) => item.name).join(', ')}`,
      })
      invalidateCollection(queryClient, vars.id)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: 'control',
        result: 'failure',
        target: vars.name,
        message: errorMessage(error),
      })
    },
  })
}

/* ---------------- 驱动 ---------------- */

export function useDriverGroups() {
  return useQuery({
    queryKey: queryKeys.gateway.driverGroups,
    queryFn: () => listDriverGroups(),
    staleTime: 60_000,
  })
}

export function useDriverList() {
  return useQuery({
    queryKey: queryKeys.gateway.drivers,
    queryFn: () => listDriversFlat(),
    staleTime: 60_000,
  })
}

export function useDriverSchema(model: string | undefined) {
  return useQuery({
    queryKey: queryKeys.gateway.driverSchema(model ?? ''),
    queryFn: () => getDriverSchema(model as string),
    enabled: Boolean(model),
    staleTime: 5 * 60_000,
  })
}

function invalidateDrivers(queryClient: QueryClient): void {
  void queryClient.invalidateQueries({ queryKey: ['gateway', 'drivers'] })
}

export function useCreateDriver() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: DriverPayload) => createDriver(payload),
    onSuccess: (driver) => {
      appendOperationLog({
        action: 'create',
        result: 'success',
        target: `${driver.driverType}/${driver.model}`,
      })
      invalidateDrivers(queryClient)
    },
    onError: (error, payload) => {
      appendOperationLog({
        action: 'create',
        result: 'failure',
        target: `${payload.driverType}/${payload.model}`,
        message: errorMessage(error),
      })
    },
  })
}

export function useUpdateDriver() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; payload: DriverPayload }) =>
      updateDriver(vars.id, vars.payload),
    onSuccess: (_driver, vars) => {
      appendOperationLog({
        action: 'update',
        result: 'success',
        target: `${vars.payload.driverType}/${vars.payload.model}`,
      })
      invalidateDrivers(queryClient)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: 'update',
        result: 'failure',
        target: `${vars.payload.driverType}/${vars.payload.model}`,
        message: errorMessage(error),
      })
    },
  })
}

export function useDeleteDriver() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; title: string }) => deleteDriver(vars.id),
    onSuccess: (_data, vars) => {
      appendOperationLog({ action: 'delete', result: 'success', target: vars.title })
      invalidateDrivers(queryClient)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: 'delete',
        result: 'failure',
        target: vars.title,
        message: errorMessage(error),
      })
    },
  })
}

/* ---------------- 推送 ---------------- */

export function usePushConnectors(params: PageParams) {
  return useQuery({
    queryKey: queryKeys.gateway.connectors(params),
    queryFn: () => listPushConnectors(params),
    refetchInterval: PUSH_POLL_MS,
    placeholderData: keepPreviousData,
  })
}

export function usePushGroups(params: PageParams) {
  return useQuery({
    queryKey: queryKeys.gateway.pushGroups(params),
    queryFn: () => listPushGroups(params),
    refetchInterval: PUSH_POLL_MS,
    placeholderData: keepPreviousData,
  })
}

function invalidatePush(queryClient: QueryClient): void {
  void queryClient.invalidateQueries({ queryKey: ['gateway', 'push'] })
}

export function useCreatePushConnector() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: PushConnectorPayload) => createPushConnector(payload),
    onSuccess: (connector) => {
      appendOperationLog({ action: 'create', result: 'success', target: connector.name })
      invalidatePush(queryClient)
    },
    onError: (error, payload) => {
      appendOperationLog({
        action: 'create',
        result: 'failure',
        target: payload.name,
        message: errorMessage(error),
      })
    },
  })
}

export function useUpdatePushConnector() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; payload: PushConnectorPayload }) =>
      updatePushConnector(vars.id, vars.payload),
    onSuccess: (_data, vars) => {
      appendOperationLog({ action: 'update', result: 'success', target: vars.payload.name })
      invalidatePush(queryClient)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: 'update',
        result: 'failure',
        target: vars.payload.name,
        message: errorMessage(error),
      })
    },
  })
}

export function useDeletePushConnector() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; name: string; cascade: boolean }) =>
      deletePushConnector(vars.id, vars.cascade),
    onSuccess: (_data, vars) => {
      appendOperationLog({ action: 'delete', result: 'success', target: vars.name })
      invalidatePush(queryClient)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: 'delete',
        result: 'failure',
        target: vars.name,
        message: errorMessage(error),
      })
    },
  })
}

export function usePushConnectorAction() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; name: string; action: 'start' | 'stop' }) =>
      vars.action === 'start' ? startPushConnector(vars.id) : stopPushConnector(vars.id),
    onSuccess: (_data, vars) => {
      appendOperationLog({
        action: vars.action,
        result: 'success',
        target: `MQTT · ${vars.name}`,
      })
      window.setTimeout(() => invalidatePush(queryClient), 400)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: vars.action,
        result: 'failure',
        target: `MQTT · ${vars.name}`,
        message: errorMessage(error),
      })
    },
  })
}

export function useCreatePushGroup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: PushGroupPayload) => createPushGroup(payload),
    onSuccess: (group) => {
      appendOperationLog({ action: 'push', result: 'success', target: group.name })
      invalidatePush(queryClient)
    },
    onError: (error, payload) => {
      appendOperationLog({
        action: 'push',
        result: 'failure',
        target: payload.name,
        message: errorMessage(error),
      })
    },
  })
}

export function useUpdatePushGroup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; payload: PushGroupPayload }) =>
      updatePushGroup(vars.id, vars.payload),
    onSuccess: (_data, vars) => {
      appendOperationLog({ action: 'push', result: 'success', target: vars.payload.name })
      invalidatePush(queryClient)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: 'push',
        result: 'failure',
        target: vars.payload.name,
        message: errorMessage(error),
      })
    },
  })
}

export function useDeletePushGroup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; name: string }) => deletePushGroup(vars.id),
    onSuccess: (_data, vars) => {
      appendOperationLog({ action: 'push', result: 'success', target: vars.name })
      invalidatePush(queryClient)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: 'push',
        result: 'failure',
        target: vars.name,
        message: errorMessage(error),
      })
    },
  })
}

/* ---------------- 网关资源 ---------------- */

export function useGatewayMeta() {
  return useQuery({ queryKey: queryKeys.gateway.meta, queryFn: getGatewayMeta })
}

export function useGatewayCpu() {
  return useQuery({
    queryKey: queryKeys.gateway.cpu,
    queryFn: getGatewayCpu,
    refetchInterval: RESOURCE_POLL_MS,
  })
}

export function useGatewayMem() {
  return useQuery({
    queryKey: queryKeys.gateway.mem,
    queryFn: getGatewayMem,
    refetchInterval: RESOURCE_POLL_MS,
  })
}

export function useGatewayDisk() {
  return useQuery({
    queryKey: queryKeys.gateway.disk,
    queryFn: getGatewayDisk,
    refetchInterval: RESOURCE_POLL_MS,
  })
}
