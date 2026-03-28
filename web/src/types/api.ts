import type {UserWithMembership} from "@/src/types/models.ts";
// export function createApiResponseSchema<T extends z.ZodTypeAny>(t: T) {
//   return (
//     z.discriminatedUnion(
//       "status",
//       [
//         z.object({
//           status: z.number().min(200).max(299),
//           data: t,
//         }),
//         z.object({
//           status: z.number().min(400).max(599),
//           error: z.string(),
//         })
//       ]
//     )
//   )
// }
//
// export type ApiResponse<T extends z.ZodTypeAny> = ReturnType<typeof createApiResponseSchema<T>>


export interface ApiResponse<T> {
  data: T;
}

export interface AuthResponse {
  user: UserWithMembership
  token: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  email: string
  password: string
  name: string
  organization_name?: string
  invite_token?: string
}

export interface CreateTimeEntryRequest {
  date: string
  items: Array<{
    project_id: string
    hours: number
    description?: string
  }>
}

export interface UpdateTimeEntryRequest {
  items: Array<{
    project_id: string
    hours: number
    description?: string
  }>
}

export interface ApiError {
  message: string
  error?: string
}

export interface CreateContractRequest {
  name: string
  km_rate: number
  currency: string
  governance_model: 'creator_controlled' | 'unanimous' | 'majority'
  is_shared: boolean
}

export interface CreateProjectRequest {
  name: string
  type: 'billable' | 'internal'
  contract_id: string
  governance_model: 'creator_controlled' | 'unanimous' | 'majority'
  is_shared: boolean
}