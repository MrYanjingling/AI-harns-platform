import { Outlet } from 'react-router-dom'

/** 管理后台常规页面的留白容器;需要满幅布局的页面(如设备管理)直接挂在 AppShell 下。 */
export function PagePad() {
  return (
    <div className="p-3 sm:p-4">
      <Outlet />
    </div>
  )
}
