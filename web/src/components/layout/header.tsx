import {LogOutIcon, UserIcon} from 'lucide-react'
import {Button} from '@/src/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/src/components/ui/dropdown-menu'
import {Avatar, AvatarFallback} from '@/src/components/ui/avatar'
import {useMutation, useSuspenseQuery} from "@tanstack/react-query";
import {AuthApis} from "@/src/api/auth.ts";

export function Header() {
  const {data: user} = useSuspenseQuery(AuthApis.profileQueryOpts)
  const {mutate: logout} = useMutation(AuthApis.logoutMutationOpts)

  const initials = user?.name
    ?.split(' ')
    .map((n: string) => n[0])
    .join('')
    .toUpperCase() ?? 'U'

  return (
    <div className="flex flex-1 items-center justify-between">
      <div className="flex items-center gap-3">
        <h1 className="text-lg font-semibold">Hourglass</h1>
        {user?.organization && (
          <>
            <span className="text-muted-foreground">|</span>
            <span className="text-sm text-muted-foreground">
              {user.organization.name}
            </span>
          </>
        )}
      </div>

      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button variant="ghost" className="gap-2">
              <Avatar className="h-7 w-7">
                <AvatarFallback className="text-xs">{initials}</AvatarFallback>
              </Avatar>
              <span className="text-sm">{user?.name}</span>
            </Button>
          }
        />

        <DropdownMenuContent align="end" className="w-48">
          <DropdownMenuLabel>
            <div className="flex flex-col">
              <span>{user?.name}</span>
              <span className="text-xs font-normal text-muted-foreground">
                {user?.email}
              </span>
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator/>
          <DropdownMenuItem className="gap-2">
            <UserIcon className="w-4 h-4"/>
            Profile
          </DropdownMenuItem>
          <DropdownMenuSeparator/>
          <DropdownMenuItem
            className="gap-2 text-destructive"
            onClick={() => logout()}
          >
            <LogOutIcon className="w-4 h-4"/>
            Log out
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}
