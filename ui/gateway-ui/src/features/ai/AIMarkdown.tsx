import type { ReactNode } from 'react'
import ReactMarkdown, { type Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'

const DANGER_PREFIX =
  /^(异常|错误|失败|告警|警告|超时|未连接|abnormal|error|failed|warning|timeout|disconnected)/i

const DANGER_SEPARATOR =
  /(?<=\*\*)[ \t]*[，,；;]?[ \t]*(?=\*\*\s*(?:异常|错误|失败|告警|警告|超时|未连接|abnormal|error|failed|warning|timeout|disconnected))/gi

function normalizeDangerLines(content: string): string {
  return content.replace(DANGER_SEPARATOR, '\n')
}

function nodeText(node: ReactNode): string {
  if (node === null || node === undefined || typeof node === 'boolean') return ''
  if (typeof node === 'string' || typeof node === 'number') return String(node)
  if (Array.isArray(node)) return node.map(nodeText).join('')
  if (typeof node === 'object' && 'props' in node) {
    return nodeText((node as { props?: { children?: ReactNode } }).props?.children)
  }
  return ''
}

const components: Components = {
  p: ({ children }) => (
    <p className="mb-2 whitespace-pre-wrap text-sm leading-6 last:mb-0">{children}</p>
  ),
  ul: ({ children }) => (
    <ul className="my-2 list-disc space-y-1 pl-5 text-sm leading-6">{children}</ul>
  ),
  ol: ({ children }) => (
    <ol className="my-2 list-decimal space-y-1 pl-5 text-sm leading-6">{children}</ol>
  ),
  li: ({ children }) => <li className="pl-0.5">{children}</li>,
  strong: ({ children }) => {
    const danger = DANGER_PREFIX.test(nodeText(children).trim())
    return (
      <strong className={danger ? 'font-semibold text-danger' : 'font-semibold text-fg'}>
        {children}
      </strong>
    )
  },
  code: ({ children }) => (
    <code className="rounded bg-surface-3 px-1 py-0.5 font-mono text-[13px] text-fg">
      {children}
    </code>
  ),
  pre: ({ children }) => (
    <pre className="my-2 overflow-x-auto rounded border border-border bg-surface p-3 font-mono text-[13px] leading-6 [&>code]:bg-transparent [&>code]:p-0">
      {children}
    </pre>
  ),
  table: ({ children }) => (
    <div className="my-2 overflow-x-auto">
      <table className="w-full border-collapse text-[13px]">{children}</table>
    </div>
  ),
  th: ({ children }) => (
    <th className="border border-border bg-surface-2 px-2 py-1 text-left font-medium text-fg">
      {children}
    </th>
  ),
  td: ({ children }) => <td className="border border-border px-2 py-1">{children}</td>,
  a: ({ href, children }) => (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      className="text-primary underline underline-offset-2"
    >
      {children}
    </a>
  ),
  blockquote: ({ children }) => (
    <blockquote className="my-2 border-l-2 border-primary/40 pl-3 text-muted">
      {children}
    </blockquote>
  ),
  h1: ({ children }) => <h1 className="mb-2 text-base font-semibold text-fg">{children}</h1>,
  h2: ({ children }) => <h2 className="mb-2 text-sm font-semibold text-fg">{children}</h2>,
  h3: ({ children }) => <h3 className="mb-1.5 text-sm font-semibold text-fg">{children}</h3>,
  hr: () => <hr className="my-3 border-border" />,
}

export function AIMarkdown({ content }: { content: string }) {
  return (
    <div className="min-w-0 break-words">
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {normalizeDangerLines(content)}
      </ReactMarkdown>
    </div>
  )
}
