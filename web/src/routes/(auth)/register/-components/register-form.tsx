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

const registerSchema = z.object({
  name: z.string().min(2, 'Name must be at least 2 characters'),
  username: z.string().min(3, 'Username must be at least 3 characters').regex(/^[a-zA-Z0-9_]+$/, 'Username can only contain letters, numbers, and underscores').optional().or(z.literal('')),
  email: z.string().email('Invalid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
  organization_name: z.string().min(2, 'Organization name must be at least 2 characters').optional(),
})

type RegisterFormData = z.infer<typeof registerSchema>

export function RegisterForm() {
  const navigate = useNavigate()
  const {mutateAsync: registerAsync, isError, error, isPending} = useMutation(AuthApis.registerMutationOpts)

  const form = useForm<RegisterFormData>({
    resolver: zodResolver(registerSchema),
    defaultValues: {
      name: '',
      username: '',
      email: '',
      password: '',
      organization_name: '',
    },
  })

  const onSubmit = (data: RegisterFormData) => {
    toast.promise(
      registerAsync(data),
      {
        loading: 'Creating account...',
        success: () => {
          navigate({to: '/', replace: true})
          return 'Account created successfully! Redirecting to dashboard...'
        },
        error: (err) => err?.message ?? 'Registration failed',
      }
    )
  }

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle>Create an account</CardTitle>
        <CardDescription>
          Register a new organization and become its admin
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium" htmlFor="name">
              Your Name
            </label>
            <Input
              id="name"
              type="text"
              placeholder="John Doe"
              {...form.register('name')}
            />
            {form.formState.errors.name && (
              <p className="text-sm text-destructive">
                {form.formState.errors.name.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium" htmlFor="username">
              Username (optional)
            </label>
            <Input
              id="username"
              type="text"
              placeholder="johndoe"
              {...form.register('username')}
            />
            {form.formState.errors.username && (
              <p className="text-sm text-destructive">
                {form.formState.errors.username.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium" htmlFor="organization_name">
              Organization Name
            </label>
            <Input
              id="organization_name"
              type="text"
              placeholder="Acme Corp"
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
              placeholder="you@example.com"
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
              {error?.message || 'Registration failed'}
            </p>
          )}

          <Button
            type="submit"
            className="w-full"
            disabled={isPending}
          >
            {isPending ? 'Creating account...' : 'Create account'}
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