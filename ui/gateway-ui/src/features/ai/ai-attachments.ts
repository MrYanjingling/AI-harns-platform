export const MAX_FILES = 5
export const MAX_FILE_BYTES = 10 << 20
export const MAX_TOTAL_BYTES = 50 << 20
export const FILE_ACCEPT =
  '.txt,.log,.json,.yaml,.yml,.csv,.md,.xml,.doc,.docx,.xls,.xlsx,.pdf,text/*,application/msword,application/vnd.ms-excel,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'

const ACCEPTED_FILE = /\.(txt|log|json|ya?ml|csv|md|xml|doc|docx|xls|xlsx|pdf)$/i

export function isAcceptedFile(file: File): boolean {
  return ACCEPTED_FILE.test(file.name) || file.type.startsWith('text/')
}
