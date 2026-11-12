import type { IncomingMessage, ServerResponse } from 'node:http'
import type { Plugin } from 'vite'
import { handlePlatform } from '../../platform-ui/mock/platform'
import { send } from '../../platform-ui/mock/utils'

/** 主应用开发期 mock:平台内存 API。 */

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
    (pathname.startsWith('/auth/') || pathname === '/iot' || pathname.startsWith('/iot/')) &&
    (await handlePlatform(req, res, method, pathname, url.searchParams))
  ) {
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
