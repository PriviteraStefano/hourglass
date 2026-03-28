import {
  ClockIcon,
  ReceiptIcon,
  CheckCircleIcon,
  FileTextIcon,
  FolderIcon,
  SettingsIcon,
  type LucideIcon
} from 'lucide-react'
import {Link} from '@tanstack/react-router'
import {Separator} from '@/src/components/ui/separator'
import {cn} from '@/lib/utils'

const navItems: Array<{ label: string; href: string; icon: LucideIcon; disabled?: boolean }> = [
  {label: 'Time', href: '/time-entries', icon: ClockIcon},
  {label: 'Expenses', href: '/expenses', icon: ReceiptIcon, disabled: true},
  {label: 'Approvals', href: '/approvals', icon: CheckCircleIcon, disabled: true},
  {label: 'divider', href: '', icon: FolderIcon, disabled: true},
  {label: 'Contracts', href: '/contracts', icon: FileTextIcon, disabled: true},
  {label: 'Projects', href: '/projects', icon: FolderIcon, disabled: true},
  {label: 'divider', href: '', icon: FolderIcon, disabled: true},
  {label: 'Settings', href: '/settings', icon: SettingsIcon, disabled: true},
]

export function Sidebar() {
  return (
    <aside className="w-56 border-r bg-muted/30 p-4">
      <nav className="space-y-1">
        {navItems.map((item, i) => {
          if (item.label === 'divider') {
            return <Separator key={i} className="my-3"/>
          } else {
            return (
              <Link
                key={item.href}
                to={item.href}
                disabled={item.disabled}
                className={cn(
                  'flex items-center gap-2 px-3 py-2 rounded-md text-sm transition-colors',
                  item.disabled
                    ? 'opacity-50 cursor-not-allowed pointer-events-none'
                    : 'hover:bg-muted text-foreground'
                )}
              >
                <item.icon className="w-4 h-4"/>
                {item.label}
                {item.disabled && (
                  <span className="ml-auto text-xs text-muted-foreground">Soon</span>
                )}
              </Link>
            )
          }
        })}
      </nav>
    </aside>
  )
}