import {useForm} from 'react-hook-form'
import {zodResolver} from '@hookform/resolvers/zod'
import {z} from 'zod'
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from '@/components/ui/card'
import {Link, useNavigate} from '@tanstack/react-router'
import {Button} from "@/components/ui/button.tsx";
import {Input} from "@/components/ui/input.tsx";
import {useMutation} from "@tanstack/react-query";
import {AuthApis} from "@/api/auth.ts";
import {toast} from "sonner";

const loginSchema = z.object({
  identifier: z.string().min(1, 'Username or email is required'),
  password: z.string().min(1, 'Password is required'),
})

type LoginFormData = z.infer<typeof loginSchema>

export function LoginForm() {
  const {mutateAsync: loginAsync, isError, error, isPending} = useMutation(AuthApis.loginMutationOpts)
  const navigate = useNavigate()

  const form = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      identifier: '',
      password: '',
    },
  })

  const onSubmit = (data: LoginFormData) => {
    toast.promise(
      loginAsync(data),
      {
        loading: 'Logging in...',
        success: () => {
          navigate({to: '/', replace: true})
          return 'Authentication successful! Redirecting to dashboard...'
        },
        error: (err) => err?.message ?? 'Authentication failed',
      }
    )
  }

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle>Log in</CardTitle>
        <CardDescription>
          Enter your credentials to access your account
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium" htmlFor="identifier">
              Username or Email
            </label>
            <Input
              id="identifier"
              type="text"
              placeholder="you@example.com or username"
              autoComplete="username"
              {...form.register('identifier')}
            />
            {form.formState.errors.identifier && (
              <p className="text-sm text-destructive">
                {form.formState.errors.identifier.message}
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
              {error?.message || 'Invalid credentials'}
            </p>
          )}

          <Button
            type="submit"
            className="w-full"
            disabled={isPending}
          >
            {isPending ? 'Logging in...' : 'Log in'}
          </Button>

          <p className="text-sm text-center text-muted-foreground">
            Don't have an account?{' '}
            <Link to="/register" className="text-primary underline">
              Register
            </Link>
          </p>
        </form>
      </CardContent>
    </Card>
  )
}