import { Outlet, useLocation } from 'react-router-dom'

export function GatewayPagePad() {
  const { pathname } = useLocation()
  return (
    <div className="min-h-full bg-surface-3 p-4 sm:p-6">
      <div key={pathname} className="animate-fade-up">
        <Outlet />
      </div>
    </div>
  )
}
