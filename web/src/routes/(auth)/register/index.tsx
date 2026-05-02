import {createFileRoute} from '@tanstack/react-router'
import {RegisterForm} from "@/routes/(auth)/register/-components/register-form.tsx";

export const Route = createFileRoute('/(auth)/register/')({
  component: () => (
    <div className="min-h-screen flex items-center justify-center bg-muted/30">
      <RegisterForm />
    </div>
  ),
})
