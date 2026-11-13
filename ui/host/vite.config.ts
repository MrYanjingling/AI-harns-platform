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
      if (!importer.includes('/platform-ui/')) return null
      // 交给 Vite 的默认解析补全扩展名与 index 文件,避免手写探查逻辑。
      return this.resolve(here('../platform-ui/src/') + source.slice(2), importer, { skipSelf: true })
    },
  }
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const useMock = env.VITE_USE_MOCK === 'true'
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
      exclude: ['@harns/core', '@harns/platform-ui'],
    },
    server: {
      port: 5173,
      // 平台服务:认证 + /iot 下的用户/角色/组织/审计/物模型/设备。
      proxy: useMock
        ? undefined
        : {
            '/api/v1/auth': { target: platformTarget, changeOrigin: true },
            '/api/v1/iot': { target: platformTarget, changeOrigin: true },
          },
    },
    build: {
      outDir: 'dist',
    },
  }
})
