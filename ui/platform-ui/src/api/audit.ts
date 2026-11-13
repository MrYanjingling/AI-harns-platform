import { platformPage } from './platform'
import type { AuditLogInfo, AuditQuery, PlatformPage } from './types'

export function listAuditLogs(query: AuditQuery): Promise<PlatformPage<AuditLogInfo>> {
  return platformPage<AuditLogInfo>('/iot/auditLogs', {
    page: query.page,
    pageSize: query.pageSize,
    app: query.app,
    userId: query.userId,
    path: query.path,
    method: query.method,
    resultCode: query.resultCode,
    start: query.start,
    end: query.end,
  })
}
