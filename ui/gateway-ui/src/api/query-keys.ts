export const queryKeys = {
  gateway: {
    all: ['gateway'] as const,
    devices: ['gateway', 'devices'] as const,
    device: (id: string) => ['gateway', 'devices', id] as const,
    meta: ['gateway', 'meta'] as const,
    cpu: ['gateway', 'cpu'] as const,
    mem: ['gateway', 'mem'] as const,
    disk: ['gateway', 'disk'] as const,
  },
}
