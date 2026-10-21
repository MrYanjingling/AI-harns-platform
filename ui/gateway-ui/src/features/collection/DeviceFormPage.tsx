import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { AlertCircle, Check, Save } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  ErrorState,
  Field,
  Input,
  PageHeader,
  PageLoading,
  Select,
  errorMessage,
  useI18n,
} from '@harns/core'
import { useCreateDevice, useDevice, useUpdateDevice } from '@/api/hooks'
import type { DeviceType } from '@/api/types'
import {
  BAUD_RATES,
  DATA_BITS,
  DEVICE_TYPES,
  MEMORY_LAYOUTS,
  PARITIES,
  STOP_BITS,
  deviceTypeMeta,
  usesSerial,
} from '@/lib/constants'
import {
  createDefaultFormValues,
  createEmptyVariable,
  detailToForm,
  validateDeviceForm,
  type DeviceFormValues,
  type VariableFormValues,
} from '@/lib/device-form'
import { VariablesEditor } from './VariablesEditor'

export function DeviceFormPage({ mode }: { mode: 'create' | 'edit' }) {
  const { t } = useI18n()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = mode === 'edit'
  const device = useDevice(id, false)
  const createDevice = useCreateDevice()
  const updateDevice = useUpdateDevice()
  const [values, setValues] = useState<DeviceFormValues>(createDefaultFormValues)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [submitError, setSubmitError] = useState<string | null>(null)

  useEffect(() => {
    if (!isEdit || !device.data) return
    setValues(detailToForm(device.data))
  }, [isEdit, device.data])

  const typeMeta = useMemo(
    () => deviceTypeMeta(values.deviceType) ?? DEVICE_TYPES[0],
    [values.deviceType],
  )

  const patch = (key: keyof DeviceFormValues, value: string) => {
    setValues((current) => ({ ...current, [key]: value }))
  }

  const selectType = (type: DeviceType) => {
    if (isEdit) return
    const meta = DEVICE_TYPES.find((item) => item.value === type)
    if (!meta) return
    setValues((current) => ({
      ...current,
      deviceType: type,
      deviceModel: meta.models[0].value,
      port: type === 'opcUa' ? '4840' : type === 's7' ? '102' : '502',
      variables: [createEmptyVariable(type)],
    }))
    setErrors({})
  }

  const addVariable = () => {
    setValues((current) => ({
      ...current,
      variables: [...current.variables, createEmptyVariable(current.deviceType)],
    }))
  }

  const removeVariable = (index: number) => {
    setValues((current) => ({
      ...current,
      variables: current.variables.filter((_, itemIndex) => itemIndex !== index),
    }))
  }

  const changeVariable = (index: number, next: Partial<VariableFormValues>) => {
    setValues((current) => ({
      ...current,
      variables: current.variables.map((item, itemIndex) =>
        itemIndex === index ? { ...item, ...next } : item,
      ),
    }))
  }

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setSubmitError(null)
    const validation = validateDeviceForm(values, t)
    setErrors(validation.errors)
    if (!validation.payload) {
      setSubmitError(t('gateway.form.toast.failed'))
      return
    }

    try {
      if (isEdit) {
        if (!id || !device.data) return
        await updateDevice.mutateAsync({
          id,
          eTag: device.data.eTag,
          name: validation.payload.name,
          payload: validation.payload,
        })
        toast.success(t('gateway.form.toast.updated'))
        navigate(`/gateway/collection/${id}`)
      } else {
        const created = await createDevice.mutateAsync(validation.payload)
        toast.success(t('gateway.form.toast.created'))
        navigate(`/gateway/collection/${created.id}`)
      }
    } catch (error) {
      const message = errorMessage(error)
      setSubmitError(message)
      toast.error(message)
    }
  }

  if (isEdit && device.isLoading) return <PageLoading />
  if (isEdit && device.isError) {
    return <ErrorState error={device.error} onRetry={() => void device.refetch()} />
  }
  if (isEdit && !device.data) {
    return (
      <EmptyState
        title={t('gateway.detail.notFound')}
        action={
          <Link
            to="/gateway/collection"
            className="inline-flex h-8 items-center rounded bg-primary px-3 text-sm font-medium text-primary-fg"
          >
            {t('gateway.form.backToCollection')}
          </Link>
        }
      />
    )
  }

  const serial = values.deviceType === 'modbus' && usesSerial(values.deviceModel)
  const submitting = createDevice.isPending || updateDevice.isPending

  return (
    <div>
      <PageHeader
        title={isEdit ? t('gateway.form.editTitle') : t('gateway.form.createTitle')}
        description={
          isEdit ? t('gateway.form.editDescription') : t('gateway.form.createDescription')
        }
        backTo="/gateway/collection"
        backLabel={t('gateway.form.backToCollection')}
      />

      <form onSubmit={(event) => void submit(event)} className="space-y-4" noValidate>
        <Card>
          <CardHeader>
            <div>
              <CardTitle>{t('gateway.form.section.protocol')}</CardTitle>
              <CardDescription>{typeMeta ? t(typeMeta.descriptionKey) : ''}</CardDescription>
            </div>
          </CardHeader>
          <CardContent className="space-y-3">
            {isEdit ? (
              <div className="flex flex-wrap items-center gap-3 rounded border border-border bg-surface-2 px-3 py-2.5">
                <span className="text-sm font-medium text-fg">{typeMeta.label}</span>
                <span className="text-xs text-muted">{values.deviceModel}</span>
                <span className="text-xs text-faint">{t('gateway.form.editProtocolLocked')}</span>
              </div>
            ) : (
              <div className="grid gap-2 sm:grid-cols-3">
                {DEVICE_TYPES.map((item) => {
                  const active = item.value === values.deviceType
                  return (
                    <button
                      key={item.value}
                      type="button"
                      onClick={() => selectType(item.value)}
                      className={
                        active
                          ? 'rounded border border-primary bg-primary-soft p-3 text-left'
                          : 'rounded border border-border bg-surface p-3 text-left transition-colors hover:border-border-strong hover:bg-surface-2'
                      }
                    >
                      <span className="flex items-center justify-between gap-2">
                        <span className="text-sm font-semibold text-fg">{item.label}</span>
                        {active ? <Check className="size-4 text-primary" /> : null}
                      </span>
                      <span className="mt-1 block text-xs leading-relaxed text-muted">
                        {t(item.descriptionKey)}
                      </span>
                    </button>
                  )
                })}
              </div>
            )}

            <Field label={t('gateway.form.field.deviceModel')} htmlFor="deviceModel" required>
              <Select
                id="deviceModel"
                value={values.deviceModel}
                disabled={isEdit}
                onChange={(event) => patch('deviceModel', event.target.value)}
              >
                {typeMeta.models.map((model) => (
                  <option key={model.value} value={model.value}>
                    {model.label}
                  </option>
                ))}
              </Select>
            </Field>
            {errors.deviceModel ? (
              <p className="text-xs text-danger">{errors.deviceModel}</p>
            ) : null}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t('gateway.form.section.basic')}</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <Field
              label={t('gateway.form.field.name')}
              htmlFor="name"
              error={errors.name}
              required
            >
              <Input
                id="name"
                value={values.name}
                placeholder={t('gateway.form.field.namePlaceholder')}
                onChange={(event) => patch('name', event.target.value)}
              />
            </Field>
            <Field
              label={t('gateway.form.field.deviceCode')}
              htmlFor="deviceCode"
              error={errors.deviceCode}
              required
            >
              <Input
                id="deviceCode"
                value={values.deviceCode}
                placeholder={t('gateway.form.field.deviceCodePlaceholder')}
                onChange={(event) => patch('deviceCode', event.target.value)}
              />
            </Field>
            <Field
              label={t('gateway.form.field.collectorCycle')}
              htmlFor="collectorCycle"
              hint={t('gateway.form.field.collectorCycleHint')}
              error={errors.collectorCycle}
              required
            >
              <Input
                id="collectorCycle"
                value={values.collectorCycle}
                inputMode="numeric"
                onChange={(event) => patch('collectorCycle', event.target.value)}
              />
            </Field>
            <Field
              label={t('gateway.form.field.variableInterval')}
              htmlFor="variableInterval"
              hint={t('gateway.form.field.variableIntervalHint')}
              error={errors.variableInterval}
            >
              <Input
                id="variableInterval"
                value={values.variableInterval}
                inputMode="numeric"
                onChange={(event) => patch('variableInterval', event.target.value)}
              />
            </Field>
            <Field
              label={t('gateway.form.field.topic')}
              htmlFor="topic"
              hint={t('gateway.form.field.topicHint')}
              error={errors.topic}
              className="sm:col-span-2"
            >
              <Input
                id="topic"
                value={values.topic}
                placeholder="data/<gatewayId>/v1/<deviceId>"
                onChange={(event) => patch('topic', event.target.value)}
              />
            </Field>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div>
              <CardTitle>{t('gateway.form.section.connection')}</CardTitle>
              <CardDescription>
                {values.deviceType === 'modbus'
                  ? t('gateway.protocol.modbus.description')
                  : values.deviceType === 'opcUa'
                    ? t('gateway.protocol.opcUa.description')
                    : t('gateway.protocol.s7.description')}
              </CardDescription>
            </div>
          </CardHeader>
          <CardContent className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <Field
              label={t('gateway.form.field.location')}
              htmlFor="location"
              hint={
                serial
                  ? t('gateway.form.field.serialLocationHint')
                  : t('gateway.form.field.locationHint')
              }
              error={errors.location}
              required
            >
              <Input
                id="location"
                value={values.location}
                placeholder={serial ? '/dev/ttyS0' : '127.0.0.1'}
                onChange={(event) => patch('location', event.target.value)}
              />
            </Field>

            {values.deviceType === 'modbus' && serial ? (
              <>
                <Field
                  label={t('gateway.form.field.baudRate')}
                  htmlFor="baudRate"
                  error={errors.baudRate}
                >
                  <Select
                    id="baudRate"
                    value={values.baudRate}
                    onChange={(event) => patch('baudRate', event.target.value)}
                  >
                    {BAUD_RATES.map((rate) => (
                      <option key={rate} value={String(rate)}>
                        {rate}
                      </option>
                    ))}
                  </Select>
                </Field>
                <Field
                  label={t('gateway.form.field.dataBits')}
                  htmlFor="dataBits"
                  error={errors.dataBits}
                >
                  <Select
                    id="dataBits"
                    value={values.dataBits}
                    onChange={(event) => patch('dataBits', event.target.value)}
                  >
                    {DATA_BITS.map((bits) => (
                      <option key={bits} value={String(bits)}>
                        {bits}
                      </option>
                    ))}
                  </Select>
                </Field>
                <Field label={t('gateway.form.field.parity')} htmlFor="parity">
                  <Select
                    id="parity"
                    value={values.parity}
                    onChange={(event) => patch('parity', event.target.value)}
                  >
                    {PARITIES.map((parity) => (
                      <option key={parity} value={parity}>
                        {parity}
                      </option>
                    ))}
                  </Select>
                </Field>
                <Field label={t('gateway.form.field.stopBits')} htmlFor="stopBits">
                  <Select
                    id="stopBits"
                    value={values.stopBits}
                    onChange={(event) => patch('stopBits', event.target.value)}
                  >
                    {STOP_BITS.map((bits) => (
                      <option key={bits} value={bits}>
                        {bits}
                      </option>
                    ))}
                  </Select>
                </Field>
              </>
            ) : (
              <Field
                label={t('gateway.form.field.port')}
                htmlFor="port"
                error={errors.port}
                required
              >
                <Input
                  id="port"
                  value={values.port}
                  inputMode="numeric"
                  onChange={(event) => patch('port', event.target.value)}
                />
              </Field>
            )}

            {values.deviceType === 'modbus' ? (
              <>
                <Field
                  label={t('gateway.form.field.slave')}
                  htmlFor="slave"
                  error={errors.slave}
                  required
                >
                  <Input
                    id="slave"
                    value={values.slave}
                    inputMode="numeric"
                    onChange={(event) => patch('slave', event.target.value)}
                  />
                </Field>
                <Field
                  label={t('gateway.form.field.memoryLayout')}
                  htmlFor="memoryLayout"
                  required
                >
                  <Select
                    id="memoryLayout"
                    value={values.memoryLayout}
                    onChange={(event) => patch('memoryLayout', event.target.value)}
                  >
                    {MEMORY_LAYOUTS.map((layout) => (
                      <option key={layout} value={layout}>
                        {layout}
                      </option>
                    ))}
                  </Select>
                </Field>
                <Field
                  label={t('gateway.form.field.positionAddress')}
                  htmlFor="positionAddress"
                  error={errors.positionAddress}
                >
                  <Input
                    id="positionAddress"
                    value={values.positionAddress}
                    inputMode="numeric"
                    placeholder="0"
                    onChange={(event) => patch('positionAddress', event.target.value)}
                  />
                </Field>
              </>
            ) : null}

            {values.deviceType === 'opcUa' ? (
              <>
                <Field
                  label={t('gateway.form.field.username')}
                  htmlFor="username"
                >
                  <Input
                    id="username"
                    value={values.username}
                    autoComplete="off"
                    onChange={(event) => patch('username', event.target.value)}
                  />
                </Field>
                <Field label={t('gateway.form.field.password')} htmlFor="password">
                  <Input
                    id="password"
                    type="password"
                    value={values.password}
                    autoComplete="new-password"
                    onChange={(event) => patch('password', event.target.value)}
                  />
                </Field>
              </>
            ) : null}

            {values.deviceType === 's7' ? (
              <>
                <Field label={t('gateway.form.field.rack')} htmlFor="rack" error={errors.rack}>
                  <Input
                    id="rack"
                    value={values.rack}
                    inputMode="numeric"
                    onChange={(event) => patch('rack', event.target.value)}
                  />
                </Field>
                <Field label={t('gateway.form.field.slot')} htmlFor="slot" error={errors.slot}>
                  <Input
                    id="slot"
                    value={values.slot}
                    inputMode="numeric"
                    onChange={(event) => patch('slot', event.target.value)}
                  />
                </Field>
              </>
            ) : null}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div>
              <CardTitle>{t('gateway.form.section.variables')}</CardTitle>
              <CardDescription>
                {t('gateway.collection.variableCount', { count: values.variables.length })}
              </CardDescription>
            </div>
          </CardHeader>
          <CardContent>
            <VariablesEditor
              deviceType={values.deviceType}
              variables={values.variables}
              errors={errors}
              onAdd={addVariable}
              onRemove={removeVariable}
              onChange={changeVariable}
            />
          </CardContent>
        </Card>

        {submitError ? (
          <div
            role="alert"
            className="flex items-start gap-2 rounded border border-danger/25 bg-danger/5 px-3 py-2.5 text-xs text-danger"
          >
            <AlertCircle className="mt-0.5 size-4 shrink-0" />
            <span className="leading-relaxed">{submitError}</span>
          </div>
        ) : null}

        <div className="flex items-center justify-end gap-2">
          <Link
            to="/gateway/collection"
            className="inline-flex h-8 items-center rounded border border-border-strong bg-surface px-3 text-sm text-fg transition-colors hover:bg-surface-2"
          >
            {t('gateway.form.cancel')}
          </Link>
          <Button
            type="submit"
            variant="primary"
            loading={submitting}
            icon={<Save className="size-4" />}
          >
            {isEdit ? t('gateway.form.submit.save') : t('gateway.form.submit.create')}
          </Button>
        </div>
      </form>
    </div>
  )
}
