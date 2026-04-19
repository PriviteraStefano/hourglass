import {useForm} from 'react-hook-form'
import {zodResolver} from '@hookform/resolvers/zod'
import {z} from 'zod'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from '@/components/ui/card'
import {Link, useNavigate} from '@tanstack/react-router'
import {useMutation} from "@tanstack/react-query";
import {AuthApis} from "@/api/auth.ts";
import {toast} from "sonner";

const bootstrapSchema = z.object({
  org_name: z.string().min(2, 'Organization name must be at least 2 characters'),
  firstname: z.string().min(1, 'First name is required'),
  lastname: z.string().min(1, 'Last name is required'),
  username: z.string().min(3, 'Username must be at least 3 characters').regex(/^[a-zA-Z0-9_]+$/, 'Username can only contain letters, numbers, and underscores'),
  email: z.string().email('Invalid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
})

type BootstrapFormData = z.infer<typeof bootstrapSchema>

export function BootstrapOrgForm() {
  const navigate = useNavigate()
  const {mutateAsync: bootstrapAsync, isError, error, isPending} = useMutation(AuthApis.bootstrapMutationOpts)

  const form = useForm<BootstrapFormData>({
    resolver: zodResolver(bootstrapSchema),
    defaultValues: {
      org_name: '',
      firstname: '',
      lastname: '',
      username: '',
      email: '',
      password: '',
    },
  })

  const onSubmit = (data: BootstrapFormData) => {
    toast.promise(
      bootstrapAsync(data),
      {
        loading: 'Creating organization and account...',
        success: () => {
          navigate({to: '/', replace: true})
          return 'Organization created! Redirecting to dashboard...'
        },
        error: (err) => err?.message ?? 'Failed to create organization',
      }
    )
  }

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle>Create Organization</CardTitle>
        <CardDescription>
          Start by creating your organization and admin account
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium" htmlFor="org_name">
              Organization Name
            </label>
            <Input
              id="org_name"
              type="text"
              placeholder="Acme Corp"
              {...form.register('org_name')}
            />
            {form.formState.errors.org_name && (
              <p className="text-sm text-destructive">
                {form.formState.errors.org_name.message}
              </p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium" htmlFor="firstname">
                First Name
              </label>
              <Input
                id="firstname"
                type="text"
                placeholder="John"
                {...form.register('firstname')}
              />
              {form.formState.errors.firstname && (
                <p className="text-sm text-destructive">
                  {form.formState.errors.firstname.message}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium" htmlFor="lastname">
                Last Name
              </label>
              <Input
                id="lastname"
                type="text"
                placeholder="Doe"
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
            <label className="text-sm font-medium" htmlFor="username">
              Username
            </label>
            <Input
              id="username"
              type="text"
              placeholder="johndoe"
              autoComplete="username"
              {...form.register('username')}
            />
            {form.formState.errors.username && (
              <p className="text-sm text-destructive">
                {form.formState.errors.username.message}
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
              placeholder="you@example.com"
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
              {error?.message || 'Failed to create organization'}
            </p>
          )}

          <Button
            type="submit"
            className="w-full"
            disabled={isPending}
          >
            {isPending ? 'Creating...' : 'Create Organization'}
          </Button>

          <p className="text-sm text-center text-muted-foreground">
            Already have an account?{' '}
            <Link to="/login" className="text-primary underline">
              Log in
            </Link>
          </p>
        </form>
      </CardContent>
    </Card>
  )
}
