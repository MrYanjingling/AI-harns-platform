import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { useMemo } from 'react'
import { Info, Save, X } from 'lucide-react'
import type { DeviceType } from '@/api/types'
import { useI18n } from '@/i18n'
import { BAUD_RATES, DATA_BITS, MEMORY_LAYOUTS, PARITIES, PROTOCOLS, STOP_BITS } from '@/lib/constants'
import { Button } from '@/components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Field, Input, Select } from '@/components/ui/Field'
import { createDeviceFormSchema, type DeviceFormValues } from './schema'
import { VariablesEditor } from './VariablesEditor'

export interface DeviceFormProps {
  mode: 'create' | 'edit'
  type: DeviceType
  initialValues: DeviceFormValues
  submitting: boolean
  onSubmit: (values: DeviceFormValues) => void
  onCancel: () => void
}

export function DeviceForm({
  mode,
  type,
  initialValues,
  submitting,
  onSubmit,
  onCancel,
}: DeviceFormProps) {
  const { t } = useI18n()
  const protocol = PROTOCOLS[type]
  const schema = useMemo(() => createDeviceFormSchema(t), [t])
  const form = useForm<DeviceFormValues>({
    resolver: zodResolver(schema),
    defaultValues: initialValues,
    mode: 'onTouched',
  })
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isDirty },
  } = form

  const model = watch('deviceModel')
  const isSerial = type === 'modbus' && model === 'modbusRtu'
  const title = useMemo(
    () => (mode === 'create' ? t('layout.title.deviceNew') : t('layout.title.deviceEdit')),
    [mode, t],
  )

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="space-y-4 pb-2"
      noValidate
      aria-label={title}
    >
      <Card>
        <CardHeader>
          <div>
            <CardTitle>{t('devices.form.basic')}</CardTitle>
            <CardDescription>{t('devices.form.basicDescription')}</CardDescription>
          </div>
          <span className="rounded-md bg-surface-2 px-2 py-1 text-[11px] font-medium text-muted">
            {protocol.label}
          </span>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Field
            label={t('devices.form.name')}
            htmlFor="device-name"
            required
            error={errors.name?.message}
          >
            <Input
              id="device-name"
              placeholder={t('devices.form.namePlaceholder')}
              {...register('name')}
              aria-invalid={!!errors.name}
            />
          </Field>
          <Field
            label={t('devices.form.code')}
            htmlFor="device-code"
            required
            hint={t('devices.form.codeHint')}
            error={errors.deviceCode?.message}
          >
            <Input
              id="device-code"
              placeholder={t('devices.form.codePlaceholder')}
              {...register('deviceCode')}
              aria-invalid={!!errors.deviceCode}
            />
          </Field>
          <Field
            label={t('devices.form.model')}
            htmlFor="device-model"
            required
            error={errors.deviceModel?.message}
          >
            {mode === 'create' && protocol.models.length > 1 ? (
              <Select
                id="device-model"
                {...register('deviceModel', {
                  onChange: (event) => {
                    // 切换型号时补齐该型号的默认端口,避免沿用另一个型号的端口。
                    if (event.target.value === 'modbusRtu') {
                      setValue('modbusLocation', '')
                    }
                  },
                })}
              >
                {protocol.models.map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label} — {t(item.hintKey)}
                  </option>
                ))}
              </Select>
            ) : (
              <Input id="device-model" value={model} readOnly disabled className="font-mono" />
            )}
          </Field>

          <Field
            label={t('devices.form.collectorCycle')}
            htmlFor="collector-cycle"
            required
            hint={t('devices.form.collectorHint')}
            error={errors.collectorCycle?.message}
          >
            <Input
              id="collector-cycle"
              type="number"
              min={1}
              max={86400}
              {...register('collectorCycle', { valueAsNumber: true })}
            />
          </Field>
          <Field
            label={t('devices.form.variableInterval')}
            htmlFor="variable-interval"
            hint={t('devices.form.variableIntervalHint')}
            error={errors.variableInterval?.message}
          >
            <Input
              id="variable-interval"
              type="number"
              min={0}
              max={86400}
              {...register('variableInterval', { valueAsNumber: true })}
            />
          </Field>
          <Field
            label={t('devices.form.topic')}
            htmlFor="topic"
            hint={t('devices.form.topicHint')}
            error={errors.topic?.message}
          >
            <Input
              id="topic"
              placeholder={t('devices.form.topicPlaceholder')}
              {...register('topic')}
            />
          </Field>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div>
            <CardTitle>{t('devices.form.connection')}</CardTitle>
            <CardDescription>{t(protocol.descriptionKey)}</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {type === 'modbus' ? (
            <>
              <Field
                label={
                  isSerial ? t('devices.form.serialDevice') : t('devices.form.deviceAddress')
                }
                htmlFor="mb-location"
                required
                hint={
                  isSerial ? t('devices.form.serialHint') : t('devices.form.addressHint')
                }
                error={errors.modbusLocation?.message}
              >
                <Input
                  id="mb-location"
                  placeholder={isSerial ? '/dev/ttyS0' : '192.168.1.10'}
                  {...register('modbusLocation')}
                />
              </Field>
              {isSerial ? (
                <>
                  <Field
                    label={t('devices.form.baud')}
                    htmlFor="mb-baud"
                    error={errors.modbusBaudRate?.message}
                  >
                    <Select
                      id="mb-baud"
                      {...register('modbusBaudRate', { valueAsNumber: true })}
                    >
                      {BAUD_RATES.map((rate) => (
                        <option key={rate} value={rate}>
                          {rate}
                        </option>
                      ))}
                    </Select>
                  </Field>
                  <Field
                    label={t('devices.form.dataBits')}
                    htmlFor="mb-databits"
                    error={errors.modbusDataBits?.message}
                  >
                    <Select
                      id="mb-databits"
                      {...register('modbusDataBits', { valueAsNumber: true })}
                    >
                      {DATA_BITS.map((bits) => (
                        <option key={bits} value={bits}>
                          {bits}
                        </option>
                      ))}
                    </Select>
                  </Field>
                  <Field
                    label={t('devices.form.parity')}
                    htmlFor="mb-parity"
                    error={errors.modbusParity?.message}
                  >
                    <Select id="mb-parity" {...register('modbusParity')}>
                      {PARITIES.map((item) => (
                        <option key={item.value} value={item.value}>
                          {t(item.labelKey)}
                        </option>
                      ))}
                    </Select>
                  </Field>
                  <Field
                    label={t('devices.form.stopBits')}
                    htmlFor="mb-stopbits"
                    error={errors.modbusStopBits?.message}
                  >
                    <Select id="mb-stopbits" {...register('modbusStopBits')}>
                      {STOP_BITS.map((item) => (
                        <option key={item.value} value={item.value}>
                          {item.label}
                        </option>
                      ))}
                    </Select>
                  </Field>
                </>
              ) : (
                <Field
                  label={t('devices.form.port')}
                  htmlFor="mb-port"
                  required
                  error={errors.modbusPort?.message}
                >
                  <Input
                    id="mb-port"
                    type="number"
                    min={1}
                    max={65535}
                    {...register('modbusPort', { valueAsNumber: true })}
                  />
                </Field>
              )}
              <Field
                label={t('devices.form.slave')}
                htmlFor="mb-slave"
                required
                error={errors.slave?.message}
              >
                <Input
                  id="mb-slave"
                  type="number"
                  min={1}
                  max={65535}
                  {...register('slave', { valueAsNumber: true })}
                />
              </Field>
              <Field
                label={t('devices.form.memoryLayout')}
                htmlFor="mb-memory"
                hint={t('devices.form.memoryLayoutHint')}
                error={errors.memoryLayout?.message}
              >
                <Select id="mb-memory" {...register('memoryLayout')}>
                  {MEMORY_LAYOUTS.map((layout) => (
                    <option key={layout} value={layout}>
                      {layout}
                    </option>
                  ))}
                </Select>
              </Field>
              <Field
                label={t('devices.form.startOffset')}
                htmlFor="mb-position"
                hint={t('devices.form.startOffsetHint')}
                error={errors.positionAddress?.message}
              >
                <Input
                  id="mb-position"
                  type="number"
                  min={0}
                  max={65535}
                  {...register('positionAddress', { valueAsNumber: true })}
                />
              </Field>
            </>
          ) : null}

          {type === 'opcUa' ? (
            <>
              <Field
                label="Endpoint"
                htmlFor="ua-location"
                required
                hint={t('devices.form.endpointHint')}
                error={errors.opcLocation?.message}
                className="sm:col-span-2"
              >
                <Input
                  id="ua-location"
                  placeholder="opc.tcp://192.168.1.20"
                  className="font-mono"
                  {...register('opcLocation')}
                />
              </Field>
              <Field
                label={t('devices.form.port')}
                htmlFor="ua-port"
                required
                error={errors.opcPort?.message}
              >
                <Input
                  id="ua-port"
                  type="number"
                  min={1}
                  max={65535}
                  {...register('opcPort', { valueAsNumber: true })}
                />
              </Field>
              <Field
                label={t('devices.form.username')}
                htmlFor="ua-username"
                error={errors.opcUsername?.message}
              >
                <Input
                  id="ua-username"
                  autoComplete="off"
                  placeholder={t('devices.form.anonymousHint')}
                  {...register('opcUsername')}
                />
              </Field>
              <Field
                label={t('devices.form.password')}
                htmlFor="ua-password"
                error={errors.opcPassword?.message}
              >
                <Input
                  id="ua-password"
                  type="password"
                  autoComplete="new-password"
                  placeholder={t('devices.form.anonymousHint')}
                  {...register('opcPassword')}
                />
              </Field>
            </>
          ) : null}

          {type === 's7' ? (
            <>
              <Field
                label={t('devices.form.plcAddress')}
                htmlFor="s7-location"
                required
                hint={t('devices.form.addressHint')}
                error={errors.s7Location?.message}
              >
                <Input
                  id="s7-location"
                  placeholder="192.168.1.30"
                  {...register('s7Location')}
                />
              </Field>
              <Field
                label={t('devices.form.port')}
                htmlFor="s7-port"
                required
                error={errors.s7Port?.message}
              >
                <Input
                  id="s7-port"
                  type="number"
                  min={1}
                  max={65535}
                  {...register('s7Port', { valueAsNumber: true })}
                />
              </Field>
              <Field
                label={t('devices.form.rack')}
                htmlFor="s7-rack"
                error={errors.s7Rack?.message}
              >
                <Input
                  id="s7-rack"
                  type="number"
                  min={0}
                  max={7}
                  {...register('s7Rack', { valueAsNumber: true })}
                />
              </Field>
              <Field
                label={t('devices.form.slot')}
                htmlFor="s7-slot"
                error={errors.s7Slot?.message}
              >
                <Input
                  id="s7-slot"
                  type="number"
                  min={0}
                  max={31}
                  {...register('s7Slot', { valueAsNumber: true })}
                />
              </Field>
            </>
          ) : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div>
            <CardTitle>{t('devices.form.variables')}</CardTitle>
            <CardDescription>{t('devices.form.variablesDescription')}</CardDescription>
          </div>
        </CardHeader>
        <CardContent>
          <VariablesEditor form={form} type={type} />
          {errors.variables?.root?.message ? (
            <p className="mt-2 text-xs text-danger">{errors.variables.root.message}</p>
          ) : null}
        </CardContent>
      </Card>

      <div className="sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-3 rounded border border-border bg-surface/85 px-4 py-3 shadow-sm backdrop-blur">
        <p className="flex items-center gap-1.5 text-xs text-faint">
          <Info className="size-3.5 shrink-0" />
          {mode === 'create' ? t('devices.form.createHint') : t('devices.form.editHint')}
          {isDirty ? (
            <span className="ml-1 text-warning">
              · {t('devices.form.unsaved')}
            </span>
          ) : null}
        </p>
        <div className="flex items-center gap-2">
          <Button variant="ghost" icon={<X className="size-4" />} onClick={onCancel}>
            {t('common.action.cancel')}
          </Button>
          <Button
            type="submit"
            variant="primary"
            loading={submitting}
            icon={<Save className="size-4" />}
          >
            {mode === 'create' ? t('devices.form.create') : t('devices.form.save')}
          </Button>
        </div>
      </div>
    </form>
  )
}
