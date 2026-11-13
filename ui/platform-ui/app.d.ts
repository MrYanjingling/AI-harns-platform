import type { ComponentType, ReactNode } from 'react'
import type { RouteObject } from 'react-router-dom'
import type { Messages, NavSection } from '@harns/core'

/** 平台子应用对主应用暴露的稳定接口。 */
export declare const platformRoutes: RouteObject[]
export declare function createPlatformNav(base?: string): NavSection[]
export declare const platformMessages: Messages
export declare const PlatformAccountMenu: ComponentType
export declare const LoginPage: ComponentType
export declare const RequireAuth: ComponentType<{ children: ReactNode }>
export declare const AuthProvider: ComponentType<{ children: ReactNode }>
export declare function useAuth(): unknown
