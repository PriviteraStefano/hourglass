import {type EntryStatus} from '@/types'
import {cn} from '@/lib/utils.ts'

interface StatusBadgeProps {
  status: EntryStatus
  className?: string
}

const statusConfig: Record<EntryStatus, { label: string; className: string }> = {
  draft: {
    label: 'Draft',
    className: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300',
  },
  submitted: {
    label: 'Submitted',
    className: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300',
  },
  pending_manager: {
    label: 'Pending Manager',
    className: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300',
  },
  pending_finance: {
    label: 'Pending Finance',
    className: 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300',
  },
  approved: {
    label: 'Approved',
    className: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300',
  },
  rejected: {
    label: 'Rejected',
    className: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300',
  },
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const config = statusConfig[status] ?? {className: ""}
  
  return (
    <span
      className={cn(
        'inline-flex items-center px-2 py-0.5 rounded text-xs font-medium',
        config.className,
        className
      )}
    >
      {config.label}
    </span>
  )
}