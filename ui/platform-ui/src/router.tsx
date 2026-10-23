import { lazy, Suspense } from 'react'
import { createBrowserRouter } from 'react-router-dom'
import { AppShell } from '@/components/layout/AppShell'
import { PageLoading } from '@/components/ui/Spinner'
import { DashboardPage } from '@/features/dashboard/DashboardPage'
import { DevicesPage } from '@/features/devices/DevicesPage'
import { DeviceCreatePage } from '@/features/devices/DeviceCreatePage'
import { DeviceDetailPage } from '@/features/devices/DeviceDetailPage'
import { DeviceEditPage } from '@/features/devices/DeviceEditPage'
import { MetricsPage } from '@/features/metrics/MetricsPage'
import { KpiPage } from '@/features/kpi/KpiPage'
import { TagsPage } from '@/features/tags/TagsPage'
import { TimeSeriesPage } from '@/features/timeseries/TimeSeriesPage'
import { SettingsPage } from '@/features/settings/SettingsPage'
import { NotFoundPage } from '@/features/NotFoundPage'

// 大屏引入 Three.js,按需加载,避免拖慢管理后台首屏。
const FactoryScreenPage = lazy(() =>
  import('@/features/screen/FactoryScreenPage').then((module) => ({
    default: module.FactoryScreenPage,
  })),
)

export const router = createBrowserRouter([
  {
    path: '/screen',
    element: (
      <Suspense
        fallback={
          <div className="dark flex min-h-dvh items-center justify-center bg-bg">
            <PageLoading label="正在加载工厂大屏…" />
          </div>
        }
      >
        <FactoryScreenPage />
      </Suspense>
    ),
  },
  {
    path: '/',
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: 'devices', element: <DevicesPage /> },
      { path: 'devices/new', element: <DeviceCreatePage /> },
      { path: 'devices/:id', element: <DeviceDetailPage /> },
      { path: 'devices/:id/edit', element: <DeviceEditPage /> },
      { path: 'data/metrics', element: <MetricsPage /> },
      { path: 'data/kpis', element: <KpiPage /> },
      { path: 'data/tags', element: <TagsPage /> },
      { path: 'data/timeseries', element: <TimeSeriesPage /> },
      { path: 'settings', element: <SettingsPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])
