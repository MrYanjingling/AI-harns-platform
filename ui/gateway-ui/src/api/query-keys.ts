export const queryKeys = {
  gatewayMeta: ['gateway', 'meta'] as const,
  collections: ['collections'] as const,
  collection: (id: string) => ['collections', id] as const,
  drivers: ['drivers'] as const,
  pushConnectors: ['push', 'connectors'] as const,
  pushGroups: ['push', 'groups'] as const,
}
