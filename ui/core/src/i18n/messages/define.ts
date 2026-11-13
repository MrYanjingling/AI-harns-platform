/**
 * 并排声明中英文消息,强制两种语言的 key 完全一致。
 * 返回值的 zh-CN 分支保留字面量类型,供上层推导 MessageKey。
 */
export function defineMessages<const Zh extends Record<string, string>>(messages: {
  'zh-CN': Zh
  'en-US': Record<keyof Zh, string>
}): { 'zh-CN': Zh; 'en-US': Record<keyof Zh, string> } {
  return messages
}
