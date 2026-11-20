import { useEffect, useState, type ChangeEvent, type FormEvent } from 'react'
import { Eye, EyeOff } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  Card,
  CardContent,
  CopyButton,
  ErrorState,
  Field,
  Input,
  cn,
  Skeleton,
  Switch,
  Textarea,
  showErrorToast,
  useI18n,
} from '@harns/core'
import {
  useAISettings,
  useTestAIModel,
  useUpdateAIModel,
  useUpdateAISettings,
} from '@/api/hooks'
import type { AIModelPayload } from '@/api/types'

const EMPTY_FORM: AIModelPayload = {
  name: '',
  remark: '',
  model: '',
  apiKey: '',
  endpointUrl: '',
}

function validEndpoint(value: string): boolean {
  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

function SecretInput({
  id,
  value,
  placeholder,
  autoComplete,
  readOnly = false,
  onChange,
  className,
  inputClassName,
}: {
  id: string
  value: string
  placeholder?: string
  autoComplete?: string
  readOnly?: boolean
  onChange?: (event: ChangeEvent<HTMLInputElement>) => void
  className?: string
  inputClassName?: string
}) {
  const { t } = useI18n()
  const [visible, setVisible] = useState(false)
  return (
    <div className={cn('relative', className)}>
      <Input
        id={id}
        type={visible ? 'text' : 'password'}
        value={value}
        placeholder={placeholder}
        autoComplete={autoComplete}
        readOnly={readOnly}
        className={cn('pr-9', inputClassName)}
        onChange={onChange}
      />
      <button
        type="button"
        aria-label={t(visible ? 'gateway.ai.hideSecret' : 'gateway.ai.showSecret')}
        title={t(visible ? 'gateway.ai.hideSecret' : 'gateway.ai.showSecret')}
        onClick={() => setVisible((current) => !current)}
        className="absolute top-1/2 right-2 flex size-6 -translate-y-1/2 items-center justify-center rounded text-faint transition-colors hover:bg-surface-2 hover:text-fg"
      >
        {visible ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
      </button>
    </div>
  )
}

export function AIModelsPage() {
  const { t } = useI18n()
  const settings = useAISettings()
  const updateSettings = useUpdateAISettings()
  const updateModel = useUpdateAIModel()
  const testModel = useTestAIModel()
  const [form, setForm] = useState<AIModelPayload>(EMPTY_FORM)
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!settings.data) return
    setForm({
      name: settings.data.name,
      remark: settings.data.remark,
      model: settings.data.model,
      endpointUrl: settings.data.endpointUrl,
      apiKey: '',
    })
    setErrors({})
  }, [settings.data])

  const assistantEnabled = settings.data?.assistantEnabled ?? false
  const mcpEnabled = settings.data?.mcpEnabled ?? false
  const token = settings.data?.mcpToken ?? ''
  const apiKeyLocked = Boolean(settings.data?.hasApiKey)

  const updateSwitches = (next: { assistantEnabled: boolean; mcpEnabled: boolean }) => {
    updateSettings.mutate(next, {
      onSuccess: () => {
        if (next.assistantEnabled !== assistantEnabled) {
          toast.success(
            t(
              next.assistantEnabled
                ? 'gateway.ai.models.enabledToast'
                : 'gateway.ai.models.disabledToast',
            ),
          )
        }
        if (next.mcpEnabled !== mcpEnabled) {
          toast.success(
            t(next.mcpEnabled ? 'gateway.ai.mcp.enabledToast' : 'gateway.ai.mcp.disabledToast'),
          )
        }
      },
      onError: showErrorToast,
    })
  }

  const validate = (): AIModelPayload | undefined => {
    const next: Record<string, string> = {}
    const name = form.name.trim()
    const model = form.model.trim()
    const endpointUrl = form.endpointUrl.trim()
    if (!name) next.name = t('gateway.ai.validation.nameRequired')
    if (!model) next.model = t('gateway.ai.validation.modelRequired')
    if (!endpointUrl || !validEndpoint(endpointUrl)) {
      next.endpointUrl = t('gateway.ai.validation.endpointInvalid')
    }
    if (!apiKeyLocked && !form.apiKey?.trim()) {
      next.apiKey = t('gateway.ai.validation.apiKeyRequired')
    }
    setErrors(next)
    if (Object.keys(next).length > 0) return undefined
    return {
      name,
      remark: form.remark.trim(),
      model,
      endpointUrl,
      ...(!apiKeyLocked && form.apiKey?.trim() ? { apiKey: form.apiKey.trim() } : {}),
    }
  }

  const save = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const payload = validate()
    if (!payload) return
    updateModel.mutate(payload, {
      onSuccess: () => {
        setForm((current) => ({ ...current, apiKey: '' }))
        toast.success(t('gateway.ai.toast.updated'))
      },
      onError: showErrorToast,
    })
  }

  const test = () => {
    const payload = validate()
    if (!payload) return
    testModel.mutate(payload, {
      onSuccess: () => toast.success(t('gateway.ai.models.testSuccess')),
      onError: showErrorToast,
    })
  }

  if (settings.isLoading) {
    return (
      <div className="grid gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]">
        <Skeleton className="h-80 w-full" />
        <Skeleton className="h-28 w-full" />
      </div>
    )
  }

  if (settings.isError) {
    return (
      <div className="flex min-h-0 flex-1 flex-col">
        <ErrorState error={settings.error} onRetry={settings.refetch} compact />
      </div>
    )
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      <div className="grid gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]">
        <Card>
          <CardContent className="space-y-4 p-4">
            <div className="flex items-center justify-between gap-4">
              <p className="text-sm font-medium text-muted">
                {t('gateway.ai.models.enable')}
              </p>
              <Switch
                checked={assistantEnabled}
                label={t('gateway.ai.models.enable')}
                disabled={updateSettings.isPending}
                onChange={(checked) =>
                  updateSwitches({ assistantEnabled: checked, mcpEnabled })
                }
              />
            </div>

            {assistantEnabled ? (
              <form
                id="gateway-ai-model-form"
                className="space-y-4 border-t border-border pt-4"
                onSubmit={save}
              >
                <Field
                  label={t('gateway.ai.field.name')}
                  htmlFor="gateway-ai-config-name"
                  labelClassName="text-sm"
                  error={errors.name}
                  required
                >
                  <Input
                    id="gateway-ai-config-name"
                    value={form.name}
                    placeholder={t('gateway.ai.placeholder.name')}
                    onChange={(event) => setForm({ ...form, name: event.target.value })}
                  />
                </Field>
                <Field
                  label={t('common.field.remark')}
                  htmlFor="gateway-ai-model-remark"
                  labelClassName="text-sm"
                >
                  <Textarea
                    id="gateway-ai-model-remark"
                    value={form.remark}
                    rows={3}
                    className="min-h-20 resize-none"
                    onChange={(event) => setForm({ ...form, remark: event.target.value })}
                  />
                </Field>
                <Field
                  label={t('gateway.ai.field.apiKey')}
                  htmlFor="gateway-ai-model-key"
                  labelClassName="text-sm"
                  error={errors.apiKey}
                  required={!apiKeyLocked}
                >
                  <Input
                    id="gateway-ai-model-key"
                    type={apiKeyLocked ? 'text' : 'password'}
                    autoComplete="new-password"
                    value={
                      apiKeyLocked
                        ? (settings.data?.apiKey || '********')
                        : (form.apiKey ?? '')
                    }
                    placeholder="sk-..."
                    disabled={apiKeyLocked}
                    onChange={(event) => setForm({ ...form, apiKey: event.target.value })}
                  />
                </Field>
                <Field
                  label={t('gateway.ai.field.model')}
                  htmlFor="gateway-ai-model-name"
                  labelClassName="text-sm"
                  error={errors.model}
                  required
                >
                  <Input
                    id="gateway-ai-model-name"
                    value={form.model}
                    placeholder={t('gateway.ai.placeholder.model')}
                    onChange={(event) => setForm({ ...form, model: event.target.value })}
                  />
                </Field>
                <Field
                  label={t('gateway.ai.field.endpointUrl')}
                  htmlFor="gateway-ai-model-endpoint"
                  labelClassName="text-sm"
                  error={errors.endpointUrl}
                  required
                >
                  <Input
                    id="gateway-ai-model-endpoint"
                    value={form.endpointUrl}
                    placeholder="https://api.openai.com/v1/chat/completions"
                    onChange={(event) =>
                      setForm({ ...form, endpointUrl: event.target.value })
                    }
                  />
                </Field>
                <div className="flex items-center justify-end gap-2 border-t border-border pt-3">
                  <Button
                    variant="outline"
                    loading={testModel.isPending}
                    disabled={updateModel.isPending}
                    onClick={test}
                  >
                    {t('gateway.ai.models.test')}
                  </Button>
                  <Button
                    type="submit"
                    variant="primary"
                    loading={updateModel.isPending}
                    disabled={testModel.isPending}
                  >
                    {t('common.action.save')}
                  </Button>
                </div>
              </form>
            ) : null}
          </CardContent>
        </Card>

        <Card className="h-fit">
          <CardContent className="space-y-4 p-4">
            <div className="flex items-center justify-between gap-4">
              <p className="text-sm font-medium text-muted">{t('gateway.ai.mcp.enable')}</p>
              <Switch
                checked={mcpEnabled}
                label={t('gateway.ai.mcp.enable')}
                disabled={updateSettings.isPending}
                onChange={(checked) =>
                  updateSwitches({ assistantEnabled, mcpEnabled: checked })
                }
              />
            </div>
            {mcpEnabled ? (
              <div className="space-y-3 border-t border-border pt-4">
                {token ? (
                  <div className="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-3">
                    <span className="text-sm font-medium text-muted">
                      {t('gateway.ai.mcp.token')}
                    </span>
                    <div className="flex min-w-0 items-center gap-2 rounded border border-border bg-surface-2 px-2">
                      <SecretInput
                        id="gateway-ai-mcp-token"
                        value={token}
                        readOnly
                        className="min-w-0 flex-1"
                        inputClassName="h-8 border-0 bg-transparent font-mono text-[13px] focus:ring-0"
                      />
                      <CopyButton value={token} />
                    </div>
                  </div>
                ) : null}
              </div>
            ) : null}
          </CardContent>
        </Card>
      </div>

    </div>
  )
}
