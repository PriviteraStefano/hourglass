import {useForm} from 'react-hook-form'
import {zodResolver} from '@hookform/resolvers/zod'
import {z} from 'zod'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from '@/components/ui/card'
import {useSearchParams, useNavigate} from '@tanstack/react-router'
import {useQuery, useMutation} from "@tanstack/react-query";
import {useEffect} from "react";
import {AuthApis} from "@/api/auth.ts";
import {toast} from "sonner";

const acceptInviteSchema = z.object({
  email: z.string().email('Invalid email address'),
  username: z.string().min(3, 'Username must be at least 3 characters').regex(/^[a-zA-Z0-9_]+$/, 'Username can only contain letters, numbers, and underscores'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
})

type AcceptInviteFormData = z.infer<typeof acceptInviteSchema>

export function InvitationAcceptForm() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const token = searchParams.get('token')
  const code = searchParams.get('code')

  const {data: invitation, isLoading: loadingInvitation} = useQuery({
    ...(token ? AuthApis.validateInvitationTokenQueryOpts(token) : AuthApis.validateInvitationCodeQueryOpts(code || '')),
    enabled: !!token || !!code,
  })

  const {mutateAsync: acceptAsync, isError, error, isPending} = useMutation(AuthApis.acceptInvitationMutationOpts)

  const form = useForm<AcceptInviteFormData>({
    resolver: zodResolver(acceptInviteSchema),
    defaultValues: {
      email: '',
      username: '',
      password: '',
    },
  })

  useEffect(() => {
    if (invitation?.email) {
      form.reset({ ...form.getValues(), email: invitation.email || '' })
    }
  }, [invitation?.email])

  const onSubmit = (data: AcceptInviteFormData) => {
    if (!token && !code) {
      toast.error('Invalid invitation')
      return
    }

    toast.promise(
      acceptAsync({token: token || code || '', ...data}),
      {
        loading: 'Accepting invitation...',
        success: () => {
          navigate({to: '/login', replace: true})
          return 'Invitation accepted! You can now log in.'
        },
        error: (err) => err?.message ?? 'Failed to accept invitation',
      }
    )
  }

  if (loadingInvitation) {
    return (
      <Card className="w-full max-w-md">
        <CardContent className="pt-6">
          <p className="text-center">Loading invitation...</p>
        </CardContent>
      </Card>
    )
  }

  if (!invitation) {
    return (
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Invalid Invitation</CardTitle>
          <CardDescription>
            This invitation link is invalid or has expired.
          </CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle>Accept Invitation</CardTitle>
        <CardDescription>
          You've been invited to join an organization. Create your account to get started.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
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
              {error?.message || 'Failed to accept invitation'}
            </p>
          )}

          <Button
            type="submit"
            className="w-full"
            disabled={isPending}
          >
            {isPending ? 'Accepting...' : 'Accept Invitation'}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
