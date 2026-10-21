import { common } from './common'
import { components } from './components'
import { layout } from './layout'

export interface Messages {
  'zh-CN': Record<string, string>
  'en-US': Record<string, string>
}

/** 核心层字典:通用文案、无业务语义组件、应用外壳。 */
export const coreMessages: Messages = {
  'zh-CN': {
    ...common['zh-CN'],
    ...components['zh-CN'],
    ...layout['zh-CN'],
  },
  'en-US': {
    ...common['en-US'],
    ...components['en-US'],
    ...layout['en-US'],
  },
}

export { defineMessages } from './define'
