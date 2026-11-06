import { useEffect, useMemo, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'
import { useCreateUser, useOrgTree, useRoles, useUpdateUser } from '@/api/hooks'
import type { UserInfo } from '@/api/types'
import { useI18n, type Translate } from '@/i18n'
import { AUTH_MODES, GENDERS } from '@/lib/constants'
import { errorMessage } from '@/lib/http'
import { cn } from '@/lib/cn'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { Spinner } from '@/components/ui/Spinner'
import { flattenOrgs } from './org-utils'

const ROLE_LOOKUP_SIZE = 100
const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

type FormValues = {
  userName: string
  userNo: string
  nickName: string
  password: string
  birthDate: string
  gender: string
  mail: string
  phone: string
  note: string
  authKey: string
  authMode: string
  expireTimes: string
  status: string
}

function createUserFormSchema(t: Translate): z.ZodType<FormValues> {
  return z.object({
    userName: z
      .string()
      .trim()
      .min(1, t('identity.userForm.validation.userNameRequired'))
      .max(64, t('identity.userForm.validation.userNameMax')),
    userNo: z
      .string()
      .trim()
      .min(1, t('identity.userForm.validation.userNoRequired'))
      .max(64, t('identity.userForm.validation.userNoMax')),
    nickName: z
      .string()
      .trim()
      .min(1, t('identity.userForm.validation.nickNameRequired'))
      .max(64, t('identity.userForm.validation.nickNameMax')),
    password: z
      .string()
      .max(64, t('identity.userForm.validation.passwordMax'))
      .refine(
        (value) => value === '' || value.length >= 6,
        t('identity.userForm.validation.passwordMin'),
      ),
    birthDate: z.string(),
    gender: z.string(),
    mail: z
      .string()
      .trim()
      .max(128, t('identity.userForm.validation.mailMax'))
      .refine(
        (value) => value === '' || EMAIL_PATTERN.test(value),
        t('identity.userForm.validation.mailInvalid'),
      ),
    phone: z.string().trim().max(32, t('identity.userForm.validation.phoneMax')),
    note: z.string().trim().max(255, t('identity.userForm.validation.noteMax')),
    authKey: z.string().trim().max(128, t('identity.userForm.validation.authKeyMax')),
    authMode: z.string(),
    expireTimes: z
      .string()
      .regex(/^\d*$/, t('identity.userForm.validation.expireTimes')),
    status: z.string(),
  })
}

const EMPTY_FORM: FormValues = {
  userName: '',
  userNo: '',
  nickName: '',
  password: '',
  birthDate: '',
  gender: '',
  mail: '',
  phone: '',
  note: '',
  authKey: '',
  authMode: 'local',
  expireTimes: '0',
  status: 'enabled',
}

export function UserFormDialog({
  open,
  onClose,
  user,
}: {
  open: boolean
  onClose: () => void
  user?: UserInfo
}) {
  const isEdit = Boolean(user)
  const { t } = useI18n()
  const schema = useMemo(() => createUserFormSchema(t), [t])
  const createMutation = useCreateUser()
  const updateMutation = useUpdateUser()
  const roles = useRoles({ page: 1, pageSize: ROLE_LOOKUP_SIZE })
  const orgTree = useOrgTree()
  const [roleIds, setRoleIds] = useState<string[]>([])
  const [orgIds, setOrgIds] = useState<string[]>([])

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: EMPTY_FORM })

  useEffect(() => {
    if (!open) return
    if (user) {
      reset({
        userName: user.userName,
        userNo: user.userNo,
        nickName: user.nickName,
        password: '',
        birthDate: user.birthDate ?? '',
        gender: user.gender ?? '',
        mail: user.mail ?? '',
        phone: user.phone ?? '',
        note: user.note ?? '',
        authKey: '',
        authMode: user.authMode || 'local',
        expireTimes: String(user.expireTimes ?? 0),
        status: user.status || 'enabled',
      })
      setRoleIds(user.roleIds ?? [])
      setOrgIds(user.orgIds ?? [])
    } else {
      reset(EMPTY_FORM)
      setRoleIds([])
      setOrgIds([])
    }
  }, [open, user, reset])

  const pending = createMutation.isPending || updateMutation.isPending
  const orgOptions = flattenOrgs(orgTree.data ?? [])

  const toggle = (list: string[], id: string) =>
    list.includes(id) ? list.filter((item) => item !== id) : [...list, id]

  const onSubmit = handleSubmit(async (values) => {
    const base = {
      userName: values.userName,
      userNo: values.userNo,
      nickName: values.nickName,
      birthDate: values.birthDate || undefined,
      gender: values.gender ? (values.gender as 'male' | 'female') : undefined,
      mail: values.mail || undefined,
      phone: values.phone || undefined,
      note: values.note || undefined,
      authKey: values.authKey || undefined,
      authMode: values.authMode as 'local' | 'ldap' | 'radius',
      expireTimes: Number(values.expireTimes || 0),
      status: values.status as 'enabled' | 'disabled',
      ...(values.password ? { password: values.password } : {}),
    }
    try {
      if (user) {
        // 后端更新接口只接受基础字段,角色/组织绑定无法在此修改。
        await updateMutation.mutateAsync({ id: user.id, payload: base })
        toast.success(t('identity.userForm.updatedToast'))
      } else {
        await createMutation.mutateAsync({ ...base, roleIds, orgIds })
        toast.success(
          values.password
            ? t('identity.userForm.createdToast')
            : t('identity.userForm.createdWithPasswordToast', { name: values.userName }),
        )
      }
      onClose()
    } catch (error) {
      toast.error(errorMessage(error))
    }
  })

  return (
    <Dialog
      open={open}
      onClose={onClose}
      size="xl"
      title={isEdit ? t('identity.userForm.editTitle') : t('identity.userForm.createTitle')}
      description={
        isEdit
          ? t('identity.userForm.editDescription')
          : t('identity.userForm.createDescription')
      }
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={pending}>
            {t('common.action.cancel')}
          </Button>
          <Button variant="primary" onClick={onSubmit} loading={pending}>
            {isEdit ? t('common.action.save') : t('common.action.create')}
          </Button>
        </>
      }
    >
      <div className="max-h-[68vh] space-y-4 overflow-y-auto pr-1">
        <div className="grid gap-3 sm:grid-cols-2">
          <Field
            label={t('identity.userForm.userName')}
            htmlFor="user-name"
            required
            error={errors.userName?.message}
          >
            <Input id="user-name" {...register('userName')} />
          </Field>
          <Field
            label={t('identity.userForm.userNo')}
            htmlFor="user-no"
            required
            error={errors.userNo?.message}
          >
            <Input id="user-no" {...register('userNo')} />
          </Field>
          <Field
            label={t('identity.userForm.nickName')}
            htmlFor="user-nick"
            required
            error={errors.nickName?.message}
          >
            <Input id="user-nick" {...register('nickName')} />
          </Field>
          <Field
            label={t('identity.userForm.password')}
            htmlFor="user-password"
            error={errors.password?.message}
            hint={
              isEdit
                ? t('identity.userForm.passwordHintEdit')
                : t('identity.userForm.passwordHintCreate')
            }
          >
            <Input
              id="user-password"
              type="password"
              autoComplete="new-password"
              {...register('password')}
            />
          </Field>
          <Field label={t('identity.userForm.birthDate')} htmlFor="user-birth">
            <Input id="user-birth" type="date" {...register('birthDate')} />
          </Field>
          <Field label={t('identity.userForm.gender')} htmlFor="user-gender">
            <Select id="user-gender" {...register('gender')}>
              <option value="">{t('identity.userForm.genderUnset')}</option>
              {GENDERS.map((item) => (
                <option key={item.value} value={item.value}>
                  {t(item.labelKey)}
                </option>
              ))}
            </Select>
          </Field>
          <Field
            label={t('identity.userForm.mail')}
            htmlFor="user-mail"
            error={errors.mail?.message}
          >
            <Input id="user-mail" type="email" {...register('mail')} />
          </Field>
          <Field
            label={t('identity.userForm.phone')}
            htmlFor="user-phone"
            error={errors.phone?.message}
          >
            <Input id="user-phone" {...register('phone')} />
          </Field>
          <Field label={t('identity.userForm.authMode')} htmlFor="user-auth-mode">
            <Select id="user-auth-mode" {...register('authMode')}>
              {AUTH_MODES.map((item) => (
                <option key={item.value} value={item.value}>
                  {t(item.labelKey)}
                </option>
              ))}
            </Select>
          </Field>
          <Field
            label={t('identity.userForm.authKey')}
            htmlFor="user-auth-key"
            error={errors.authKey?.message}
          >
            <Input
              id="user-auth-key"
              placeholder={t('identity.userForm.authKeyPlaceholder')}
              {...register('authKey')}
            />
          </Field>
          <Field label={t('identity.userForm.status')} htmlFor="user-status">
            <Select id="user-status" {...register('status')}>
              <option value="enabled">{t('common.field.enabled')}</option>
              <option value="disabled">{t('common.field.disabled')}</option>
            </Select>
          </Field>
          <Field
            label={t('identity.userForm.expireTimes')}
            htmlFor="user-expire"
            hint={t('identity.userForm.expireHint')}
            error={errors.expireTimes?.message}
          >
            <Input id="user-expire" type="number" min={0} {...register('expireTimes')} />
          </Field>
        </div>

        <Field
          label={t('identity.userForm.note')}
          htmlFor="user-note"
          error={errors.note?.message}
        >
          <Textarea id="user-note" {...register('note')} />
        </Field>

        {!isEdit ? (
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="rounded border border-border p-3">
              <p className="mb-2 text-xs font-medium text-muted">
                {t('identity.userForm.bindRoles')}
              </p>
              {roles.isLoading ? (
                <Spinner className="size-4 text-primary" />
              ) : (roles.data?.list ?? []).length === 0 ? (
                <p className="text-[11px] text-faint">{t('identity.userForm.noRoles')}</p>
              ) : (
                <div className="max-h-40 space-y-1.5 overflow-y-auto">
                  {(roles.data?.list ?? []).map((role) => (
                    <label
                      key={role.id}
                      className={cn(
                        'flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1 text-xs hover:bg-surface-2',
                        roleIds.includes(role.id) && 'bg-primary-soft/50',
                      )}
                    >
                      <input
                        type="checkbox"
                        className="size-3.5 accent-[var(--primary)]"
                        checked={roleIds.includes(role.id)}
                        onChange={() => setRoleIds((current) => toggle(current, role.id))}
                      />
                      <span className="font-medium">{role.name}</span>
                      <span className="font-mono text-[10px] text-faint">{role.code}</span>
                    </label>
                  ))}
                </div>
              )}
            </div>

            <div className="rounded border border-border p-3">
              <p className="mb-2 text-xs font-medium text-muted">
                {t('identity.userForm.bindOrgs')}
              </p>
              {orgTree.isLoading ? (
                <Spinner className="size-4 text-primary" />
              ) : orgOptions.length === 0 ? (
                <p className="text-[11px] text-faint">{t('identity.userForm.noOrgs')}</p>
              ) : (
                <div className="max-h-40 space-y-1.5 overflow-y-auto">
                  {orgOptions.map((org) => (
                    <label
                      key={org.id}
                      className={cn(
                        'flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1 text-xs hover:bg-surface-2',
                        orgIds.includes(org.id) && 'bg-primary-soft/50',
                      )}
                      style={{ paddingLeft: `${8 + org.depth * 14}px` }}
                    >
                      <input
                        type="checkbox"
                        className="size-3.5 accent-[var(--primary)]"
                        checked={orgIds.includes(org.id)}
                        onChange={() => setOrgIds((current) => toggle(current, org.id))}
                      />
                      <span className="font-medium">{org.name}</span>
                      <span className="font-mono text-[10px] text-faint">{org.code}</span>
                    </label>
                  ))}
                </div>
              )}
            </div>
          </div>
        ) : (
          <div className="rounded border border-border bg-surface-2/50 px-3 py-2 text-[11px] leading-relaxed text-muted">
            {t('identity.userForm.boundRoles', {
              names: user?.roles?.length
                ? user.roles.map((role) => role.name).join(t('identity.listSeparator'))
                : t('common.state.none'),
            })}
            <br />
            {t('identity.userForm.boundOrgs', {
              names: user?.orgs?.length
                ? user.orgs.map((org) => org.name).join(t('identity.listSeparator'))
                : t('common.state.none'),
            })}
          </div>
        )}
      </div>
    </Dialog>
  )
}
