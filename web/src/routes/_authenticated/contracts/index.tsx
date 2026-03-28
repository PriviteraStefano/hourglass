import {createFileRoute} from '@tanstack/react-router'
import {ContractList, tabsSchema} from './-components/contract-list'
import {z} from "zod";

export const Route = createFileRoute('/_authenticated/contracts/')({
  component: ContractList,
  validateSearch: z.object({
    tab: tabsSchema.optional().default('owned'),
    searchQuery: z.string().optional().default(''),
  })
})