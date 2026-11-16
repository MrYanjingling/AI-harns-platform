import { useState, type ReactNode } from 'react'
import { Server, ShieldCheck, Upload } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Dialog,
  ErrorState,
  Skeleton,
  Textarea,
  cn,
  showErrorToast,
  formatDateTime,
  useI18n,
} from '@harns/core'
import { useImportLicense, useLicense, useSystemInfo } from '@/api/hooks'

const aboutCardClass = 'border-[1.5px] border-border-strong/75 shadow-[0_1px_3px_rgba(28,47,76,0.06)]'
const descriptionGridClass =
  'grid grid-cols-1 items-baseline gap-x-4 px-4 sm:grid-cols-[minmax(0,1fr)_340px]'

function DescriptionRow({
  label,
  value,
  nowrap = false,
}: {
  label: string
  value: ReactNode
  nowrap?: boolean
}) {
  return (
    <>
      <dt className="pt-2.5 pb-0 text-xs whitespace-nowrap text-muted sm:py-2.5 sm:text-sm">{label}</dt>
      <dd
        className={cn(
          'min-w-0 pt-0 pb-2.5 text-xs break-words text-fg sm:py-2.5 sm:text-sm',
          nowrap && 'break-all whitespace-normal sm:break-normal sm:whitespace-nowrap',
        )}
      >
        {value || '—'}
      </dd>
    </>
  )
}

function DescriptionSkeleton({ rows }: { rows: number }) {
  return (
    <dl className={descriptionGridClass}>
      {Array.from({ length: rows }).map((_, index) => (
        <DescriptionRow key={index} label="" value={<Skeleton className="h-4 w-48" />} />
      ))}
    </dl>
  )
}

export function AboutPage() {
  const { t } = useI18n()
  const systemInfo = useSystemInfo()
  const license = useLicense()
  const importMutation = useImportLicense()
  const [importOpen, setImportOpen] = useState(false)
  const [licenseData, setLicenseData] = useState('')
  const licenseInfo = license.data?.license
  const authorized = license.data?.status === 'authorized' && Boolean(licenseInfo)

  const closeImport = () => {
    setImportOpen(false)
    setLicenseData('')
  }

  const submitImport = () => {
    const data = licenseData.trim()
    if (!data) {
      toast.error(t('gateway.about.license.dataRequired'))
      return
    }
    importMutation.mutate(data, {
      onSuccess: () => {
        closeImport()
        toast.success(t('gateway.about.license.importSuccess'))
      },
      onError: (error) => {
        showErrorToast(error)
      },
    })
  }

  return (
    <div className="mx-auto w-full max-w-7xl space-y-4">
      <div className="grid gap-4 xl:grid-cols-2">
        <Card className={cn(aboutCardClass, 'flex h-full flex-col')}>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-1.5 text-sm font-medium text-muted">
              <Server className="size-4 text-faint" />
              {t('gateway.about.basic.title')}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex-1 p-0">
            {systemInfo.isError ? (
              <div className="p-4">
                <ErrorState error={systemInfo.error} onRetry={() => void systemInfo.refetch()} compact />
              </div>
            ) : systemInfo.isLoading ? (
              <DescriptionSkeleton rows={5} />
            ) : (
              <dl className={descriptionGridClass}>
                <DescriptionRow label={t('gateway.about.field.version')} value={systemInfo.data?.version} />
                <DescriptionRow
                  label={t('gateway.about.field.commitId')}
                  value={systemInfo.data?.commitId}
                />
                <DescriptionRow
                  label={t('gateway.about.field.hardwareId')}
                  value={systemInfo.data?.hardwareId}
                  nowrap
                />
                <DescriptionRow
                  label={t('gateway.about.field.architecture')}
                  value={systemInfo.data?.architecture}
                />
                <DescriptionRow
                  label={t('gateway.about.field.cpuModel')}
                  value={systemInfo.data?.cpuModel}
                />
                <DescriptionRow
                  label={t('gateway.about.field.osVersion')}
                  value={systemInfo.data?.osVersion}
                />
                <DescriptionRow
                  label={t('gateway.about.field.kernelVersion')}
                  value={systemInfo.data?.kernelVersion}
                />
              </dl>
            )}
          </CardContent>
        </Card>

        <Card className={cn(aboutCardClass, 'flex h-full flex-col')}>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-1.5 text-sm font-medium text-muted">
              <ShieldCheck className="size-4 text-faint" />
              {t('gateway.about.license.title')}
            </CardTitle>
            <Button
              variant="primary"
              size="sm"
              icon={<Upload className="size-3.5" />}
              onClick={() => setImportOpen(true)}
            >
              {t('gateway.about.license.import')}
            </Button>
          </CardHeader>
          <CardContent className="flex-1 p-0">
            {license.isError ? (
              <div className="p-4">
                <ErrorState error={license.error} onRetry={() => void license.refetch()} compact />
              </div>
            ) : license.isLoading ? (
              <DescriptionSkeleton rows={7} />
            ) : (
              <dl className={descriptionGridClass}>
                <DescriptionRow
                  label={t('gateway.about.license.status')}
                  value={
                    <span className={authorized ? 'font-medium text-success' : 'font-medium text-danger'}>
                      {authorized
                        ? t('gateway.about.license.authorized')
                        : t('gateway.about.license.unauthorized')}
                    </span>
                  }
                />
                {licenseInfo ? (
                  <>
                    <DescriptionRow
                      label={t('gateway.about.license.maxVariable')}
                      value={licenseInfo.maxVariable}
                    />
                    {authorized ? (
                      <>
                        <DescriptionRow
                          label={t('gateway.about.license.contact')}
                          value={licenseInfo.contact}
                        />
                        <DescriptionRow
                          label={t('gateway.about.license.contactEmail')}
                          value={licenseInfo.contactEmail}
                        />
                        <DescriptionRow
                          label={t('gateway.about.license.company')}
                          value={licenseInfo.company}
                        />
                        <DescriptionRow
                          label={t('gateway.about.license.expiresAt')}
                          value={formatDateTime(licenseInfo.licenseExpiresAt)}
                        />
                      </>
                    ) : null}
                    <DescriptionRow
                      label={t('gateway.about.license.softwareRunDeadline')}
                      value={formatDateTime(licenseInfo.softwareRunDeadline)}
                    />
                  </>
                ) : null}
              </dl>
            )}
          </CardContent>
        </Card>
      </div>

      <Dialog
        open={importOpen}
        onClose={closeImport}
        title={t('gateway.about.license.importTitle')}
        className="w-[574px] max-w-[calc(100vw-2rem)] rounded-lg"
        headerClassName="h-10 items-center pt-0"
        footerClassName="min-h-10 items-center py-1"
        footer={
          <>
            <Button variant="outline" onClick={closeImport}>
              {t('common.action.cancel')}
            </Button>
            <Button
              variant="primary"
              loading={importMutation.isPending}
              onClick={submitImport}
              className="self-center"
            >
              {t('common.action.save')}
            </Button>
          </>
        }
      >
        <Textarea
          id="gateway-license-data"
          aria-label={t('gateway.about.license.importTitle')}
          autoFocus
          value={licenseData}
          onChange={(event) => setLicenseData(event.target.value)}
          spellCheck={false}
          className="h-[90px] w-full resize-none border-faint font-mono text-xs hover:border-muted"
        />
      </Dialog>
    </div>
  )
}
