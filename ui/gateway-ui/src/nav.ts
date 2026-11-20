import { createElement } from 'react'
import {
  Bot,
  Cpu,
  FlaskConical,
  Gauge,
  Info,
  LayoutDashboard,
  MessageCircle,
  ListTree,
  PlugZap,
  Radio,
  ScrollText,
  Send,
  Settings,
  Settings2,
  Sparkles,
  Terminal,
  Users,
  Boxes,
} from 'lucide-react'
import type { NavSection } from '@harns/core'

function icon(component: typeof LayoutDashboard) {
  return createElement(component, { className: 'size-4' })
}

/** 网关侧边栏:一级菜单及其二级菜单。 */
export function createGatewayNav(base = '', isAdmin = false): NavSection[] {
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
          ],
        },
        {
          labelKey: 'gateway.nav.collection',
          icon: icon(Radio),
          items: [
            {
              to: `${base}/collections`,
              labelKey: 'gateway.nav.collection.instance',
              hintKey: 'gateway.nav.collectionHint',
              icon: icon(ListTree),
            },
            {
              to: `${base}/monitoring/points`,
              labelKey: 'gateway.nav.monitor.points',
              hintKey: 'gateway.nav.monitor.pointsHint',
              icon: icon(Gauge),
            },
          ],
        },
        {
          labelKey: 'gateway.nav.push',
          icon: icon(Send),
          items: [
            {
              to: `${base}/push`,
              labelKey: 'gateway.nav.push.instance',
              hintKey: 'gateway.nav.pushHint',
              icon: icon(PlugZap),
              end: true,
            },
            {
              to: `${base}/push/groups`,
              labelKey: 'gateway.nav.push.groups',
              hintKey: 'gateway.nav.push.groupsHint',
              icon: icon(Boxes),
            },
          ],
        },
        {
          labelKey: 'gateway.nav.drivers',
          icon: icon(Cpu),
          items: [
            {
              to: `${base}/drivers`,
              labelKey: 'gateway.nav.drivers.settings',
              hintKey: 'gateway.nav.driversHint',
              icon: icon(Settings2),
            },
          ],
        },
        {
          labelKey: 'gateway.nav.logs',
          icon: icon(ScrollText),
          items: [
            {
              to: `${base}/logs`,
              labelKey: 'gateway.nav.logs.realtime',
              hintKey: 'gateway.nav.logsHint',
              icon: icon(Terminal),
            },
          ],
        },
        {
          labelKey: 'gateway.nav.simulator',
          icon: icon(FlaskConical),
          items: [
            {
              to: `${base}/simulator/modbus`,
              labelKey: 'gateway.nav.simulator.modbus',
              hintKey: 'gateway.nav.simulatorHint',
              icon: icon(FlaskConical),
            },
          ],
        },
        {
          labelKey: 'gateway.nav.ai',
          icon: icon(Sparkles),
          items: [
            {
              to: `${base}/ai/assistant`,
              labelKey: 'gateway.nav.ai.assistant',
              hintKey: 'gateway.nav.ai.assistantHint',
              icon: icon(MessageCircle),
            },
            ...(isAdmin
              ? [
                  {
                    to: `${base}/ai/models`,
                    labelKey: 'gateway.nav.ai.models',
                    hintKey: 'gateway.nav.ai.modelsHint',
                    icon: icon(Bot),
                  },
                ]
              : []),
          ],
        },
        {
          labelKey: 'gateway.nav.system',
          icon: icon(Settings),
          items: [
            {
              to: `${base}/system/users`,
              labelKey: 'gateway.nav.system.users',
              hintKey: 'gateway.nav.system.usersHint',
              icon: icon(Users),
            },
            {
              to: `${base}/system/about`,
              labelKey: 'gateway.nav.system.about',
              hintKey: 'gateway.nav.system.aboutHint',
              icon: icon(Info),
            },
          ],
        },
      ],
    },
  ]
}
