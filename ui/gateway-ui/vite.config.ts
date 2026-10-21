import { fileURLToPath } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { mockApiPlugin } from './mock'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const useMock = env.VITE_USE_MOCK === 'true'
  const gatewayTarget = env.VITE_GATEWAY_TARGET || 'http://127.0.0.1:32200'

  return {
    plugins: [react(), tailwindcss(), ...(useMock ? [mockApiPlugin()] : [])],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
        '@harns/core': fileURLToPath(new URL('../core/src/index.ts', import.meta.url)),
      },
    },
    optimizeDeps: {
      exclude: ['@harns/core'],
    },
    server: {
      port: 5175,
      // 网关独立开发时,所有 /api/v1 请求都指向 gateway。
      proxy: useMock
        ? undefined
        : {
            '/api/v1': { target: gatewayTarget, changeOrigin: true },
          },
    },
    build: {
      outDir: 'dist',
    },
  }
})
