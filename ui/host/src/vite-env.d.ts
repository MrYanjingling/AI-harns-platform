/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_USE_MOCK?: string
  readonly VITE_PLATFORM_TARGET?: string
  readonly VITE_PLATFORM_BASE?: string
  readonly VITE_MINIO_PUBLIC_BASE?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
