import { ChevronDown, Columns3, Download, RefreshCw, RotateCcw, Search } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { DataTable } from '@/components/shared/DataTable'
import { FilterBar } from '@/components/shared/FilterBar'
import { Toolbar } from '@/components/shared/Toolbar'
import { useI18n } from '@/providers/I18nProvider'

const controlClass =
  'h-9 rounded-lg border border-border bg-surface px-3 text-sm text-fg outline-none placeholder:text-faint focus:ring-2 focus:ring-primary/25'

export function CollectionsPage() {
  const { t } = useI18n()
  return (
    <>
      <FilterBar>
        <select className={controlClass} defaultValue="fuzzy">
          <option value="fuzzy">{t('filter.fuzzy')}</option>
        </select>
        <input className={`${controlClass} w-48`} placeholder={t('filter.collectionName')} />
        <select className={controlClass} defaultValue="all">
          <option value="all">{t('filter.driver')}</option>
        </select>
        <input className={`${controlClass} w-40`} placeholder={t('filter.status')} />
        <div className="ml-auto flex items-center gap-2">
          <Button size="icon" aria-label={t('action.search')}>
            <Search className="size-4" />
          </Button>
          <Button size="icon" aria-label={t('action.reset')}>
            <RotateCcw className="size-4" />
          </Button>
          <Button size="icon" aria-label={t('action.expand')}>
            <ChevronDown className="size-4" />
          </Button>
        </div>
      </FilterBar>
      <Toolbar>
        <Button size="sm">
          <Columns3 className="size-4" />
          {t('action.columns')}
        </Button>
        <Button size="sm">
          <Download className="size-4" />
          {t('action.export')}
        </Button>
        <Button size="sm">
          <RefreshCw className="size-4" />
          {t('action.refresh')}
        </Button>
      </Toolbar>
      <DataTable
        columns={[
          { key: 'name', title: t('table.name') },
          { key: 'driver', title: t('filter.driver') },
          { key: 'status', title: t('table.status') },
          { key: 'cycle', title: t('table.cycle') },
          { key: 'lastCollect', title: t('table.lastCollect') },
        ]}
        emptyText={t('table.empty')}
      />
    </>
  )
}
