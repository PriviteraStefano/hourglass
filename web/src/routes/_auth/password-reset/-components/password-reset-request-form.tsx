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

const requestResetSchema = z.object({
  identifier: z.string().min(1, 'Email or username is required'),
})

type RequestResetFormData = z.infer<typeof requestResetSchema>

export function PasswordResetRequestForm() {
  const navigate = useNavigate()
  const {mutateAsync: requestResetAsync, isError, error, isPending, isSuccess} = useMutation(AuthApis.requestPasswordResetMutationOpts)

  const form = useForm<RequestResetFormData>({
    resolver: zodResolver(requestResetSchema),
    defaultValues: {
      identifier: '',
    },
  })

  const onSubmit = (data: RequestResetFormData) => {
    toast.promise(
      requestResetAsync(data),
      {
        loading: 'Sending reset code...',
        success: () => {
          return 'Reset code sent! Check your email.'
        },
        error: (err) => err?.message ?? 'Failed to send reset code',
      }
    )
  }

  if (isSuccess) {
    return (
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Check your email</CardTitle>
          <CardDescription>
            We sent a reset code to your email. Enter it below along with your new password.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Link to="/password-reset/verify" className="text-primary underline">
            Enter reset code
          </Link>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle>Reset Password</CardTitle>
        <CardDescription>
          Enter your email or username and we'll send you a reset code
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

          {isError && (
            <p className="text-sm text-destructive">
              {error?.message || 'Failed to send reset code'}
            </p>
          )}

          <Button
            type="submit"
            className="w-full"
            disabled={isPending}
          >
            {isPending ? 'Sending...' : 'Send Reset Code'}
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
