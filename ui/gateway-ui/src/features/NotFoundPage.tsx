import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/Button'

export function NotFoundPage() {
  return (
    <div className="flex min-h-[50vh] flex-col items-center justify-center gap-3 text-center">
      <div className="text-4xl font-semibold text-faint">404</div>
      <p className="text-sm text-muted">页面不存在</p>
      <Link to="/overview">
        <Button variant="primary">返回概览</Button>
      </Link>
    </div>
  )
}
