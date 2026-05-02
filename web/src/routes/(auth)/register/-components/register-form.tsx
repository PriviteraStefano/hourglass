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
import {useState} from "react";

const registerSchema = z.object({
  org_selection: z.enum(['create', 'join']),
  organization_name: z.string().min(2, 'Organization name must be at least 2 characters').optional(),
  invite_code: z.string().min(6, 'Invite code must be at least 6 characters').optional(),
  firstname: z.string().min(1, 'First name is required'),
  lastname: z.string().min(1, 'Last name is required'),
  username: z.string().min(3, 'Username must be at least 3 characters').regex(/^[a-zA-Z0-9_]+$/, 'Username can only contain letters, numbers, and underscores'),
  email: z.string().email('Invalid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
}).refine((data) => {
  if (data.org_selection === 'create' && !data.organization_name) return false
  if (data.org_selection === 'join' && !data.invite_code) return false
  return true
}, {
  message: 'Organization name or invite code is required',
  path: ['organization_name'],
})

type RegisterFormData = z.infer<typeof registerSchema>

export function RegisterForm() {
  const navigate = useNavigate()
  const [orgSelection, setOrgSelection] = useState<'create' | 'join'>('create')
  const {mutateAsync: registerAsync, isError, error, isPending} = useMutation(AuthApis.registerMutationOpts)

  const form = useForm<RegisterFormData>({
    resolver: zodResolver(registerSchema),
    defaultValues: {
      org_selection: 'create',
      organization_name: '',
      invite_code: '',
      firstname: '',
      lastname: '',
      username: '',
      email: '',
      password: '',
    },
  })

  const onSubmit = (data: RegisterFormData) => {
    const payload = {
      firstname: data.firstname,
      lastname: data.lastname,
      username: data.username,
      email: data.email,
      password: data.password,
      ...(data.org_selection === 'create' ? { organization_name: data.organization_name } : { invite_code: data.invite_code }),
    }

    toast.promise(
      registerAsync(payload),
      {
        loading: 'Creating account...',
        success: () => {
          navigate({to: '/', replace: true})
          return 'Account created! Redirecting to dashboard...'
        },
        error: (err) => err?.message ?? 'Registration failed',
      }
    )
  }

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle>Create Account</CardTitle>
        <CardDescription>
          Register as a new user
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">I want to</label>
            <div className="flex gap-4">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  value="create"
                  checked={orgSelection === 'create'}
                  onChange={() => {
                    setOrgSelection('create')
                    form.setValue('org_selection', 'create')
                  }}
                  className="accent-primary"
                />
                <span className="text-sm">Create a new organization</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  value="join"
                  checked={orgSelection === 'join'}
                  onChange={() => {
                    setOrgSelection('join')
                    form.setValue('org_selection', 'join')
                  }}
                  className="accent-primary"
                />
                <span className="text-sm">Join an organization</span>
              </label>
            </div>
          </div>

          {orgSelection === 'create' ? (
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
          ) : (
            <div className="space-y-2">
              <label className="text-sm font-medium" htmlFor="invite_code">
                Invite Code
              </label>
              <Input
                id="invite_code"
                type="text"
                placeholder="ABC123"
                {...form.register('invite_code')}
              />
              {form.formState.errors.invite_code && (
                <p className="text-sm text-destructive">
                  {form.formState.errors.invite_code.message}
                </p>
              )}
            </div>
          )}

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
              {error?.message || 'Registration failed'}
            </p>
          )}

          <Button
            type="submit"
            className="w-full"
            disabled={isPending}
          >
            {isPending ? 'Creating account...' : 'Create Account'}
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