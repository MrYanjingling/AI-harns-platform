import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { getCurrentUser, login, logout } from './auth'
import { listAuditLogs } from './audit'
import { uploadImage } from './files'
import { getGatewayCpu, getGatewayDisk, getGatewayMem, getGatewayMeta } from './gateway'
import {
  createOrg,
  createRole,
  createUser,
  deleteOrg,
  deleteRole,
  deleteUser,
  getOrgTree,
  getUser,
  listRoles,
  listUsers,
  resetUserPassword,
  updateOrg,
  updateRole,
  updateUser,
  type IdentityQuery,
} from './identity'
import { queryKeys } from './query-keys'
import {
  addThingModelProperties,
  createThingModel,
  deleteThingModel,
  deleteThingModelProperties,
  getThingModel,
  importThingModel,
  listThingModels,
  updateThingModel,
  updateThingModelProperty,
  type ThingModelQuery,
} from './thing-models'
import {
  createThings,
  deleteThings,
  getThing,
  listThings,
  updateThing,
  type ThingQuery,
} from './things'
import type {
  AuditQuery,
  CreateOrgRequest,
  CreateRoleRequest,
  CreateThingModelRequest,
  CreateThingsRequest,
  CreateUserRequest,
  LoginRequest,
  ThingModelDocument,
  ThingModelPropertyInput,
  UpdateOrgRequest,
  UpdateRoleRequest,
  UpdateThingModelRequest,
  UpdateThingRequest,
  UpdateUserRequest,
} from './types'

const RESOURCE_POLL_MS = 10_000

/* ---------------- 网关 ---------------- */

export function useGatewayMeta() {
  return useQuery({
    queryKey: queryKeys.gateway.meta,
    queryFn: getGatewayMeta,
  })
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

/* ---------------- 平台:认证 ---------------- */

export function useCurrentUser(enabled = true) {
  return useQuery({
    queryKey: queryKeys.platform.currentUser,
    queryFn: () => getCurrentUser(),
    enabled,
    staleTime: 5 * 60_000,
    retry: false,
  })
}

export function useLogin() {
  return useMutation({ mutationFn: (req: LoginRequest) => login(req) })
}

export function useLogout() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => logout(),
    // 无论服务端是否成功,本地会话都要清理。
    onSettled: () => queryClient.clear(),
  })
}

/* ---------------- 平台:用户 ---------------- */

export function useUsers(query: IdentityQuery) {
  return useQuery({
    queryKey: queryKeys.platform.users.list(query),
    queryFn: () => listUsers(query),
  })
}

export function useUser(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.platform.users.detail(id ?? ''),
    queryFn: () => getUser(id as string),
    enabled: Boolean(id),
  })
}

export function useCreateUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateUserRequest) => createUser(payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.platform.users.all }),
  })
}

export function useUpdateUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; payload: UpdateUserRequest }) =>
      updateUser(vars.id, vars.payload),
    onSuccess: (_data, vars) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.platform.users.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.platform.users.detail(vars.id) })
    },
  })
}

export function useDeleteUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteUser(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.platform.users.all }),
  })
}

export function useResetUserPassword() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => resetUserPassword(id),
    onSuccess: (_data, id) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.platform.users.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.platform.users.detail(id) })
    },
  })
}

/* ---------------- 平台:角色 ---------------- */

export function useRoles(query: IdentityQuery) {
  return useQuery({
    queryKey: queryKeys.platform.roles.list(query),
    queryFn: () => listRoles(query),
  })
}

export function useCreateRole() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateRoleRequest) => createRole(payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.platform.roles.all }),
  })
}

export function useUpdateRole() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; payload: UpdateRoleRequest }) =>
      updateRole(vars.id, vars.payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.platform.roles.all }),
  })
}

export function useDeleteRole() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteRole(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.platform.roles.all }),
  })
}

/* ---------------- 平台:组织 ---------------- */

export function useOrgTree() {
  return useQuery({
    queryKey: queryKeys.platform.orgs.tree,
    queryFn: getOrgTree,
  })
}

export function useCreateOrg() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateOrgRequest) => createOrg(payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.platform.orgs.all }),
  })
}

export function useUpdateOrg() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; payload: UpdateOrgRequest }) =>
      updateOrg(vars.id, vars.payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.platform.orgs.all }),
  })
}

export function useDeleteOrg() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteOrg(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.platform.orgs.all }),
  })
}

/* ---------------- 平台:审计日志 ---------------- */

export function useAuditLogs(query: AuditQuery) {
  return useQuery({
    queryKey: queryKeys.platform.audit.list(query),
    queryFn: () => listAuditLogs(query),
  })
}

/* ---------------- 平台:物模型 ---------------- */

export function useThingModels(query: ThingModelQuery) {
  return useQuery({
    queryKey: queryKeys.platform.thingModels.list(query),
    queryFn: () => listThingModels(query),
  })
}

export function useThingModel(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.platform.thingModels.detail(id ?? ''),
    queryFn: () => getThingModel(id as string),
    enabled: Boolean(id),
  })
}

export function useCreateThingModel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateThingModelRequest) => createThingModel(payload),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: queryKeys.platform.thingModels.all }),
  })
}

export function useUpdateThingModel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; payload: UpdateThingModelRequest }) =>
      updateThingModel(vars.id, vars.payload),
    onSuccess: (_data, vars) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.platform.thingModels.all })
      void queryClient.invalidateQueries({
        queryKey: queryKeys.platform.thingModels.detail(vars.id),
      })
    },
  })
}

export function useDeleteThingModel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteThingModel(id),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: queryKeys.platform.thingModels.all }),
  })
}

export function useAddThingModelProperties() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; properties: ThingModelPropertyInput[] }) =>
      addThingModelProperties(vars.id, vars.properties),
    onSuccess: (_data, vars) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.platform.thingModels.all })
      void queryClient.invalidateQueries({
        queryKey: queryKeys.platform.thingModels.detail(vars.id),
      })
    },
  })
}

export function useUpdateThingModelProperty() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: {
      id: string
      identifier: string
      property: ThingModelPropertyInput
    }) => updateThingModelProperty(vars.id, vars.identifier, vars.property),
    onSuccess: (_data, vars) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.platform.thingModels.all })
      void queryClient.invalidateQueries({
        queryKey: queryKeys.platform.thingModels.detail(vars.id),
      })
    },
  })
}

export function useDeleteThingModelProperties() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; identifiers: string[] }) =>
      deleteThingModelProperties(vars.id, vars.identifiers),
    onSuccess: (_data, vars) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.platform.thingModels.all })
      void queryClient.invalidateQueries({
        queryKey: queryKeys.platform.thingModels.detail(vars.id),
      })
    },
  })
}

export function useImportThingModel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; document: ThingModelDocument }) =>
      importThingModel(vars.id, vars.document),
    onSuccess: (_data, vars) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.platform.thingModels.all })
      void queryClient.invalidateQueries({
        queryKey: queryKeys.platform.thingModels.detail(vars.id),
      })
    },
  })
}

/* ---------------- 平台:设备(Thing) ---------------- */

export function useThings(query: ThingQuery) {
  return useQuery({
    queryKey: queryKeys.platform.things.list(query),
    queryFn: () => listThings(query),
    refetchInterval: 10_000,
  })
}

export function useThing(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.platform.things.detail(id ?? ''),
    queryFn: () => getThing(id as string),
    enabled: Boolean(id),
    refetchInterval: 5_000,
  })
}

export function useCreateThings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateThingsRequest) => createThings(payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.platform.things.all }),
  })
}

export function useUpdateThing() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; payload: UpdateThingRequest }) =>
      updateThing(vars.id, vars.payload),
    onSuccess: (_data, vars) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.platform.things.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.platform.things.detail(vars.id) })
    },
  })
}

export function useDeleteThings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (ids: string[]) => deleteThings(ids),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.platform.things.all }),
  })
}

/* ---------------- 平台:文件 ---------------- */

export function useUploadImage() {
  return useMutation({ mutationFn: (file: File) => uploadImage(file) })
}
