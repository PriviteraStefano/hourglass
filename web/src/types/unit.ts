import {z} from 'zod'

export const UnitSchema = z.object({
  id: z.string(),
  org_id: z.uuid(),
  name: z.string(),
  description: z.string().nullable(),
  parent_unit_id: z.string().optional(),
  hierarchy_level: z.number(),
  code: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
})

export type Unit = z.infer<typeof UnitSchema>

export const UnitTreeNodeSchema = z.object({
    unit: UnitSchema,
    member_count: z.number(),
    total_member_count: z.number(),
    get children() {
      return z.array(UnitTreeNodeSchema).optional()
    }
  }
)

export type UnitTreeNode = z.infer<typeof UnitTreeNodeSchema>

export const UnitMemberSchema = z.object({
  id: z.string(),
  org_id: z.uuid(),
  user_id: z.uuid(),
  user_name: z.string(),
  user_email: z.email(),
  unit_id: z.string(),
  is_primary: z.boolean(),
  role: z.string(),
  start_date: z.string(),
  end_date: z.string().nullable().optional(),
  created_at: z.string(),
})

export type UnitMember = z.infer<typeof UnitMemberSchema>

export const CreateUnitRequestSchema = z.object({
  name: z.string().min(1).max(255),
  description: z.string().nullable(),
  parent_unit_id: z.string().nullable(),
  code: z.string().min(1).max(50),
})

export type CreateUnitRequest = z.infer<typeof CreateUnitRequestSchema>

export const UpdateUnitRequestSchema = z.object({
  name: z.string().min(1).max(255),
  description: z.string().nullable(),
  code: z.string().min(1).max(50),
  parent_unit_id: z.string().nullable().optional(),
})

export type UpdateUnitRequest = z.infer<typeof UpdateUnitRequestSchema>

export const AddUnitMemberRequestSchema = z.object({
  user_id: z.uuid(),
  role: z.string().min(1),
  is_primary: z.boolean(),
})

export type AddUnitMemberRequest = z.infer<typeof AddUnitMemberRequestSchema>

export const UpdateUnitMemberRequestSchema = z.object({
  is_primary: z.boolean(),
  end_date: z.string().nullable().optional(),
})

export type UpdateUnitMemberRequest = z.infer<typeof UpdateUnitMemberRequestSchema>
