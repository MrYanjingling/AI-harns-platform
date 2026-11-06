import { useMemo, useState } from 'react'
import { ChevronDown, ListTree, Search } from 'lucide-react'
import type { DeviceDetail, DeviceType } from '@/api/types'
import { useI18n } from '@/i18n'
import { cn } from '@/lib/cn'
import { Input, Select } from '@/components/ui/Field'
import { buildDeviceTree, filterTree, type DeviceTreeNode, type TreeMode } from './device-list'

function TreeRow({
  node,
  depth,
  activeProtocol,
  activeModel,
  open,
  onToggle,
  onSelect,
}: {
  node: DeviceTreeNode
  depth: number
  activeProtocol: DeviceType | ''
  activeModel: string
  open: boolean
  onToggle: () => void
  onSelect: (node: DeviceTreeNode) => void
}) {
  const { t } = useI18n()
  const hasChildren = Boolean(node.children?.length)
  const active =
    activeProtocol === node.protocol &&
    (node.model ? activeModel === node.model : !activeModel)

  return (
    <>
      <div
        className={cn(
          'flex items-center gap-1 rounded-sm py-1 pr-2 text-sm',
          active ? 'bg-primary-soft font-medium text-primary' : 'hover:bg-surface-2',
        )}
        style={{ paddingLeft: depth * 14 + 4 }}
      >
        {hasChildren ? (
          <button
            type="button"
            onClick={onToggle}
            aria-label={
              open
                ? t('devices.tree.collapseNode', { name: node.label })
                : t('devices.tree.expandNode', { name: node.label })
            }
            className="flex size-4 shrink-0 items-center justify-center text-faint hover:text-fg"
          >
            <ChevronDown className={cn('size-3 transition-transform', open ? '' : '-rotate-90')} />
          </button>
        ) : (
          <span className="size-4 shrink-0" />
        )}
        <button
          type="button"
          onClick={() => onSelect(node)}
          title={node.label}
          className="min-w-0 flex-1 truncate text-left"
        >
          {node.label}
        </button>
        <span className="shrink-0 text-xs text-faint tabular">{node.count}</span>
      </div>
      {hasChildren && open
        ? node.children?.map((child) => (
            <TreeRow
              key={child.key}
              node={child}
              depth={depth + 1}
              activeProtocol={activeProtocol}
              activeModel={activeModel}
              open={false}
              onToggle={() => undefined}
              onSelect={onSelect}
            />
          ))
        : null}
    </>
  )
}

export function DeviceTreePanel({
  devices,
  mode,
  onModeChange,
  activeProtocol,
  activeModel,
  totalCount,
  loading,
  onSelect,
}: {
  devices: DeviceDetail[]
  mode: TreeMode
  onModeChange: (mode: TreeMode) => void
  activeProtocol: DeviceType | ''
  activeModel: string
  totalCount: number
  loading: boolean
  onSelect: (protocol: DeviceType | '', model: string) => void
}) {
  const { t } = useI18n()
  const [query, setQuery] = useState('')
  const [searchOpen, setSearchOpen] = useState(false)
  const [closed, setClosed] = useState<Set<string>>(new Set())

  const tree = useMemo(() => buildDeviceTree(devices, mode), [devices, mode])
  const visibleTree = useMemo(() => filterTree(tree, query), [tree, query])
  const expandableKeys = tree.filter((node) => node.children?.length).map((node) => node.key)
  const allOpen = expandableKeys.every((key) => !closed.has(key))

  const toggleNode = (key: string) => {
    setClosed((current) => {
      const next = new Set(current)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  const toggleAll = () => {
    setClosed(allOpen ? new Set(expandableKeys) : new Set())
  }

  const handleSelectNode = (node: DeviceTreeNode) => {
    onSelect(node.protocol ?? '', node.model ?? '')
  }

  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-border p-3">
        <div className="mb-2 flex items-center justify-between">
          <span className="text-sm font-medium">{t('devices.tree.title')}</span>
          <div className="flex items-center gap-0.5">
            <button
              type="button"
              onClick={() => setSearchOpen((current) => !current)}
              aria-label={t('devices.tree.search')}
              title={t('devices.tree.search')}
              className="flex size-6 items-center justify-center rounded text-muted hover:bg-surface-2 hover:text-fg"
            >
              <Search className="size-3.5" />
            </button>
            <button
              type="button"
              onClick={toggleAll}
              aria-label={
                allOpen ? t('devices.tree.collapseAll') : t('devices.tree.expandAll')
              }
              title={allOpen ? t('devices.tree.collapseAll') : t('devices.tree.expandAll')}
              className="flex size-6 items-center justify-center rounded text-muted hover:bg-surface-2 hover:text-fg"
            >
              <ListTree className="size-3.5" />
            </button>
          </div>
        </div>
        <Select
          aria-label={t('devices.tree.groupBy')}
          value={mode}
          onChange={(event) => onModeChange(event.target.value as TreeMode)}
          className="h-7 text-xs"
        >
          <option value="protocol">{t('devices.tree.mode.protocol')}</option>
          <option value="model">{t('devices.tree.mode.model')}</option>
        </Select>
        {searchOpen ? (
          <Input
            autoFocus
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t('devices.tree.filter')}
            aria-label={t('devices.tree.filter')}
            className="mt-2 h-7 text-xs"
          />
        ) : null}
      </div>

      <div className="flex-1 overflow-y-auto p-2 text-sm">
        <div
          className={cn(
            'flex items-center gap-1 rounded-sm py-1 pr-2 pl-1',
            !activeProtocol && !activeModel
              ? 'bg-primary-soft font-medium text-primary'
              : 'hover:bg-surface-2',
          )}
        >
          <span className="size-4 shrink-0" />
          <button
            type="button"
            onClick={() => onSelect('', '')}
            className="min-w-0 flex-1 truncate text-left"
          >
            {t('devices.tree.all')}
          </button>
          <span className="shrink-0 text-xs text-faint tabular">{totalCount}</span>
        </div>

        {visibleTree.map((node) => (
          <TreeRow
            key={node.key}
            node={node}
            depth={1}
            activeProtocol={activeProtocol}
            activeModel={activeModel}
            open={query.trim() ? true : !closed.has(node.key)}
            onToggle={() => toggleNode(node.key)}
            onSelect={handleSelectNode}
          />
        ))}

        {loading ? (
          <p className="px-2 py-3 text-xs text-faint">{t('devices.tree.loading')}</p>
        ) : null}
        {!loading && visibleTree.length === 0 ? (
          <p className="px-2 py-3 text-xs text-faint">{t('devices.tree.noMatch')}</p>
        ) : null}
      </div>
    </div>
  )
}
