import { auth } from './auth'
import { common } from './common'
import { components } from './components'
import { dashboard } from './dashboard'
import { devices } from './devices'
import { enums } from './enums'
import { identity } from './identity'
import { layout } from './layout'
import { metrics } from './metrics'
import { screen } from './screen'
import { settings } from './settings'
import { thingModels } from './thingModels'
import { things } from './things'
import { timeseries } from './timeseries'

/** 所有域的消息按 key 前缀合并;新增域时在此登记。 */
export const messages = {
  'zh-CN': {
    ...auth['zh-CN'],
    ...common['zh-CN'],
    ...components['zh-CN'],
    ...dashboard['zh-CN'],
    ...devices['zh-CN'],
    ...enums['zh-CN'],
    ...identity['zh-CN'],
    ...layout['zh-CN'],
    ...metrics['zh-CN'],
    ...screen['zh-CN'],
    ...settings['zh-CN'],
    ...thingModels['zh-CN'],
    ...things['zh-CN'],
    ...timeseries['zh-CN'],
  },
  'en-US': {
    ...auth['en-US'],
    ...common['en-US'],
    ...components['en-US'],
    ...dashboard['en-US'],
    ...devices['en-US'],
    ...enums['en-US'],
    ...identity['en-US'],
    ...layout['en-US'],
    ...metrics['en-US'],
    ...screen['en-US'],
    ...settings['en-US'],
    ...thingModels['en-US'],
    ...things['en-US'],
    ...timeseries['en-US'],
  },
} as const

export type MessageKey = keyof (typeof messages)['zh-CN']

// 编译期校验:英文必须覆盖中文的全部 key。
const enUS: Record<MessageKey, string> = messages['en-US']
void enUS
