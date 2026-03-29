import {LogOutIcon, UserIcon} from 'lucide-react'
import {Button} from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {Avatar, AvatarFallback} from '@/components/ui/avatar'
import {useMutation, useSuspenseQuery} from "@tanstack/react-query";
import {AuthApis} from "@/api/auth.ts";

export function ProfileMenu() {
  const {data: {user}} = useSuspenseQuery(AuthApis.profileQueryOpts)
  const {mutate: logout} = useMutation(AuthApis.logoutMutationOpts)

  const initials = user?.name
    ?.split(' ')
    .map((n: string) => n[0])
    .join('')
    .toUpperCase() ?? 'U'
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button variant="ghost" size={"icon-lg"} className="rounded-full">
            <Avatar className="h-7 w-7">
              <AvatarFallback className="text-xs">{initials}</AvatarFallback>
            </Avatar>
            {/*<span className="text-sm">{user?.name}</span>*/}
          </Button>
        }
      />

      <DropdownMenuContent align="end" className="w-48">
        <DropdownMenuGroup>
          <DropdownMenuLabel>
            <div className="flex flex-col">
              <span className={"font-semibold"}>{user?.name}</span>
              <span className="text-xs font-normal">
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
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}