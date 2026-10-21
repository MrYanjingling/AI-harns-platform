/* 业务无关的公共层:网络、格式化、主题、国际化、UI 原语与应用外壳。 */

export { cn } from './lib/cn'
export {
  buildQuery,
  errorMessage,
  isApiError,
  request,
  ApiError,
  type ApiFieldError,
  type ApiResponse,
  type RequestOptions,
  type Service,
} from './lib/http'
export {
  formatApiTime,
  formatApiTimeMs,
  formatBytes,
  formatDateTime,
  formatDateTimeShort,
  formatRelative,
  fromDateTimeLocal,
  maskSecret,
  parseDate,
  percentValue,
  toDateTimeLocal,
} from './lib/format'
export {
  clearSessionToken,
  getSessionToken,
  notifySessionExpired,
  SESSION_EXPIRED_EVENT,
  setSessionToken,
} from './lib/session'
export { queryClient } from './lib/query-client'
export { PAGE_SIZES } from './lib/constants'

export {
  coreMessages,
  defineMessages,
  DEFAULT_LOCALE,
  I18nProvider,
  LOCALES,
  LOCALE_STORAGE_KEY,
  getLocale,
  isLocale,
  mergeMessages,
  registerMessages,
  setCurrentLocale,
  translate,
  useI18n,
  type InterpolationVars,
  type Locale,
  type MessageKey,
  type Messages,
  type Translate,
} from './i18n'

export { ThemeProvider, useTheme } from './providers/ThemeProvider'

export { Badge, type BadgeProps } from './components/ui/Badge'
export { Button, type ButtonProps } from './components/ui/Button'
export {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from './components/ui/Card'
export { ConfirmDialog, type ConfirmDialogProps } from './components/ui/ConfirmDialog'
export { CopyButton } from './components/ui/CopyButton'
export { Dialog, type DialogProps } from './components/ui/Dialog'
export { EmptyState } from './components/ui/EmptyState'
export { Field, Input, Label, Select, Textarea, type FieldProps } from './components/ui/Field'
export { ProgressBar } from './components/ui/ProgressBar'
export { Skeleton, TableSkeleton } from './components/ui/Skeleton'
export { PageLoading, Spinner } from './components/ui/Spinner'
export { Switch } from './components/ui/Switch'
export { Table, TBody, TD, TH, THead, TR } from './components/ui/Table'
export { Tabs, type TabItem } from './components/ui/Tabs'

export { GaugeRing, toneForPercent, type GaugeTone } from './components/charts/GaugeRing'
export { ErrorState } from './components/shared/ErrorState'
export { PageHeader } from './components/shared/PageHeader'
export { Pagination } from './components/shared/Pagination'
export { StatCard } from './components/shared/StatCard'
export { NotFoundPage } from './components/shared/NotFoundPage'

export { AccountMenu, type AccountMenuProps } from './components/layout/AccountMenu'
export { AppShell, type AppShellProps } from './components/layout/AppShell'
export { LanguageSwitcher } from './components/layout/LanguageSwitcher'
export { PagePad } from './components/layout/PagePad'
export { TechBackdrop } from './components/layout/TechBackdrop'
export {
  buildNavEntries,
  sectionForPath,
  titleKeyFromMatches,
  type NavEntry,
  type NavGroup,
  type NavLeaf,
  type NavSection,
} from './components/layout/nav'
