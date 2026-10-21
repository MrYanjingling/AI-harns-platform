/**
 * 平台子应用对外出口。
 * 主应用只依赖这里的路由、导航、字典与认证能力,不直接引用 features。
 */
export { platformRoutes } from './routes'
export { createPlatformNav } from './components/layout/nav-config'
export { PlatformAccountMenu } from './components/layout/PlatformAccountMenu'
export { platformMessages } from './i18n/messages'
export { AuthProvider, useAuth } from './providers/AuthProvider'
export { LoginPage } from './features/auth/LoginPage'
export { RequireAuth } from './features/auth/RequireAuth'
