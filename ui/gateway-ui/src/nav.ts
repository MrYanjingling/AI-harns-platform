import type { NavSection } from '@harns/core'

/** 网关子应用的侧边栏结构;五个菜单对应 pkg/gateway 的能力。 */
export function createGatewayNav(base = '/gateway'): NavSection[] {
  return [
    {
      key: 'gateway',
      labelKey: 'gateway.nav.section',
      entry: `${base}/collection`,
      prefixes: [base],
      groups: [
        {
          items: [
            {
              to: `${base}/subscriptions`,
              labelKey: 'gateway.nav.subscriptions',
              hintKey: 'gateway.nav.subscriptionsHint',
            },
            {
              to: `${base}/collection`,
              labelKey: 'gateway.nav.collection',
              hintKey: 'gateway.nav.collectionHint',
            },
            {
              to: `${base}/plugins`,
              labelKey: 'gateway.nav.plugins',
              hintKey: 'gateway.nav.pluginsHint',
            },
            {
              to: `${base}/logs`,
              labelKey: 'gateway.nav.logs',
              hintKey: 'gateway.nav.logsHint',
            },
            {
              to: `${base}/performance`,
              labelKey: 'gateway.nav.performance',
              hintKey: 'gateway.nav.performanceHint',
            },
          ],
        },
      ],
    },
  ]
}
