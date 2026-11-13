import { platformData } from './platform'
import type { UploadedFile } from './types'

/** 上传图片,返回可写入 things.image 的 MinIO key。 */
export function uploadImage(file: File): Promise<UploadedFile> {
  const formData = new FormData()
  formData.append('file', file)
  return platformData<UploadedFile>('/iot/files', { method: 'POST', formData })
}
