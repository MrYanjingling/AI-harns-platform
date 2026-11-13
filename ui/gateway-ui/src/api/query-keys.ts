export const queryKeys = {
  gateway: {
    collections: (params: object) => ['gateway', 'collections', params] as const,
    collection: (id: string) => ['gateway', 'collections', id] as const,
    variables: (id: string, params: object) =>
      ['gateway', 'collections', id, 'variables', params] as const,
    driverGroups: ['gateway', 'drivers', 'groups'] as const,
    drivers: ['gateway', 'drivers', 'flat'] as const,
    driverSchema: (model: string) => ['gateway', 'drivers', 'schema', model] as const,
    connectors: (params: object) =>
      ['gateway', 'push', 'connectors', params] as const,
    pushGroups: (params: object) =>
      ['gateway', 'push', 'groups', params] as const,
    meta: ['gateway', 'meta'] as const,
    cpu: ['gateway', 'cpu'] as const,
    mem: ['gateway', 'mem'] as const,
    disk: ['gateway', 'disk'] as const,
  },
}
