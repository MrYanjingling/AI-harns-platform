import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  FormDialog,
  Field,
  Input,
  SearchSelect,
  showErrorToast,
  useI18n,
} from '@harns/core'
import { useCreatePushConnector, useUpdatePushConnector } from '@/api/hooks'
import type {
  ConnectorConfig,
  KafkaSASLMechanism,
  PushConnector,
  PushConnectorPayload,
} from '@/api/types'

interface ConnectorForm {
  name: string
  type: string
  brokerUrl: string
  username: string
  password: string
  clientId: string
  groupId: string
  saslMechanism: KafkaSASLMechanism
  qos: string
  keepAlive: string
}

const KAFKA_ID_LETTERS = 'abcdefghijklmnopqrstuvwxyz'

function randomKafkaID(): string {
  const values = new Uint8Array(6)
  crypto.getRandomValues(values)
  return `harns-${Array.from(values, (value) => KAFKA_ID_LETTERS[value % KAFKA_ID_LETTERS.length]).join('')}`
}

const EMPTY_CONNECTOR: ConnectorForm = {
  name: '',
  type: 'mqtt',
  brokerUrl: '',
  username: '',
  password: '',
  clientId: '',
  groupId: '',
  saslMechanism: 'none',
  qos: '0',
  keepAlive: '60',
}

const SASL_MECHANISMS: KafkaSASLMechanism[] = ['none', 'plain', 'scram-sha-256', 'scram-sha-512']

function connectorToForm(connector: PushConnector): ConnectorForm {
  const config = connector.pushConfig ?? ({} as ConnectorConfig)
  const type = connector.type || 'mqtt'
  const configuredMechanism = config.saslMechanism as KafkaSASLMechanism | undefined
  const saslMechanism =
    configuredMechanism && SASL_MECHANISMS.includes(configuredMechanism)
      ? configuredMechanism
      : config.username || config.password
        ? 'plain'
        : 'none'
  return {
    name: connector.name,
    type,
    brokerUrl: config.brokerUrl ?? '',
    username: config.username ?? '',
    password: config.password ?? '',
    clientId: config.clientId || (type === 'kafka' ? randomKafkaID() : ''),
    groupId: config.groupId || (type === 'kafka' ? randomKafkaID() : ''),
    saslMechanism,
    qos: String(config.qos ?? 0),
    keepAlive: String(config.keepAlive ?? 60),
  }
}

function connectorPayload(form: ConnectorForm): PushConnectorPayload {
  const config: ConnectorConfig = {
    brokerUrl: form.brokerUrl.trim(),
  }
  if (form.clientId.trim()) config.clientId = form.clientId.trim()
  if (form.type === 'kafka') {
    if (form.groupId.trim()) config.groupId = form.groupId.trim()
    config.saslMechanism = form.saslMechanism
    if (form.saslMechanism !== 'none') {
      if (form.username.trim()) config.username = form.username.trim()
      if (form.password.trim()) config.password = form.password.trim()
    }
  } else {
    if (form.username.trim()) config.username = form.username.trim()
    if (form.password.trim()) config.password = form.password.trim()
    config.qos = Number(form.qos)
    const keepAlive = Number(form.keepAlive)
    if (Number.isFinite(keepAlive) && keepAlive > 0) config.keepAlive = keepAlive
  }
  return { type: form.type, name: form.name.trim(), pushConfig: config }
}

function validateConnector(
  form: ConnectorForm,
  t: ReturnType<typeof useI18n>['t'],
): Record<string, string> {
  const errors: Record<string, string> = {}
  if (!form.name.trim()) errors.name = t('gateway.form.validation.nameRequired')
  if (!form.brokerUrl.trim()) errors.brokerUrl = t('gateway.push.validation.brokerRequired')
  else if (form.type === 'kafka') {
    const valid = form.brokerUrl
      .split(',')
      .map((item) => item.trim())
      .every((item) => /^(?:(?:tcp|kafka):\/\/)?[^:\s]+:\d+$/.test(item))
    if (!valid) errors.brokerUrl = t('gateway.push.validation.brokerFormat')
  } else if (!/^(mqtt|ws|wss|ssl|tcp):\/\/.+/i.test(form.brokerUrl.trim())) {
    errors.brokerUrl = t('gateway.push.validation.brokerFormat')
  }
  const keepAlive = Number(form.keepAlive)
  if (form.keepAlive.trim() !== '' && (!Number.isFinite(keepAlive) || keepAlive <= 0)) {
    errors.keepAlive = t('gateway.push.validation.keepAliveInvalid')
  }
  return errors
}

interface ConnectorFormDialogProps {
  open: boolean
  connector: PushConnector | null
  onClose: () => void
}

export function ConnectorFormDialog({ open, connector, onClose }: ConnectorFormDialogProps) {
  const { t } = useI18n()
  const create = useCreatePushConnector()
  const update = useUpdatePushConnector()
  const [form, setForm] = useState<ConnectorForm>(EMPTY_CONNECTOR)
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!open) return
    setForm(connector ? connectorToForm(connector) : EMPTY_CONNECTOR)
    setErrors({})
  }, [open, connector])

  const changeType = (type: string) => {
    setForm((current) => {
      const next = { ...current, type }
      if (type === 'kafka') {
        if (!next.clientId.trim()) next.clientId = randomKafkaID()
        if (!next.groupId.trim()) next.groupId = randomKafkaID()
      }
      return next
    })
  }

  const submit = async () => {
    const nextErrors = validateConnector(form, t)
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length > 0) return
    try {
      if (connector) {
        await update.mutateAsync({ id: connector.id, payload: connectorPayload(form) })
        toast.success(t('gateway.push.toast.connectorUpdated'))
      } else {
        await create.mutateAsync(connectorPayload(form))
        toast.success(t('gateway.push.toast.connectorCreated'))
      }
      onClose()
    } catch (error) {
      showErrorToast(error)
    }
  }

  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={
        connector
          ? t('gateway.push.connector.editTitle')
          : t('gateway.push.connector.createTitle')
      }
      footer={
        <>
          <Button variant="outline" onClick={onClose}>
            {t('common.action.cancel')}
          </Button>
          <Button
            variant="primary"
            loading={create.isPending || update.isPending}
            onClick={() => void submit()}
          >
            {t('common.action.save')}
          </Button>
        </>
      }
    >
      <Card>
        <CardHeader>
          <CardTitle>{t('gateway.form.section.basic')}</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-x-8 gap-y-4 sm:grid-cols-2">
          <Field
            label={t('common.field.name')}
            htmlFor="push-connector-name"
            labelClassName="text-sm"
            labelWidthClassName="w-[96px]"
            inline
            required
            error={errors.name}
          >
            <Input
              id="push-connector-name"
              value={form.name}
              className="w-full max-w-[240px]"
              autoFocus
              autoComplete="off"
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
          </Field>
          <Field
            label={t('gateway.push.field.type')}
            htmlFor="push-connector-type"
            labelClassName="text-sm"
            labelWidthClassName="w-[96px]"
            inline
          >
            <SearchSelect
              id="push-connector-type"
              value={form.type}
              options={[
                { value: 'mqtt', label: 'MQTT' },
                { value: 'kafka', label: 'KAFKA' },
              ]}
              onChange={changeType}
              ariaLabel={t('gateway.push.field.type')}
              emptyText={t('common.state.empty')}
              searchable
              className="w-[240px]"
            />
          </Field>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('gateway.form.section.protocol')}</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-x-8 gap-y-4 sm:grid-cols-2">
          <Field
            label={t('gateway.push.field.brokerUrl')}
            htmlFor="push-connector-broker"
            labelClassName="text-sm"
            labelWidthClassName="w-[96px]"
            inline
            required
            error={errors.brokerUrl}
          >
            <Input
              id="push-connector-broker"
              value={form.brokerUrl}
              className="w-full max-w-[240px]"
              placeholder={form.type === 'kafka' ? 'localhost:9092' : 'tcp://broker:1883'}
              autoComplete="off"
              onChange={(event) => setForm({ ...form, brokerUrl: event.target.value })}
            />
          </Field>
          <Field
            label={t('gateway.push.field.clientId')}
            htmlFor="push-connector-client-id"
            labelClassName="text-sm"
            labelWidthClassName="w-[96px]"
            inline
          >
            <Input
              id="push-connector-client-id"
              value={form.clientId}
              className="w-full max-w-[240px]"
              autoComplete="off"
              onChange={(event) => setForm({ ...form, clientId: event.target.value })}
            />
          </Field>
          {form.type === 'kafka' ? (
            <>
              <Field
                label={t('gateway.push.field.groupId')}
                htmlFor="push-connector-group-id"
                labelClassName="text-sm"
                labelWidthClassName="w-[96px]"
                inline
              >
                <Input
                  id="push-connector-group-id"
                  value={form.groupId}
                  className="w-full max-w-[240px]"
                  autoComplete="off"
                  onChange={(event) => setForm({ ...form, groupId: event.target.value })}
                />
              </Field>
              <Field
                label={t('gateway.push.field.saslMechanism')}
                htmlFor="push-connector-sasl"
                labelClassName="text-sm"
                labelWidthClassName="w-[96px]"
                inline
              >
                <SearchSelect
                  id="push-connector-sasl"
                  value={form.saslMechanism}
                  options={[
                    { value: 'none', label: t('gateway.push.sasl.none') },
                    { value: 'plain', label: 'SASL PLAIN' },
                    { value: 'scram-sha-256', label: 'SASL SCRAM-SHA-256' },
                    { value: 'scram-sha-512', label: 'SASL SCRAM-SHA-512' },
                  ]}
                  onChange={(saslMechanism) =>
                    setForm({ ...form, saslMechanism: saslMechanism as KafkaSASLMechanism })
                  }
                  ariaLabel={t('gateway.push.field.saslMechanism')}
                  emptyText={t('common.state.empty')}
                  searchable={false}
                  className="w-[240px]"
                />
              </Field>
            </>
          ) : null}
          {form.type === 'mqtt' || form.saslMechanism !== 'none' ? (
            <>
              <Field
                label={t('gateway.push.field.username')}
                htmlFor="push-connector-username"
                labelClassName="text-sm"
                labelWidthClassName="w-[96px]"
                inline
              >
                <Input
                  id="push-connector-username"
                  value={form.username}
                  className="w-full max-w-[240px]"
                  autoComplete="off"
                  onChange={(event) => setForm({ ...form, username: event.target.value })}
                />
              </Field>
              <Field
                label={t('gateway.push.field.password')}
                htmlFor="push-connector-password"
                labelClassName="text-sm"
                labelWidthClassName="w-[96px]"
                inline
              >
                <Input
                  id="push-connector-password"
                  type="password"
                  className="w-full max-w-[240px]"
                  value={form.password}
                  autoComplete="new-password"
                  onChange={(event) => setForm({ ...form, password: event.target.value })}
                />
              </Field>
            </>
          ) : null}
          {form.type === 'mqtt' ? (
            <>
          <Field
            label={t('gateway.push.field.qos')}
            htmlFor="push-connector-qos"
            labelClassName="text-sm"
            labelWidthClassName="w-[96px]"
            inline
          >
            <SearchSelect
              id="push-connector-qos"
              value={form.qos}
              options={[
                { value: '0', label: '0' },
                { value: '1', label: '1' },
                { value: '2', label: '2' },
              ]}
              onChange={(qos) => setForm({ ...form, qos })}
              ariaLabel={t('gateway.push.field.qos')}
              emptyText={t('common.state.empty')}
              searchable={false}
              className="w-[240px]"
            />
          </Field>
          <Field
            label={t('gateway.push.field.keepAlive')}
            htmlFor="push-connector-keep-alive"
            labelClassName="text-sm"
            labelWidthClassName="w-[96px]"
            inline
            error={errors.keepAlive}
          >
            <Input
              id="push-connector-keep-alive"
              value={form.keepAlive}
              className="w-full max-w-[240px]"
              inputMode="numeric"
              onChange={(event) => setForm({ ...form, keepAlive: event.target.value })}
            />
          </Field>
            </>
          ) : null}
        </CardContent>
      </Card>
    </FormDialog>
  )
}
