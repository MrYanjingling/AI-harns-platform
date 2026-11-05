import { useEffect, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'
import { useCreateUser, useOrgTree, useRoles, useUpdateUser } from '@/api/hooks'
import type { UserInfo } from '@/api/types'
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

const schema = z.object({
  userName: z.string().trim().min(1, '请输入用户名').max(64, '用户名不能超过 64 个字符'),
  userNo: z.string().trim().min(1, '请输入工号').max(64, '工号不能超过 64 个字符'),
  nickName: z.string().trim().min(1, '请输入昵称').max(64, '昵称不能超过 64 个字符'),
  password: z
    .string()
    .max(64, '密码不能超过 64 个字符')
    .refine((value) => value === '' || value.length >= 6, '密码至少 6 位'),
  birthDate: z.string(),
  gender: z.string(),
  mail: z
    .string()
    .trim()
    .max(128, '邮箱不能超过 128 个字符')
    .refine((value) => value === '' || EMAIL_PATTERN.test(value), '邮箱格式不正确'),
  phone: z.string().trim().max(32, '手机号不能超过 32 个字符'),
  note: z.string().trim().max(255, '备注不能超过 255 个字符'),
  authKey: z.string().trim().max(128, '认证密钥不能超过 128 个字符'),
  authMode: z.string(),
  expireTimes: z.string().regex(/^\d*$/, '有效时长必须是非负整数'),
  status: z.string(),
})

type FormValues = z.infer<typeof schema>

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
        toast.success('用户已更新')
      } else {
        await createMutation.mutateAsync({ ...base, roleIds, orgIds })
        toast.success(
          values.password
            ? '用户已创建'
            : `用户已创建,初始密码为 ${values.userName}@qwer`,
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
      title={isEdit ? '编辑用户' : '新建用户'}
      description={
        isEdit
          ? '角色与组织绑定仅在创建时维护,更新接口不修改绑定关系。'
          : '密码留空时使用默认密码「用户名@qwer」。'
      }
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={pending}>
            取消
          </Button>
          <Button variant="primary" onClick={onSubmit} loading={pending}>
            {isEdit ? '保存' : '创建'}
          </Button>
        </>
      }
    >
      <div className="max-h-[68vh] space-y-4 overflow-y-auto pr-1">
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="用户名" htmlFor="user-name" required error={errors.userName?.message}>
            <Input id="user-name" {...register('userName')} />
          </Field>
          <Field label="工号" htmlFor="user-no" required error={errors.userNo?.message}>
            <Input id="user-no" {...register('userNo')} />
          </Field>
          <Field label="昵称" htmlFor="user-nick" required error={errors.nickName?.message}>
            <Input id="user-nick" {...register('nickName')} />
          </Field>
          <Field
            label="密码"
            htmlFor="user-password"
            error={errors.password?.message}
            hint={isEdit ? '留空表示不修改' : '留空使用默认密码'}
          >
            <Input
              id="user-password"
              type="password"
              autoComplete="new-password"
              {...register('password')}
            />
          </Field>
          <Field label="出生日期" htmlFor="user-birth">
            <Input id="user-birth" type="date" {...register('birthDate')} />
          </Field>
          <Field label="性别" htmlFor="user-gender">
            <Select id="user-gender" {...register('gender')}>
              <option value="">未设置</option>
              {GENDERS.map((item) => (
                <option key={item.value} value={item.value}>
                  {item.label}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="邮箱" htmlFor="user-mail" error={errors.mail?.message}>
            <Input id="user-mail" type="email" {...register('mail')} />
          </Field>
          <Field label="手机号" htmlFor="user-phone" error={errors.phone?.message}>
            <Input id="user-phone" {...register('phone')} />
          </Field>
          <Field label="认证方式" htmlFor="user-auth-mode">
            <Select id="user-auth-mode" {...register('authMode')}>
              {AUTH_MODES.map((item) => (
                <option key={item.value} value={item.value}>
                  {item.label}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="认证密钥" htmlFor="user-auth-key" error={errors.authKey?.message}>
            <Input id="user-auth-key" placeholder="LDAP / RADIUS 时使用" {...register('authKey')} />
          </Field>
          <Field label="状态" htmlFor="user-status">
            <Select id="user-status" {...register('status')}>
              <option value="enabled">启用</option>
              <option value="disabled">禁用</option>
            </Select>
          </Field>
          <Field
            label="有效时长"
            htmlFor="user-expire"
            hint="分钟;0 表示使用系统默认"
            error={errors.expireTimes?.message}
          >
            <Input id="user-expire" type="number" min={0} {...register('expireTimes')} />
          </Field>
        </div>

        <Field label="备注" htmlFor="user-note" error={errors.note?.message}>
          <Textarea id="user-note" {...register('note')} />
        </Field>

        {!isEdit ? (
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="rounded-xl border border-border p-3">
              <p className="mb-2 text-xs font-medium text-muted">绑定角色</p>
              {roles.isLoading ? (
                <Spinner className="size-4 text-primary" />
              ) : (roles.data?.list ?? []).length === 0 ? (
                <p className="text-[11px] text-faint">暂无可绑定角色</p>
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

            <div className="rounded-xl border border-border p-3">
              <p className="mb-2 text-xs font-medium text-muted">绑定组织</p>
              {orgTree.isLoading ? (
                <Spinner className="size-4 text-primary" />
              ) : orgOptions.length === 0 ? (
                <p className="text-[11px] text-faint">暂无可绑定组织</p>
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
          <div className="rounded-xl border border-border bg-surface-2/50 px-3 py-2 text-[11px] leading-relaxed text-muted">
            已绑定角色:
            {user?.roles?.length ? user.roles.map((role) => role.name).join('、') : '无'}
            ;已绑定组织:
            {user?.orgs?.length ? user.orgs.map((org) => org.name).join('、') : '无'}
          </div>
        )}
      </div>
    </Dialog>
  )
}
