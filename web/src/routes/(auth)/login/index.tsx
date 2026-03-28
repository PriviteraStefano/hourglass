import {createFileRoute} from '@tanstack/react-router'
import {LoginForm} from "@/routes/(auth)/login/-components/login-form.tsx";

export const Route = createFileRoute('/(auth)/login/')({
  component: () => (
    <div className="min-h-screen flex items-center justify-center bg-muted/30">
      <LoginForm />
    </div>
  ),
})