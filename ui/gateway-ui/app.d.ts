import type { RouteObject } from 'react-router-dom'
import type { Messages, NavSection } from '@harns/core'

/** 网关子应用对主应用暴露的稳定接口。 */
export declare const gatewayRoutes: RouteObject[]
export declare function createGatewayNav(base?: string): NavSection[]
export declare const gatewayMessages: Messages
