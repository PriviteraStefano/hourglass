import {memo, useMemo, useState} from 'react'
import {Controller, useForm} from 'react-hook-form'
import {zodResolver} from '@hookform/resolvers/zod'
import {z} from 'zod'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {Textarea} from '@/components/ui/textarea'
import {ScrollArea} from '@/components/ui/scroll-area'
import {RadioGroup, RadioGroupItem} from '@/components/ui/radio-group'
import {Building2, ChevronDown, ChevronRight} from 'lucide-react'
import {cn} from '@/lib/utils'
import {Field, FieldContent, FieldError, FieldLabel} from '@/components/ui/field.tsx'
import type {UnitTreeNode} from '@/types/unit.ts'
import {getDescendantIds} from '../utils/tree-utils'
import {useOrgHierarchyStore} from "@/routes/_authenticated/org-hierarchy/-context/org-hierarchy-context.tsx";
import {useMutation, useSuspenseQuery} from "@tanstack/react-query";
import {createUnitMutationOpts, unitTreeQueryOpts, updateUnitMutationOpts} from "@/api/units.ts";
import {toast} from "sonner";

function generateSlug(value: string) {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, '')
    .replace(/\s+/g, '-')
    .substring(0, 50)
}

const unitFormSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  code: z.string().min(1, 'Code is required'),
  description: z.string().optional(),
  parent_unit_id: z.string().optional(),
})

type UnitFormData = z.infer<typeof unitFormSchema>

export function UnitFormDialog() {
  const {data: tree} = useSuspenseQuery(unitTreeQueryOpts)
  const {mutateAsync: createUnit} = useMutation(createUnitMutationOpts)
  const {mutateAsync: updateUnit} = useMutation(updateUnitMutationOpts)

  const unit = useOrgHierarchyStore(s => s.selectedUnit)
  const open = useOrgHierarchyStore(s => s.formOpen)
  const onOpenChange = useOrgHierarchyStore(s => s.setFormOpen)
  const mode = useOrgHierarchyStore(s => s.formMode)
  const editingUnit = useOrgHierarchyStore(s => s.editingUnit)

  const form = useForm<UnitFormData>({
    resolver: zodResolver(
      unitFormSchema.refine(
        (data) => !(mode === 'edit' && unit && data.parent_unit_id === unit.id),
        {message: 'Unit cannot be its own parent', path: ['parent_unit_id']}
      )
    ),
    defaultValues: {
      name: mode === 'edit' && editingUnit ? editingUnit.name : '',
      code: mode === 'edit' && editingUnit ? editingUnit.code : '',
      description: mode === 'edit' && editingUnit ? editingUnit.description || '' : '',
      parent_unit_id: mode === 'edit' && editingUnit ? editingUnit.parent_unit_id : undefined,
    },
  })

  const disabledIds = useMemo(() => {
    const ids = new Set<string>()
    if (mode === 'edit' && editingUnit) {
      ids.add(editingUnit.id)
      const descendants = getDescendantIds(editingUnit.id, tree)
      for (const id of descendants) ids.add(id)
    }
    return ids
  }, [mode, editingUnit, tree])

  const handleFormSubmit = async (data: UnitFormData) => {
    try {
      if (mode === 'edit' && editingUnit) {
        await toast.promise(
          updateUnit({
            id: editingUnit.id, body: {
              name: data.name.trim(),
              description: data.description?.trim() || null,
              code: data.code.trim(),
            }
          }), {
            loading: "Updating unit...",
            success: "Unit updated successfully",
            error: "Failed to update unit",
          }
        )
      } else {
        await toast.promise(
          createUnit({
            name: data.name.trim(),
            description: data.description?.trim() || null,
            code: data.code.trim(),
            parent_unit_id: data.parent_unit_id || null,
          }), {
            loading: "Creating unit...",
            success: "Unit created successfully",
            error: "Failed to create unit",
          }
        )
      }
      onOpenChange(false)
    } catch {
      // toast.promise already showed the error
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>{mode === 'create' ? 'Create Unit' : 'Edit Unit'}</DialogTitle>
          <DialogDescription>
            {mode === 'create'
              ? 'Add a new business unit to the organization.'
              : 'Update the business unit details.'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={form.handleSubmit(handleFormSubmit)} className="flex-1 overflow-hidden flex flex-col">
          <div className="flex-1 overflow-y-auto space-y-4 py-4">
            <Controller
              name="name"
              control={form.control}
              render={({field, fieldState}) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldContent>
                    <FieldLabel htmlFor="name">Name</FieldLabel>
                    <Input
                      id="name"
                      placeholder="Engineering"
                      {...field}
                      onChange={(e) => {
                        field.onChange(e)
                        if (mode === 'create' || (mode === 'edit' && !form.getValues('code'))) {
                          form.setValue('code', generateSlug(e.target.value), {shouldValidate: true})
                        }
                      }}
                    />
                  </FieldContent>
                  <FieldError errors={[fieldState.error]}/>
                </Field>
              )}
            />

            <Controller
              name="code"
              control={form.control}
              render={({field, fieldState}) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldContent>
                    <FieldLabel htmlFor="code">Code</FieldLabel>
                    <Input id="code" placeholder="eng" {...field}/>
                  </FieldContent>
                  <FieldError errors={[fieldState.error]}/>
                </Field>
              )}
            />

            <Controller
              name="description"
              control={form.control}
              render={({field, fieldState}) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldContent>
                    <FieldLabel htmlFor="description">Description</FieldLabel>
                    <Textarea
                      id="description"
                      placeholder="Optional description..."
                      rows={2}
                      {...field}
                    />
                  </FieldContent>
                  <FieldError errors={[fieldState.error]}/>
                </Field>
              )}
            />

            <Controller
              name="parent_unit_id"
              control={form.control}
              render={({field, fieldState}) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldContent>
                    <FieldLabel>Parent Unit</FieldLabel>
                    <div className="border rounded-lg p-2">
                      <ScrollArea className="h-48">
                        <RadioGroup value={field.value || ''} onValueChange={(val) => field.onChange(val || undefined)}>
                          <div className="space-y-1">
                            <button
                              type="button"
                              onClick={() => field.onChange(undefined)}
                              className={cn(
                                'flex items-center gap-2 w-full text-left py-1 px-2 rounded text-sm',
                                field.value === undefined && 'bg-primary text-primary-foreground'
                              )}
                            >
                              <RadioGroupItem value="" id="root"/>
                              <Building2 className="h-3.5 w-3.5"/>
                              <span>No Parent (Root)</span>
                            </button>
                            {tree.map((node) => (
                              <TreeNodeSelector
                                key={node.unit.id}
                                node={node}
                                selectedId={field.value}
                                disabledIds={disabledIds}
                                onSelect={field.onChange}
                              />
                            ))}
                          </div>
                        </RadioGroup>
                      </ScrollArea>
                    </div>
                  </FieldContent>
                  <FieldError errors={[fieldState.error]}/>
                </Field>
              )}
            />
          </div>

          <DialogFooter className="pt-4">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit">{mode === 'create' ? 'Create' : 'Save'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}


const TreeNodeSelector = memo(
  function TreeNodeSelector(
    {
      node,
      selectedId,
      disabledIds,
      onSelect,
    }: {
      node: UnitTreeNode
      selectedId: string | undefined
      disabledIds: Set<string>
      onSelect: (id: string) => void
    }) {
    const [expanded, setExpanded] = useState(true)
    const isDisabled = disabledIds.has(node.unit.id)
    const isSelected = selectedId === node.unit.id


    return (
      <div className="ml-2">
        <button
          type="button"
          onClick={() => !isDisabled && onSelect(node.unit.id)}
          disabled={isDisabled}
          className={cn(
            'flex items-center gap-1 w-full text-left py-1 px-2 rounded text-sm',
            isSelected && 'bg-primary text-primary-foreground',
            !isSelected && !isDisabled && 'hover:bg-muted',
            isDisabled && 'opacity-40 cursor-not-allowed'
          )}
        >
          {node.children && node.children.length > 0 ? (
            <span
              onClick={(e) => {
                e.stopPropagation()
                setExpanded(!expanded)
              }}
              className="cursor-pointer p-0.5"
            >
            {expanded ? <ChevronDown className="h-3.5 w-3.5"/> : <ChevronRight className="h-3.5 w-3.5"/>}
          </span>
          ) : (
            <span className="w-5"/>
          )}
          <Building2 className="h-3.5 w-3.5 shrink-0"/>
          <span className="truncate">{node.unit.name}</span>
          <span className="text-xs opacity-60 ml-1 shrink-0">{node.unit.code}</span>
        </button>
        {expanded && node.children && node.children.length > 0 && (
          <div>
            {node.children.map((child) => (
              <TreeNodeSelector
                key={child.unit.id}
                node={child}
                selectedId={selectedId}
                disabledIds={disabledIds}
                onSelect={onSelect}
              />
            ))}
          </div>
        )}
      </div>
    )
  }
)
