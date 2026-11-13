import { buildQuery } from '@harns/core'
import { platformData, platformPage } from './platform'
import type {
  CreateThingsRequest,
  DeleteThingsResult,
  PlatformPage,
  ThingDetail,
  ThingInfo,
  UpdateThingRequest,
} from './types'

export interface ThingQuery {
  page: number
  pageSize: number
  keyword?: string
  thingModelId?: string
}

export function listThings(query: ThingQuery): Promise<PlatformPage<ThingInfo>> {
  return platformPage<ThingInfo>('/iot/things', {
    page: query.page,
    pageSize: query.pageSize,
    keyword: query.keyword,
    thingModelId: query.thingModelId,
  })
}

export function getThing(id: string): Promise<ThingDetail> {
  return platformData<ThingDetail>(`/iot/things/${id}`)
}

export function createThings(payload: CreateThingsRequest): Promise<ThingDetail[]> {
  return platformData<ThingDetail[]>('/iot/things', { method: 'POST', body: payload })
}

export function updateThing(id: string, payload: UpdateThingRequest): Promise<ThingDetail> {
  return platformData<ThingDetail>(`/iot/things/${id}`, { method: 'PUT', body: payload })
}

export function deleteThings(ids: string[], cascade = false): Promise<DeleteThingsResult> {
  return platformData<DeleteThingsResult>(`/iot/things${buildQuery({ cascade })}`, {
    method: 'DELETE',
    body: ids,
  })
}
