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

const verifyResetSchema = z.object({
  identifier: z.string().min(1, 'Email or username is required'),
  code: z.string().length(6, 'Code must be 6 digits'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
})

type VerifyResetFormData = z.infer<typeof verifyResetSchema>

export function PasswordResetVerifyForm() {
  const navigate = useNavigate()
  const {mutateAsync: verifyResetAsync, isError, error, isPending} = useMutation(AuthApis.verifyPasswordResetMutationOpts)

  const form = useForm<VerifyResetFormData>({
    resolver: zodResolver(verifyResetSchema),
    defaultValues: {
      identifier: '',
      code: '',
      password: '',
    },
  })

  const onSubmit = (data: VerifyResetFormData) => {
    toast.promise(
      verifyResetAsync(data),
      {
        loading: 'Resetting password...',
        success: () => {
          navigate({to: '/login', replace: true})
          return 'Password reset successful! You can now log in.'
        },
        error: (err) => err?.message ?? 'Failed to reset password',
      }
    )
  }

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle>Enter Reset Code</CardTitle>
        <CardDescription>
          Enter the 6-digit code from your email and your new password
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium" htmlFor="identifier">
              Email or Username
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
            <label className="text-sm font-medium" htmlFor="code">
              Reset Code
            </label>
            <Input
              id="code"
              type="text"
              placeholder="123456"
              maxLength={6}
              {...form.register('code')}
            />
            {form.formState.errors.code && (
              <p className="text-sm text-destructive">
                {form.formState.errors.code.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium" htmlFor="password">
              New Password
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
              {error?.message || 'Failed to reset password'}
            </p>
          )}

          <Button
            type="submit"
            className="w-full"
            disabled={isPending}
          >
            {isPending ? 'Resetting...' : 'Reset Password'}
          </Button>

          <p className="text-sm text-center text-muted-foreground">
            Remember your password?{' '}
            <Link to="/login" className="text-primary underline">
              Log in
            </Link>
          </p>
        </form>
      </CardContent>
    </Card>
  )
}
