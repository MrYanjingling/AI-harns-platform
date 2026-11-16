import { Outlet, useLocation } from 'react-router-dom'

export function GatewayPagePad() {
  const { pathname } = useLocation()
  return (
    <div className="flex min-h-full flex-col bg-surface-3 px-4 pt-4 pb-0 sm:px-6 sm:pt-6 sm:pb-0">
      <div key={pathname} className="flex min-h-0 flex-1 flex-col animate-fade-up">
        <Outlet />
      </div>
    </div>
  )
}
