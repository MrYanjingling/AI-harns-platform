import type {
  AuditQuery,
  KpiQueryRequest,
  MetricsQueryRequest,
  TagTimeSeriesRequest,
  TimeSeriesSource,
} from './types'
import type { IdentityQuery } from './identity'
import type { ThingModelQuery } from './thing-models'
import type { ThingQuery } from './things'

export const queryKeys = {
  gateway: {
    all: ['gateway'] as const,
    meta: ['gateway', 'meta'] as const,
    cpu: ['gateway', 'cpu'] as const,
    mem: ['gateway', 'mem'] as const,
    disk: ['gateway', 'disk'] as const,
  },
  devices: {
    all: ['devices'] as const,
    list: (exploded: boolean) => ['devices', 'list', exploded] as const,
    detail: (id: string) => ['devices', 'detail', id] as const,
  },
  data: {
    metrics: ['data', 'metrics'] as const,
    kpis: ['data', 'kpis'] as const,
    tags: ['data', 'tags'] as const,
    metricsValues: (req: MetricsQueryRequest) => ['data', 'metricsValues', req] as const,
    kpiValues: (req: KpiQueryRequest) => ['data', 'kpiValues', req] as const,
    timeSeries: (source: TimeSeriesSource, req: TagTimeSeriesRequest) =>
      ['data', 'timeSeries', source, req] as const,
  },
  platform: {
    currentUser: ['platform', 'current-user'] as const,
    users: {
      all: ['platform', 'users'] as const,
      list: (query: IdentityQuery) => ['platform', 'users', 'list', query] as const,
      detail: (id: string) => ['platform', 'users', 'detail', id] as const,
    },
    roles: {
      all: ['platform', 'roles'] as const,
      list: (query: IdentityQuery) => ['platform', 'roles', 'list', query] as const,
    },
    orgs: {
      all: ['platform', 'orgs'] as const,
      tree: ['platform', 'orgs', 'tree'] as const,
    },
    audit: {
      all: ['platform', 'audit'] as const,
      list: (query: AuditQuery) => ['platform', 'audit', 'list', query] as const,
    },
    thingModels: {
      all: ['platform', 'thing-models'] as const,
      list: (query: ThingModelQuery) => ['platform', 'thing-models', 'list', query] as const,
      detail: (id: string) => ['platform', 'thing-models', 'detail', id] as const,
    },
    things: {
      all: ['platform', 'things'] as const,
      list: (query: ThingQuery) => ['platform', 'things', 'list', query] as const,
      detail: (id: string) => ['platform', 'things', 'detail', id] as const,
    },
  },
}
