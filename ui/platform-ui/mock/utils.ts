import type { IncomingMessage, ServerResponse } from 'node:http'

export function send(
  res: ServerResponse,
  status: number,
  body?: unknown,
  headers?: Record<string, string>,
) {
  if (headers) {
    for (const [key, value] of Object.entries(headers)) res.setHeader(key, value)
  }
  if (body === undefined) {
    res.statusCode = status
    res.end()
    return
  }
  res.statusCode = status
  res.setHeader('Content-Type', 'application/json; charset=utf-8')
  res.end(JSON.stringify(body))
}

export function readJson(req: IncomingMessage): Promise<unknown> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = []
    req.on('data', (chunk: Buffer) => chunks.push(chunk))
    req.on('end', () => {
      const raw = Buffer.concat(chunks).toString('utf8')
      if (!raw) return resolve(undefined)
      try {
        resolve(JSON.parse(raw))
      } catch {
        reject(new Error('malformed json'))
      }
    })
    req.on('error', reject)
  })
}

export function readRaw(req: IncomingMessage): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = []
    req.on('data', (chunk: Buffer) => chunks.push(chunk))
    req.on('end', () => resolve(Buffer.concat(chunks)))
    req.on('error', reject)
  })
}
