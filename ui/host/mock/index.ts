import type { IncomingMessage, ServerResponse } from 'node:http'
import type { Plugin } from 'vite'
import { handlePlatform } from '../../platform-ui/mock/platform'
import { handleGateway } from '../../gateway-ui/mock/gateway'
import { send } from '../../gateway-ui/mock/utils'

/** 主应用开发期 mock:同一 /api/v1 前缀下分派到平台与网关两套内存 API。 */

const PREFIX = '/api/v1'
const LATENCY_MS = 120

function sendError(res: ServerResponse, status: number, message: string) {
  send(res, status, { code: 10001, message, errors: [{ code: 10001, message }] })
}

async function handle(req: IncomingMessage, res: ServerResponse): Promise<boolean> {
  const method = (req.method ?? 'GET').toUpperCase()
  const url = new URL(req.url ?? '/', 'http://mock.local')

  if (!url.pathname.startsWith(PREFIX)) return false

  const pathname = url.pathname.slice(PREFIX.length) || '/'

  if (
    pathname.startsWith('/auth/') ||
    pathname === '/iot' ||
    pathname.startsWith('/iot/')
  ) {
    if (await handlePlatform(req, res, method, pathname, url.searchParams)) return true
  } else if (await handleGateway(req, res, method, pathname, url.searchParams)) {
    return true
  }

  const message = `mock 未实现: ${method} ${pathname}`
  send(res, 404, { code: 10004, message, errors: [{ code: 10004, message }] })
  return true
}

export function mockApiPlugin(): Plugin {
  return {
    name: 'harns-host-mock-api',
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
            sendError(res, 400, error instanceof Error ? error.message : 'mock error')
          }
        })()
      })
    },
  }
}
