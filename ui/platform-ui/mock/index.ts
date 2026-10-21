import type { IncomingMessage, ServerResponse } from 'node:http'
import type { Plugin } from 'vite'
import { gatewayMeta } from './fixtures'
import { handlePlatform } from './platform'
import { send } from './utils'

/**
 * 仅开发期生效的 mock API。
 * 与真实后端保持同一套 /api/v1 路径与 JSON 形状,
 * 让前端可以在 Go 服务未启动时完整演示。
 */

const PREFIX = '/api/v1'
const LATENCY_MS = 120

function sendErrors(res: ServerResponse, status: number, messages: string[], code = 10001) {
  send(res, status, { errors: messages.map((message) => ({ code, message })) })
}

async function handle(req: IncomingMessage, res: ServerResponse): Promise<boolean> {
  const method = (req.method ?? 'GET').toUpperCase()
  const url = new URL(req.url ?? '/', 'http://mock.local')
  const path = url.pathname

  if (!path.startsWith(PREFIX)) return false

  const pathname = path.slice(PREFIX.length) || '/'
  const query = url.searchParams

  // 平台服务(auth + /iot/**)由独立的有状态 mock 处理。
  if (pathname.startsWith('/auth/') || pathname === '/iot' || pathname.startsWith('/iot/')) {
    return handlePlatform(req, res, method, pathname, query)
  }

  /* ---------------- 网关 ---------------- */

  if (method === 'GET' && pathname === '/gatewayMeta') {
    send(res, 200, { code: 200, message: 'success', data: gatewayMeta }, { ETag: gatewayMeta.eTag })
    return true
  }

  if (method === 'GET' && pathname === '/gatewayCpu') {
    const t = new Date().getSeconds()
    const base = 18 + Math.sin(t / 9) * 6
    send(res, 200, {
      code: 200,
      message: 'success',
      data: {
        cpus: {
          '1min': `${base.toFixed(0)}%`,
          '5min': `${(base - 2).toFixed(0)}%`,
          '15min': `${(base - 4).toFixed(0)}%`,
        },
      },
    })
    return true
  }

  if (method === 'GET' && pathname === '/gatewayMem') {
    const usedPercent = 54 + Math.sin(new Date().getSeconds() / 12) * 3
    send(res, 200, {
      code: 200,
      message: 'success',
      data: {
        mem: {
          Total: '3906MB',
          Used: `${Math.round((3906 * usedPercent) / 100)}MB`,
          UsedPercent: `${usedPercent.toFixed(0)}%`,
        },
      },
    })
    return true
  }

  if (method === 'GET' && pathname === '/gatewayDisk') {
    send(res, 200, {
      code: 200,
      message: 'success',
      data: {
        disk: {
          '/': { Total: '29072MB', Used: '11248MB', UsedPercent: '39%' },
          '/data': { Total: '119234MB', Used: '74821MB', UsedPercent: '63%' },
        },
      },
    })
    return true
  }

  // 未实现的 /api/v1 路径显式返回 404,避免误以为连到了真实后端。
  send(res, 404, { errors: [{ code: 10004, message: `mock 未实现: ${method} ${pathname}` }] })
  return true
}

export function mockApiPlugin(): Plugin {
  return {
    name: 'harns-mock-api',
    apply: 'serve',
    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        const path = (req.url ?? '').split('?')[0]
        if (!path.startsWith(PREFIX)) {
          next()
          return
        }

        void (async () => {
          await new Promise((resolve) => setTimeout(resolve, LATENCY_MS))
          try {
            const handled = await handle(req, res)
            if (!handled) next()
          } catch (error) {
            const message = error instanceof Error ? error.message : 'mock error'
            sendErrors(res, 400, [message])
          }
        })()
      })
    },
  }
}
