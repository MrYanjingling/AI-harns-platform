/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_GATEWAY_TARGET?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
