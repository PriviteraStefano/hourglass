import {createFileRoute} from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/')({
  component: () => {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-semibold">Dashboard</h1>
        <p className="text-muted-foreground">
          Welcome! Select a section from the sidebar to get started.
        </p>
      </div>
    )
  },
})