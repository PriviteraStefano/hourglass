import {useForm} from 'react-hook-form'
import {zodResolver} from '@hookform/resolvers/zod'
import {z} from 'zod'
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from '@/components/ui/card'
import {useNavigate} from '@tanstack/react-router'
import {Button} from "@/components/ui/button.tsx";
import {Input} from "@/components/ui/input.tsx";
import {useMutation} from "@tanstack/react-query";
import {AuthApis} from "@/api/auth.ts";
import {toast} from "sonner";

const bootstrapSchema = z.object({
  organization_name: z.string().min(1, 'Organization name is required'),
  email: z.string().email('Invalid email address'),
  username: z.string().min(3, 'Username must be at least 3 characters').regex(/^[a-zA-Z0-9_]+$/, 'Username can only contain letters, numbers, and underscores'),
  firstname: z.string().min(1, 'First name is required'),
  lastname: z.string().min(1, 'Last name is required'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
})

type BootstrapFormData = z.infer<typeof bootstrapSchema>

export function BootstrapForm() {
  const {mutateAsync: bootstrapAsync, isError, error, isPending} = useMutation(AuthApis.bootstrapMutationOpts)
  const navigate = useNavigate()

  const form = useForm<BootstrapFormData>({
    resolver: zodResolver(bootstrapSchema),
    defaultValues: {
      organization_name: '',
      email: '',
      username: '',
      firstname: '',
      lastname: '',
      password: '',
    },
  })

  const onSubmit = (data: BootstrapFormData) => {
    toast.promise(
      bootstrapAsync(data),
      {
        loading: 'Setting up your organization...',
        success: () => {
          navigate({to: '/', replace: true})
          return 'Organization created! Redirecting...'
        },
        error: (err) => err?.message ?? 'Setup failed',
      }
    )
  }

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle>Setup Your Organization</CardTitle>
        <CardDescription>
          Create your organization and admin account to get started
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium" htmlFor="organization_name">
              Organization Name
            </label>
            <Input
              id="organization_name"
              type="text"
              placeholder="Acme Corp"
              autoComplete="organization"
              {...form.register('organization_name')}
            />
            {form.formState.errors.organization_name && (
              <p className="text-sm text-destructive">
                {form.formState.errors.organization_name.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium" htmlFor="email">
              Email
            </label>
            <Input
              id="email"
              type="email"
              placeholder="admin@example.com"
              autoComplete="email"
              {...form.register('email')}
            />
            {form.formState.errors.email && (
              <p className="text-sm text-destructive">
                {form.formState.errors.email.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium" htmlFor="username">
              Username
            </label>
            <Input
              id="username"
              type="text"
              placeholder="admin"
              autoComplete="username"
              {...form.register('username')}
            />
            {form.formState.errors.username && (
              <p className="text-sm text-destructive">
                {form.formState.errors.username.message}
              </p>
            )}
          </div>

          <div className="flex gap-4">
            <div className="space-y-2 flex-1">
              <label className="text-sm font-medium" htmlFor="firstname">
                First Name
              </label>
              <Input
                id="firstname"
                type="text"
                placeholder="John"
                autoComplete="given-name"
                {...form.register('firstname')}
              />
              {form.formState.errors.firstname && (
                <p className="text-sm text-destructive">
                  {form.formState.errors.firstname.message}
                </p>
              )}
            </div>

            <div className="space-y-2 flex-1">
              <label className="text-sm font-medium" htmlFor="lastname">
                Last Name
              </label>
              <Input
                id="lastname"
                type="text"
                placeholder="Doe"
                autoComplete="family-name"
                {...form.register('lastname')}
              />
              {form.formState.errors.lastname && (
                <p className="text-sm text-destructive">
                  {form.formState.errors.lastname.message}
                </p>
              )}
            </div>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium" htmlFor="password">
              Password
            </label>
            <Input
              id="password"
              type="password"
              placeholder="••••••••"
              autoComplete="new-password"
              {...form.register('password')}
            />
            {form.formState.errors.password && (
              <p className="text-sm text-destructive">
                {form.formState.errors.password.message}
              </p>
            )}
          </div>

          {isError && (
            <p className="text-sm text-destructive">
              {error?.message || 'Setup failed'}
            </p>
          )}

          <Button
            type="submit"
            className="w-full"
            disabled={isPending}
          >
            {isPending ? 'Setting up...' : 'Create Organization'}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}