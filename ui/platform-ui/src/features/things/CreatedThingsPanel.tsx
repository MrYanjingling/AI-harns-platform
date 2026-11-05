import { useNavigate } from 'react-router-dom'
import { CheckCircle2, Plus } from 'lucide-react'
import type { ThingDetail } from '@/api/types'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { CopyButton } from '@/components/ui/CopyButton'
import { TBody, TD, TH, THead, TR, Table } from '@/components/ui/Table'

function SecretCell({ value }: { value: string }) {
  if (!value) return <span className="text-faint">—</span>
  return (
    <span className="inline-flex max-w-56 items-center gap-1">
      <span className="truncate font-mono text-[11px]" title={value}>
        {value}
      </span>
      <CopyButton value={value} />
    </span>
  )
}

/** 批量创建成功后的凭证结果页:凭据只在创建响应中一次性返回。 */
export function CreatedThingsPanel({
  things,
  onContinue,
}: {
  things: ThingDetail[]
  onContinue: () => void
}) {
  const navigate = useNavigate()

  return (
    <div className="space-y-4">
      <Card className="flex flex-wrap items-center justify-between gap-3 border-emerald-600/25 bg-emerald-500/5 p-5">
        <div className="flex items-start gap-3">
          <CheckCircle2 className="mt-0.5 size-5 text-success" />
          <div>
            <p className="text-sm font-semibold">已创建 {things.length} 台设备</p>
            <p className="mt-0.5 text-xs text-muted">
              设备密钥与 Token 由后端生成,之后可在设备详情查看与复制。
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={onContinue} icon={<Plus className="size-4" />}>
            继续创建
          </Button>
          <Button variant="primary" onClick={() => navigate('/platform/things')}>
            查看设备列表
          </Button>
        </div>
      </Card>

      <Card className="overflow-hidden">
        <Table>
          <THead>
            <tr>
              <TH>设备</TH>
              <TH>MQTT 认证</TH>
              <TH>HTTP Token</TH>
              <TH>Device Secret</TH>
              <TH>Topic (up / down)</TH>
            </tr>
          </THead>
          <TBody>
            {things.map((thing) => (
              <TR key={thing.id}>
                <TD>
                  <p className="font-medium">{thing.deviceName}</p>
                  <p className="font-mono text-[11px] text-faint">{thing.deviceKey}</p>
                </TD>
                <TD className="text-xs">
                  <p className="font-mono text-[11px]">clientId: {thing.mqttClientId}</p>
                  <p className="font-mono text-[11px]">username: {thing.mqttUsername}</p>
                  <p className="font-mono text-[11px]">
                    password: <SecretCell value={thing.mqttPassword} />
                  </p>
                </TD>
                <TD>
                  <SecretCell value={thing.token} />
                </TD>
                <TD>
                  <SecretCell value={thing.deviceSecret} />
                </TD>
                <TD className="text-xs">
                  <p className="font-mono text-[11px]">{thing.upTopic}</p>
                  <p className="font-mono text-[11px] text-faint">{thing.downTopic}</p>
                </TD>
              </TR>
            ))}
          </TBody>
        </Table>
      </Card>
    </div>
  )
}
