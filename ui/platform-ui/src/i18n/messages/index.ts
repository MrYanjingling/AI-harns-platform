import { audit } from './audit'
import { auth } from './auth'
import { common } from './common'
import { components } from './components'
import { dashboard } from './dashboard'
import { enums } from './enums'
import { identity } from './identity'
import { layout } from './layout'
import { thingModels } from './thingModels'
import { things } from './things'

/** 所有域的消息按 key 前缀合并;新增域时在此登记。 */
export const messages = {
  'zh-CN': {
    ...auth['zh-CN'],
    ...audit['zh-CN'],
    ...common['zh-CN'],
    ...components['zh-CN'],
    ...dashboard['zh-CN'],
    ...enums['zh-CN'],
    ...identity['zh-CN'],
    ...layout['zh-CN'],
    ...thingModels['zh-CN'],
    ...things['zh-CN'],
  },
  'en-US': {
    ...auth['en-US'],
    ...audit['en-US'],
    ...common['en-US'],
    ...components['en-US'],
    ...dashboard['en-US'],
    ...enums['en-US'],
    ...identity['en-US'],
    ...layout['en-US'],
    ...thingModels['en-US'],
    ...things['en-US'],
  },
} as const

export type MessageKey = keyof (typeof messages)['zh-CN']

// 编译期校验:英文必须覆盖中文的全部 key。
const enUS: Record<MessageKey, string> = messages['en-US']
void enUS
