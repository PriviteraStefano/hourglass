import { createFileRoute } from '@tanstack/react-router'
import { ExportsPage } from './-components/exports-page'

export const Route = createFileRoute('/_authenticated/exports/')({
  component: ExportsPage,
})
