import { Outlet, useLocation } from 'react-router-dom'

/** 管理后台常规页面的留白容器;需要满幅布局的页面(如设备管理)直接挂在 AppShell 下。 */
export function PagePad() {
  const { pathname } = useLocation()
  return (
    <div className="p-2.5 sm:p-3">
      <div key={pathname} className="animate-page-in">
        <Outlet />
      </div>
    </div>
  )
}
