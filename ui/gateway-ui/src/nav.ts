import { createElement } from 'react'
import {
  Activity,
  Cpu,
  LayoutDashboard,
  Radio,
  ScrollText,
  Send,
} from 'lucide-react'
import type { NavSection } from '@harns/core'

function icon(component: typeof LayoutDashboard) {
  return createElement(component, { className: 'size-4' })
}

/** 网关应用的侧边栏;六个菜单覆盖采集、推送、驱动、日志与资源。 */
export function createGatewayNav(base = ''): NavSection[] {
  return [
    {
      key: 'gateway',
      labelKey: 'gateway.nav.section',
      icon: icon(LayoutDashboard),
      entry: `${base}/overview`,
      prefixes: [base || '/'],
      groups: [
        {
          items: [
            {
              to: `${base}/overview`,
              labelKey: 'gateway.nav.overview',
              hintKey: 'gateway.nav.overviewHint',
              icon: icon(LayoutDashboard),
              end: true,
            },
            {
              to: `${base}/collections`,
              labelKey: 'gateway.nav.collection',
              hintKey: 'gateway.nav.collectionHint',
              icon: icon(Radio),
            },
            {
              to: `${base}/push`,
              labelKey: 'gateway.nav.push',
              hintKey: 'gateway.nav.pushHint',
              icon: icon(Send),
            },
            {
              to: `${base}/drivers`,
              labelKey: 'gateway.nav.drivers',
              hintKey: 'gateway.nav.driversHint',
              icon: icon(Cpu),
            },
            {
              to: `${base}/logs`,
              labelKey: 'gateway.nav.logs',
              hintKey: 'gateway.nav.logsHint',
              icon: icon(ScrollText),
            },
            {
              to: `${base}/performance`,
              labelKey: 'gateway.nav.performance',
              hintKey: 'gateway.nav.performanceHint',
              icon: icon(Activity),
            },
          ],
        },
      ],
    },
  ]
}
