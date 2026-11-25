import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { QueryClient } from '@tanstack/react-query'
import { errorMessage } from '@harns/core'
import {
  addCollectionVariables,
  createCollection,
  createDriver,
  createPushConnector,
  createPushGroup,
  createSimulatorPoint,
  createUser,
  deleteCollection,
  deleteCollectionVariables,
  deleteDriver,
  deletePushConnector,
  deletePushGroup,
  deleteSimulatorPoint,
  deleteUser,
  deliverCollectionAction,
  getAISettings,
  getCollection,
  getDriverSchema,
  getGatewayCpu,
  getGatewayDisk,
  getGatewayMem,
  getGatewayMeta,
  getLicense,
  getSimulator,
  getSystemInfo,
  getSystemUpdate,
  applySystemUpdate,
  rejectSystemUpdate,
  updateSystemUpdateSettings,
  importLicense,
  listUsers,
  listCollectionVariables,
  listCollections,
  listDriverGroups,
  listDriversFlat,
  login,
  closeSimulator,
  listPushConnectors,
  listPushGroups,
  openSimulator,
  resetUserPassword,
  restartCollection,
  restartPushConnector,
  restartPushGroup,
  setCollectionEnable,
  startCollection,
  startPushConnector,
  startPushGroup,
  stopCollection,
  stopPushConnector,
  stopPushGroup,
  testAIModel,
  updateAIModel,
  updateAISettings,
  updateCollection,
  updateCollectionVariables,
  updateDriver,
  updatePushConnector,
  updatePushGroup,
  updateUser,
  uploadDriverPlugin,
} from './gateway'
import { queryKeys } from './query-keys'
import type {
  AIModelPayload,
  CollectionAction,
  CollectionPayload,
  CollectionVariableInput,
  DriverPayload,
  GatewayUserPayload,
  LoginPayload,
  PushConnectorPayload,
  PushGroupPayload,
  SimulatorDataType,
  SimulatorMode,
} from './types'
import { appendOperationLog } from '@/lib/operation-log'


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

export interface PushConnectorParams extends PageParams {
  type?: string
}

export interface PushGroupParams extends PageParams {
  connectorId?: string
}

/* ---------------- 采集器 ---------------- */

export function useCollections(params: CollectionQueryParams) {
  return useQuery({
    queryKey: queryKeys.gateway.collections(params),
    queryFn: () => listCollections(params),
    placeholderData: keepPreviousData,
  })
}

export function useCollection(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.gateway.collection(id ?? ''),
    queryFn: () => getCollection(id as string),
    enabled: Boolean(id),
  })
}

export function useCollectionVariables(id: string | undefined, params: PageParams) {
  return useQuery({
    queryKey: queryKeys.gateway.variables(id ?? '', params),
    queryFn: () => listCollectionVariables(id as string, params),
    enabled: Boolean(id),
    placeholderData: keepPreviousData,
  })
}

export function useAllCollectionVariables(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.gateway.variables(id ?? '', { all: true }),
    queryFn: async () => {
      const pageSize = 100
      const first = await listCollectionVariables(id as string, { page: 1, pageSize })
      const all = [...first.list]
      const pages = Math.ceil(first.total / pageSize)
      for (let page = 2; page <= pages; page += 1) {
        const next = await listCollectionVariables(id as string, { page, pageSize })
        all.push(...next.list)
      }
      return all
    },
    enabled: Boolean(id),
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
      void queryClient.invalidateQueries({
        queryKey: ['gateway', 'collections', vars.id, 'variables'],
      })
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
      void queryClient.invalidateQueries({
        queryKey: ['gateway', 'collections', vars.id, 'variables'],
      })
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

export function useUpdateCollectionVariables() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: {
      id: string
      name: string
      variables: CollectionVariableInput[]
    }) => updateCollectionVariables(vars.id, vars.variables),
    onSuccess: (_data, vars) => {
      invalidateCollection(queryClient, vars.id)
      void queryClient.invalidateQueries({
        queryKey: ['gateway', 'collections', vars.id, 'variables'],
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
      void queryClient.invalidateQueries({
        queryKey: ['gateway', 'collections', vars.id, 'variables'],
      })
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
  })
}

export function useDriverList() {
  return useQuery({
    queryKey: queryKeys.gateway.drivers,
    queryFn: () => listDriversFlat(),
  })
}

export function useDriverSchema(model: string | undefined) {
  return useQuery({
    queryKey: queryKeys.gateway.driverSchema(model ?? ''),
    queryFn: () => getDriverSchema(model as string),
    enabled: Boolean(model),
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

export function useUploadDriverPlugin() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; title: string; file: File }) =>
      uploadDriverPlugin(vars.id, vars.file),
    onSuccess: (_driver, vars) => {
      appendOperationLog({ action: 'update', result: 'success', target: vars.title })
      invalidateDrivers(queryClient)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: 'update',
        result: 'failure',
        target: vars.title,
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

export function usePushConnectors(params: PushConnectorParams) {
  return useQuery({
    queryKey: queryKeys.gateway.connectors(params),
    queryFn: () => listPushConnectors(params),
    placeholderData: keepPreviousData,
  })
}

export function usePushGroups(params: PushGroupParams, enabled = true) {
  return useQuery({
    queryKey: queryKeys.gateway.pushGroups(params),
    queryFn: () => listPushGroups(params),
    enabled,
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
    mutationFn: (vars: { id: string; name: string; action: 'start' | 'stop' | 'restart' }) => {
      if (vars.action === 'start') return startPushConnector(vars.id)
      if (vars.action === 'stop') return stopPushConnector(vars.id)
      return restartPushConnector(vars.id)
    },
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
      invalidatePush(queryClient)
    },
  })
}

export function usePushGroupAction() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; name: string; action: 'start' | 'stop' | 'restart' }) => {
      if (vars.action === 'start') return startPushGroup(vars.id)
      if (vars.action === 'stop') return stopPushGroup(vars.id)
      return restartPushGroup(vars.id)
    },
    onSuccess: (_data, vars) => {
      appendOperationLog({
        action: vars.action,
        result: 'success',
        target: vars.name,
      })
      window.setTimeout(() => invalidatePush(queryClient), 400)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: vars.action,
        result: 'failure',
        target: vars.name,
        message: errorMessage(error),
      })
      invalidatePush(queryClient)
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
  return useQuery({
    queryKey: queryKeys.gateway.meta,
    queryFn: getGatewayMeta,
  })
}

export function useSystemInfo() {
  return useQuery({
    queryKey: queryKeys.gateway.systemInfo,
    queryFn: getSystemInfo,
  })
}

export function useLicense() {
  return useQuery({
    queryKey: queryKeys.gateway.license,
    queryFn: getLicense,
  })
}

export function useImportLicense() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: string) => importLicense(data),
    onSuccess: (status) => {
      queryClient.setQueryData(queryKeys.gateway.license, status)
    },
  })
}

/* ---------------- 认证 ---------------- */

export function useLogin() {
  return useMutation({
    mutationFn: (payload: LoginPayload) => login(payload),
  })
}

/* ---------------- 用户 ---------------- */


export function useUsers(params: PageParams) {
  return useQuery({
    queryKey: queryKeys.gateway.users(params),
    queryFn: () => listUsers(params),
    placeholderData: keepPreviousData,
  })
}

function invalidateUsers(queryClient: QueryClient): void {
  void queryClient.invalidateQueries({ queryKey: ['gateway', 'users'] })
}

export function useCreateUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: GatewayUserPayload) => createUser(payload),
    onSuccess: () => invalidateUsers(queryClient),
  })
}

export function useUpdateUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; payload: Omit<GatewayUserPayload, 'password'> }) =>
      updateUser(vars.id, vars.payload),
    onSuccess: () => invalidateUsers(queryClient),
  })
}

export function useDeleteUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteUser(id),
    onSuccess: () => invalidateUsers(queryClient),
  })
}

export function useResetUserPassword() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => resetUserPassword(id),
    onSuccess: () => invalidateUsers(queryClient),
  })
}

export function useGatewayCpu() {
  return useQuery({
    queryKey: queryKeys.gateway.cpu,
    queryFn: getGatewayCpu,
  })
}

export function useGatewayMem() {
  return useQuery({
    queryKey: queryKeys.gateway.mem,
    queryFn: getGatewayMem,
  })
}

export function useGatewayDisk() {
  return useQuery({
    queryKey: queryKeys.gateway.disk,
    queryFn: getGatewayDisk,
  })
}


/* ---------------- 设备模拟 ---------------- */

export function useSimulator() {
  return useQuery({
    queryKey: queryKeys.gateway.simulator,
    queryFn: getSimulator,
  })
}

function invalidateSimulator(queryClient: QueryClient): void {
  void queryClient.invalidateQueries({ queryKey: queryKeys.gateway.simulator })
}

export function useOpenSimulator() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (port: number) => openSimulator(port),
    onSuccess: () => invalidateSimulator(queryClient),
  })
}

export function useCloseSimulator() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => closeSimulator(),
    onSuccess: () => invalidateSimulator(queryClient),
  })
}

export function useCreateSimulatorPoint() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: {
      address: number
      name: string
      description: string
      dataType: SimulatorDataType
      mode: SimulatorMode
    }) => createSimulatorPoint(vars.address, vars.name, vars.description, vars.dataType, vars.mode),
    onSuccess: () => invalidateSimulator(queryClient),
  })
}

export function useDeleteSimulatorPoint() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (address: number) => deleteSimulatorPoint(address),
    onSuccess: () => invalidateSimulator(queryClient),
  })
}

export function useSaveSimulatorPoints() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (vars: {
      created: {
        address: number
        name: string
        description: string
        dataType: SimulatorDataType
        mode: SimulatorMode
      }[]
      deleted: number[]
    }) => {
      for (const address of vars.deleted) {
        await deleteSimulatorPoint(address)
      }
      for (const point of vars.created) {
        await createSimulatorPoint(
          point.address,
          point.name,
          point.description,
          point.dataType,
          point.mode,
        )
      }
    },
    onSettled: () => invalidateSimulator(queryClient),
  })
}

/* ---------------- AI 配置 ---------------- */

export function useAISettings() {
  return useQuery({
    queryKey: queryKeys.gateway.aiSettings,
    queryFn: getAISettings,
  })
}

export function useUpdateAISettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: { assistantEnabled: boolean; mcpEnabled: boolean }) =>
      updateAISettings(payload),
    onSuccess: (settings) => {
      queryClient.setQueryData(queryKeys.gateway.aiSettings, settings)
    },
  })
}

export function useUpdateAIModel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: AIModelPayload) => updateAIModel(payload),
    onSuccess: (settings) => {
      queryClient.setQueryData(queryKeys.gateway.aiSettings, settings)
    },
  })
}

export function useTestAIModel() {
  return useMutation({
    mutationFn: (payload: AIModelPayload) => testAIModel(payload),
  })
}

/* ---------------- OTA ---------------- */

export function useSystemUpdate() {
  return useQuery({
    queryKey: queryKeys.gateway.systemUpdate,
    queryFn: getSystemUpdate,
    refetchInterval: (query) => {
      const state = query.state.data?.task?.state
      return state === 'received' ||
        state === 'downloading' ||
        state === 'applying' ||
        state === 'restarting'
        ? 3000
        : 15000
    },
  })
}

export function useApplySystemUpdate() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: applySystemUpdate,
    onSuccess: (state) => {
      queryClient.setQueryData(queryKeys.gateway.systemUpdate, state)
      void queryClient.invalidateQueries({ queryKey: queryKeys.gateway.systemUpdate })
    },
  })
}

export function useRejectSystemUpdate() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: rejectSystemUpdate,
    onSuccess: (state) => {
      queryClient.setQueryData(queryKeys.gateway.systemUpdate, state)
      void queryClient.invalidateQueries({ queryKey: queryKeys.gateway.systemUpdate })
    },
  })
}

export function useUpdateSystemUpdateSettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: updateSystemUpdateSettings,
    onSuccess: (state) => {
      queryClient.setQueryData(queryKeys.gateway.systemUpdate, state)
      void queryClient.invalidateQueries({ queryKey: queryKeys.gateway.systemUpdate })
    },
  })
}
