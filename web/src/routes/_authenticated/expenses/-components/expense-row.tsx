import {type Expense, CATEGORY_LABELS} from '@/types/expense-types'
import type {ExpenseCategory} from '@/types/expense-types'
import {Input} from '@/components/ui/input'
import {Button} from '@/components/ui/button'
import {SaveIcon, Trash2Icon, SendIcon, UploadIcon, FileIcon} from 'lucide-react'
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select'
import type {ChangeEvent} from "react"
import {useState} from "react"
import {StatusBadge} from '@/routes/_authenticated/time-entries/-components/status-badge.tsx'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog.tsx'

const EXPENSE_CATEGORIES: ExpenseCategory[] = [
  'mileage', 'meal', 'accommodation', 'parking',
  'travel_tickets', 'tolls', 'taxi', 'equipment', 'other',
]

interface ExpenseRowProps {
  expense: Expense
  editable: boolean
  onUpdate?: (field: string, value: string | number | null) => void
  onDelete?: () => void
  onSubmit?: () => void
  onReceiptUpload?: (file: File) => void
}

export function ExpenseRow({ expense, editable, onUpdate, onDelete, onSubmit, onReceiptUpload }: ExpenseRowProps) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [editing, setEditing] = useState(false)
  const [editValues, setEditValues] = useState<Partial<Expense>>({})

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file && onReceiptUpload) {
      onReceiptUpload(file)
    }
  }

  const toggleEdit = () => {
    if (editing) {
      // Save
      if (onUpdate && Object.keys(editValues).length > 0) {
        Object.entries(editValues).forEach(([field, value]) => {
          onUpdate(field, value)
        })
      }
      setEditValues({})
    }
    setEditing(!editing)
  }

  const handleFieldChange = (field: string, value: string | number | null) => {
    setEditValues(prev => ({...prev, [field]: value}))
    if (!editing && onUpdate) {
      onUpdate(field, value)
    }
  }

  const showKmDistance = expense.category === 'mileage' || editValues.category === 'mileage'

  return (
    <div className="flex items-center gap-2 p-2 bg-muted/30 rounded">
      <Select
        value={editValues.category ?? expense.category}
        onValueChange={(v) => handleFieldChange('category', v)}
        disabled={!editable}
      >
        <SelectTrigger className="w-36">
          <SelectValue placeholder="Category" />
        </SelectTrigger>
        <SelectContent>
          {EXPENSE_CATEGORIES.map((cat) => (
            <SelectItem key={cat} value={cat}>
              {CATEGORY_LABELS[cat]}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Input
        type="number"
        step="0.01"
        min="0"
        value={editValues.amount ?? expense.amount}
        onChange={(e: ChangeEvent<HTMLInputElement>) => handleFieldChange('amount', parseFloat(e.target.value) || 0)}
        disabled={!editable}
        className="w-24"
        placeholder="Amount"
      />

      {showKmDistance && (
        <Input
          type="number"
          step="0.1"
          min="0"
          value={editValues.km_distance ?? expense.km_distance ?? ''}
          onChange={(e: ChangeEvent<HTMLInputElement>) =>
            handleFieldChange('km_distance', e.target.value ? parseFloat(e.target.value) : null)
          }
          disabled={!editable}
          className="w-20"
          placeholder="Km"
        />
      )}

      <Input
        value={editValues.description ?? expense.description ?? ''}
        onChange={(e: ChangeEvent<HTMLInputElement>) => handleFieldChange('description', e.target.value)}
        placeholder="Description"
        disabled={!editable}
        className="flex-1"
      />

      <div className="flex items-center gap-1">
        {expense.receipt_url ? (
          <a
            href={expense.receipt_url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-muted-foreground hover:text-foreground flex items-center gap-1"
            title="View receipt"
          >
            <FileIcon className="w-3 h-3" />
          </a>
        ) : editable && onReceiptUpload && (
          <label className="cursor-pointer text-xs text-muted-foreground hover:text-foreground" title="Upload receipt">
            <UploadIcon className="w-3 h-3" />
            <input
              type="file"
              accept=".pdf,.jpg,.jpeg,.png"
              className="hidden"
              onChange={handleFileChange}
            />
          </label>
        )}
      </div>

      <StatusBadge status={expense.status} />

      <div className="flex gap-1">
        {editable && (
          <>
            {onSubmit && expense.status !== 'submitted' && (
              <Button variant="ghost" size="sm" onClick={onSubmit} title="Submit Expense">
                <SendIcon className="w-4 h-4" />
              </Button>
            )}
            <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
              <AlertDialogTrigger render={<Button variant="ghost" size="sm" title="Delete Expense"><Trash2Icon className="w-4 h-4" /></Button>} />
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Delete Expense?</AlertDialogTitle>
                  <AlertDialogDescription>
                    This action cannot be undone.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction variant="destructive" onClick={onDelete}>Delete</AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </>
        )}
      </div>
    </div>
  )
}
