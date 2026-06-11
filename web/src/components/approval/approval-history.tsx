import {format} from 'date-fns'
import {CheckIcon, XIcon, SendIcon} from 'lucide-react'
import {Separator} from '@/components/ui/separator'
import type {ApprovalRecord} from '@/types'

interface ApprovalHistoryProps {
  approvals: ApprovalRecord[]
}

const actionIcons: Record<string, React.ReactNode> = {
  approve: <CheckIcon className="w-3 h-3 text-green-600" />,
  reject: <XIcon className="w-3 h-3 text-red-600" />,
  submit: <SendIcon className="w-3 h-3 text-blue-600" />,
}

export function ApprovalHistory({ approvals }: ApprovalHistoryProps) {
  if (!approvals?.length) return null

  const sorted = [...approvals].sort(
    (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
  )

  return (
    <div className="space-y-2">
      <h3 className="text-sm font-medium">Approval History</h3>
      <div className="space-y-1">
        {sorted.map((a, i) => (
          <div key={a.id}>
            {i > 0 && <Separator className="my-1" />}
            <div className="flex items-start gap-2 text-xs text-muted-foreground py-1">
              <div className="mt-0.5 shrink-0">
                {actionIcons[a.action] ?? null}
              </div>
              <div className="flex-1">
                <span className="font-medium capitalize">{a.action}</span>
                {' by '}
                <span className="capitalize">{a.actor_role}</span>
                <span className="ml-2">
                  {format(new Date(a.created_at), 'MMM d, HH:mm')}
                </span>
                {a.comment && (
                  <p className="mt-0.5 italic text-muted-foreground/80">
                    &ldquo;{a.comment}&rdquo;
                  </p>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
