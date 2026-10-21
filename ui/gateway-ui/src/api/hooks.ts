import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { errorMessage } from '@harns/core'
import {
  controlDevice,
  createDevice,
  deleteDevice,
  getDevice,
  getGatewayCpu,
  getGatewayDisk,
  getGatewayMem,
  getGatewayMeta,
  listDevices,
  switchDeviceStatus,
  updateDevice,
} from './gateway'
import { queryKeys } from './query-keys'
import type { DeviceOperator, DevicePayload, GatewayDeviceDetail } from './types'
import { appendOperationLog } from '@/lib/operation-log'

const DEVICE_POLL_MS = 10_000
const DEVICE_DETAIL_POLL_MS = 5_000
const RESOURCE_POLL_MS = 10_000

export function useDevices() {
  return useQuery({
    queryKey: queryKeys.gateway.devices,
    queryFn: () => listDevices(true),
    refetchInterval: DEVICE_POLL_MS,
  })
}

export function useDevice(id: string | undefined, poll = true) {
  return useQuery({
    queryKey: queryKeys.gateway.device(id ?? ''),
    queryFn: () => getDevice(id as string),
    enabled: Boolean(id),
    refetchInterval: poll ? DEVICE_DETAIL_POLL_MS : false,
  })
}

function invalidateDevice(
  queryClient: ReturnType<typeof useQueryClient>,
  id?: string,
): void {
  void queryClient.invalidateQueries({ queryKey: queryKeys.gateway.devices })
  if (id) {
    void queryClient.invalidateQueries({ queryKey: queryKeys.gateway.device(id) })
  }
}

export function useCreateDevice() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: DevicePayload) => createDevice(payload),
    onSuccess: (device: GatewayDeviceDetail, payload) => {
      appendOperationLog({
        action: 'create',
        result: 'success',
        target: payload.name,
      })
      invalidateDevice(queryClient, device.id)
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

export function useUpdateDevice() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; eTag: string; name: string; payload: DevicePayload }) =>
      updateDevice(vars.id, vars.eTag, vars.payload),
    onSuccess: (_device, vars) => {
      appendOperationLog({
        action: 'update',
        result: 'success',
        target: vars.name,
      })
      invalidateDevice(queryClient, vars.id)
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

export function useDeleteDevice() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; eTag: string; name: string }) =>
      deleteDevice(vars.id, vars.eTag),
    onSuccess: (_data, vars) => {
      appendOperationLog({
        action: 'delete',
        result: 'success',
        target: vars.name,
      })
      invalidateDevice(queryClient, vars.id)
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

export function useSwitchDeviceStatus() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; name: string; status: DeviceOperator }) =>
      switchDeviceStatus(vars.id, vars.status),
    onSuccess: (_data, vars) => {
      appendOperationLog({
        action: vars.status,
        result: 'success',
        target: vars.name,
      })
      // 状态切换是异步的,稍后再取一次最新状态。
      window.setTimeout(() => invalidateDevice(queryClient, vars.id), 600)
    },
    onError: (error, vars) => {
      appendOperationLog({
        action: vars.status,
        result: 'failure',
        target: vars.name,
        message: errorMessage(error),
      })
    },
  })
}

export function useControlDevice() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: {
      id: string
      name: string
      actions: Array<Record<string, unknown>>
    }) => controlDevice(vars.id, vars.actions),
    onSuccess: (_data, vars) => {
      appendOperationLog({
        action: 'control',
        result: 'success',
        target: vars.name,
      })
      invalidateDevice(queryClient, vars.id)
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

/* ---------------- 网关资源 ---------------- */

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
