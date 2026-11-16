import type { Collection } from '@/api/types'

type CollectionLifecycle = Pick<Collection, 'enable' | 'collectStatus'>

export function canStartCollection(collection: CollectionLifecycle): boolean {
  return collection.enable && collection.collectStatus !== 'collecting'
}

export function canStopCollection(collection: CollectionLifecycle): boolean {
  return collection.collectStatus !== 'stopped'
}

export function canRestartCollection(collection: CollectionLifecycle): boolean {
  return collection.enable
}

export function canToggleCollectionEnable(collection: CollectionLifecycle): boolean {
  return !collection.enable || collection.collectStatus === 'stopped'
}
