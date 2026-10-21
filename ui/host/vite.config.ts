import { fileURLToPath } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import type { Plugin } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { mockApiPlugin } from './mock'

const here = (path: string) => fileURLToPath(new URL(path, import.meta.url))

/**
 * 子应用以源码形式被主应用直接编译,它们内部仍使用 `@/` 指向各自 src。
 * 这里按 importer 所在包解析,避免主应用的 `@` 别名串到子应用。
 */
function subAppSourceAlias(): Plugin {
  return {
    name: 'harns-subapp-source-alias',
    enforce: 'pre' as const,
    async resolveId(source: string, importer: string | undefined) {
      if (!source.startsWith('@/') || !importer) return null
      const relative = source.slice(2)
      const root = importer.includes('/platform-ui/')
        ? here('../platform-ui/src/')
        : importer.includes('/gateway-ui/')
          ? here('../gateway-ui/src/')
          : null
      if (!root) return null
      // 交给 Vite 的默认解析补全扩展名与 index 文件,避免手写探查逻辑。
      return this.resolve(root + relative, importer, { skipSelf: true })
    },
  }
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const useMock = env.VITE_USE_MOCK === 'true'
  const gatewayTarget = env.VITE_GATEWAY_TARGET || 'http://127.0.0.1:32200'
  const platformTarget = env.VITE_PLATFORM_TARGET || 'http://127.0.0.1:32300'

  return {
    plugins: [subAppSourceAlias(), react(), tailwindcss(), ...(useMock ? [mockApiPlugin()] : [])],
    resolve: {
      alias: {
        '@harns/core': here('../core/src/index.ts'),
      },
      // 子应用与主应用共享同一份 React / Router / Query 实例。
      dedupe: ['react', 'react-dom', 'react-router-dom', '@tanstack/react-query'],
    },
    optimizeDeps: {
      exclude: ['@harns/core', '@harns/platform-ui', '@harns/gateway-ui'],
    },
    server: {
      port: 5173,
      // 两套服务共用 /api/v1 前缀,按路径分流;网关路径必须写在前面。
      proxy: useMock
        ? undefined
        : {
            '/api/v1/devices': { target: gatewayTarget, changeOrigin: true },
            '/api/v1/gatewayMeta': { target: gatewayTarget, changeOrigin: true },
            '/api/v1/gatewayCpu': { target: gatewayTarget, changeOrigin: true },
            '/api/v1/gatewayMem': { target: gatewayTarget, changeOrigin: true },
            '/api/v1/gatewayDisk': { target: gatewayTarget, changeOrigin: true },
            '/api/v1/auth': { target: platformTarget, changeOrigin: true },
            '/api/v1/iot': { target: platformTarget, changeOrigin: true },
          },
    },
    build: {
      outDir: 'dist',
    },
  }
})
