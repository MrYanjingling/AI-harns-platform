import { useState } from 'react'
import { AccountMenu } from '@harns/core'
import { useAuth } from '@/providers/AuthProvider'

/** 平台子应用的账号菜单:退出登录依赖平台会话。 */
export function PlatformAccountMenu() {
  const { user, signOut } = useAuth()
  const [signingOut, setSigningOut] = useState(false)

  const handleSignOut = async () => {
    setSigningOut(true)
    try {
      await signOut()
    } finally {
      setSigningOut(false)
    }
  }

  return (
    <AccountMenu
      name={user?.nickName || user?.userName}
      subtitle={user?.userNo || user?.userName}
      onSignOut={handleSignOut}
      signingOut={signingOut}
    />
  )
}
