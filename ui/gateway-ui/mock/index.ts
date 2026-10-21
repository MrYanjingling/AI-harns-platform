import type { IncomingMessage, ServerResponse } from 'node:http'
import type { Plugin } from 'vite'
import { handleGateway } from './gateway'
import { send } from './utils'

/**
 * 网关独立开发期的内存 mock。
 * 响应形状与 pkg/apis/response.Result 一致:{code,message,data}。
 */

const PREFIX = '/api/v1'
const LATENCY_MS = 120

async function handle(req: IncomingMessage, res: ServerResponse): Promise<boolean> {
  const method = (req.method ?? 'GET').toUpperCase()
  const url = new URL(req.url ?? '/', 'http://mock.local')

  if (!url.pathname.startsWith(PREFIX)) return false

  const pathname = url.pathname.slice(PREFIX.length) || '/'
  const handled = await handleGateway(req, res, method, pathname, url.searchParams)
  if (handled) return true

  send(res, 404, {
    code: 10004,
    message: `mock 未实现: ${method} ${pathname}`,
    errors: [{ code: 10004, message: `mock 未实现: ${method} ${pathname}` }],
  })
  return true
}

export function mockApiPlugin(): Plugin {
  return {
    name: 'harns-gateway-mock-api',
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
            send(res, 400, {
              code: 10001,
              message,
              errors: [{ code: 10001, message }],
            })
          }
        })()
      })
    },
  }
}
