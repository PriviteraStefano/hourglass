import {createFileRoute} from '@tanstack/react-router'
import {TimeEntriesPage} from "@/src/routes/_authenticated/time-entries/-components/time-entries-page.tsx";

export const Route = createFileRoute('/_authenticated/time-entries/')({
  component: TimeEntriesPage
})


