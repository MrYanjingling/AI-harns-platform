import { buildQuery } from '@/lib/http'
import { platformData, platformPage } from './platform'
import type {
  CreateThingModelRequest,
  PlatformPage,
  ThingModelDetail,
  ThingModelDocument,
  ThingModelInfo,
  ThingModelProperty,
  ThingModelPropertyInput,
  UpdateThingModelRequest,
} from './types'

export interface ThingModelQuery {
  page: number
  pageSize: number
  keyword?: string
}

export function listThingModels(query: ThingModelQuery): Promise<PlatformPage<ThingModelInfo>> {
  return platformPage<ThingModelInfo>('/iot/thingModels', {
    page: query.page,
    pageSize: query.pageSize,
    keyword: query.keyword,
  })
}

export function getThingModel(id: string): Promise<ThingModelDetail> {
  return platformData<ThingModelDetail>(`/iot/thingModels/${id}`)
}

export function createThingModel(payload: CreateThingModelRequest): Promise<ThingModelDetail> {
  return platformData<ThingModelDetail>('/iot/thingModels', { method: 'POST', body: payload })
}

export function updateThingModel(
  id: string,
  payload: UpdateThingModelRequest,
): Promise<ThingModelInfo> {
  return platformData<ThingModelInfo>(`/iot/thingModels/${id}`, { method: 'PUT', body: payload })
}

export function deleteThingModel(id: string, cascade = false): Promise<null> {
  return platformData<null>(`/iot/thingModels/${id}${buildQuery({ cascade })}`, {
    method: 'DELETE',
  })
}

export function addThingModelProperties(
  id: string,
  properties: ThingModelPropertyInput[],
): Promise<ThingModelProperty[]> {
  return platformData<ThingModelProperty[]>(`/iot/thingModels/${id}/properties`, {
    method: 'POST',
    body: properties,
  })
}

export function updateThingModelProperty(
  id: string,
  identifier: string,
  property: ThingModelPropertyInput,
): Promise<ThingModelProperty> {
  return platformData<ThingModelProperty>(
    `/iot/thingModels/${id}/properties/${encodeURIComponent(identifier)}`,
    { method: 'PUT', body: property },
  )
}

export function deleteThingModelProperties(
  id: string,
  identifiers: string[],
): Promise<{ deleted: number }> {
  return platformData<{ deleted: number }>(`/iot/thingModels/${id}/properties`, {
    method: 'DELETE',
    body: identifiers,
  })
}

export function exportThingModel(id: string): Promise<ThingModelDocument> {
  return platformData<ThingModelDocument>(`/iot/thingModels/${id}/tsl`)
}

/** 导入会整体替换属性;features 省略时保留,提供时整体替换。 */
export function importThingModel(
  id: string,
  document: ThingModelDocument,
): Promise<ThingModelDetail> {
  return platformData<ThingModelDetail>(`/iot/thingModels/${id}/tsl`, {
    method: 'PUT',
    body: document,
  })
}
