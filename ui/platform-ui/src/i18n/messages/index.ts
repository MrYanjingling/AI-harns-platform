import type { Messages } from '@harns/core'
import { audit } from './audit'
import { auth } from './auth'
import { dashboard } from './dashboard'
import { enums } from './enums'
import { identity } from './identity'
import { layout } from './layout'
import { thingModels } from './thingModels'
import { things } from './things'

/** 平台子应用字典;核心层字典由入口与 coreMessages 合并。 */
export const platformMessages: Messages = {
  'zh-CN': {
    ...auth['zh-CN'],
    ...audit['zh-CN'],
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
    ...dashboard['en-US'],
    ...enums['en-US'],
    ...identity['en-US'],
    ...layout['en-US'],
    ...thingModels['en-US'],
    ...things['en-US'],
  },
}

export type PlatformMessageKey = keyof (typeof platformMessages)['zh-CN']

// 编译期校验:英文必须覆盖中文的全部 key。
const enUS: Record<PlatformMessageKey, string> = platformMessages['en-US']
void enUS
