import {Controller, useForm, useWatch} from 'react-hook-form'
import {zodResolver} from '@hookform/resolvers/zod'
import {z} from 'zod'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from '@/components/ui/card'
import {Link, useNavigate} from '@tanstack/react-router'
import {useMutation} from '@tanstack/react-query'
import {AuthApis} from '@/api/auth.ts'
import {toast} from 'sonner'
import {Field, FieldContent, FieldDescription, FieldError, FieldLabel} from '@/components/ui/field.tsx'
import {Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select.tsx'

const registerSchema = z.object({
  org_selection: z.enum(['create', 'join']),
  organization_name: z.string().min(2, 'Organization name must be at least 2 characters').optional(),
  invite_code: z.string().min(6, 'Invite code must be at least 6 characters').optional(),
  firstname: z.string().min(1, 'First name is required'),
  lastname: z.string().min(1, 'Last name is required'),
  username: z.string().min(3, 'Username must be at least 3 characters').regex(/^[a-zA-Z0-9_]+$/, 'Username can only contain letters, numbers, and underscores'),
  email: z.string().email('Invalid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
})
  .refine((data) => !(data.org_selection === 'create' && !data.organization_name), {
    message: 'Organization name is required',
    path: ['organization_name'],
  })
  .refine((data) => !(data.org_selection === 'join' && !data.invite_code), {
    message: 'Invite code is required',
    path: ['invite_code'],
  })

type RegisterFormData = z.infer<typeof registerSchema>

export function RegisterForm() {
  const navigate = useNavigate()
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

  const orgSelection = useWatch({control: form.control, name: 'org_selection'})
  const isCreating = orgSelection === 'create'

  const onSubmit = (data: RegisterFormData) => {
    const payload = {
      firstname: data.firstname,
      lastname: data.lastname,
      username: data.username,
      email: data.email,
      password: data.password,
      ...(data.org_selection === 'create' ? {organization_name: data.organization_name} : {invite_code: data.invite_code}),
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
          <Controller
            name="org_selection"
            control={form.control}
            render={({field, fieldState}) => (
              <Field data-invalid={fieldState.invalid}>
                <FieldContent>
                  <FieldLabel htmlFor={field.name}>I want to</FieldLabel>
                  <FieldDescription>Create a new organization or join one with an invite code.</FieldDescription>
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                  >
                    <SelectTrigger className={"w-full"} id={field.name}>
                      <SelectValue placeholder="Select an option" className={"capitalize"}/>
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        <SelectItem value="create">Create</SelectItem>
                        <SelectItem value="join">Join</SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </FieldContent>
                <FieldError errors={[fieldState.error]}/>
              </Field>
            )}
          />

          {isCreating ? (
            <Controller
              name="organization_name"
              control={form.control}
              render={({field, fieldState}) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel htmlFor="organization_name">Organization Name</FieldLabel>
                  <Input
                    id="organization_name"
                    type="text"
                    placeholder="Acme Corp"
                    aria-label="Organization Name"
                    {...field}
                  />
                  <FieldError errors={[fieldState.error]}/>
                </Field>)}
            />
          ) : (
            <Controller
              name="invite_code"
              control={form.control}
              render={({field, fieldState}) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel htmlFor="invite_code">Invite Code</FieldLabel>
                  <Input
                    id="invite_code"
                    type="text"
                    placeholder="ABC123"
                    aria-label="Invite Code"
                    {...field}
                  />
                  <FieldError errors={[fieldState.error]}/>
                </Field>
              )}
            />
          )}

          <div className="grid grid-cols-2 gap-4">
            <Field data-invalid={!!form.formState.errors.firstname}>
              <FieldLabel htmlFor="firstname">First Name</FieldLabel>
              <Input
                id="firstname"
                type="text"
                placeholder="John"
                aria-label="First Name"
                {...form.register('firstname')}
              />
              <FieldError errors={[form.formState.errors.firstname]}/>
            </Field>

            <Field data-invalid={!!form.formState.errors.lastname}>
              <FieldLabel htmlFor="lastname">Last Name</FieldLabel>
              <Input
                id="lastname"
                type="text"
                placeholder="Doe"
                aria-label="Last Name"
                {...form.register('lastname')}
              />
              <FieldError errors={[form.formState.errors.lastname]}/>
            </Field>
          </div>

          <Field data-invalid={!!form.formState.errors.username}>
            <FieldLabel htmlFor="username">Username</FieldLabel>
            <Input
              id="username"
              type="text"
              placeholder="johndoe"
              autoComplete="username"
              aria-label="Username"
              {...form.register('username')}
            />
            <FieldError errors={[form.formState.errors.username]}/>
          </Field>

          <Field data-invalid={!!form.formState.errors.email}>
            <FieldLabel htmlFor="email">Email</FieldLabel>
            <Input
              id="email"
              type="email"
              placeholder="you@example.com"
              autoComplete="email"
              aria-label="Email"
              {...form.register('email')}
            />
            <FieldError errors={[form.formState.errors.email]}/>
          </Field>

          <Field data-invalid={!!form.formState.errors.password}>
            <FieldLabel htmlFor="password">Password</FieldLabel>
            <Input
              id="password"
              type="password"
              placeholder="••••••••"
              autoComplete="new-password"
              aria-label="Password"
              {...form.register('password')}
            />
            <FieldError errors={[form.formState.errors.password]}/>
          </Field>

          {
            isError && (
              <p className="text-sm text-destructive">
                {error?.message || 'Registration failed'}
              </p>
            )
          }

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