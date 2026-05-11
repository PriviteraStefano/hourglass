import {cn} from "@/lib/utils.ts";
import {ReactNode} from "react";

export function Body({children, className}: { children: ReactNode, className?: string }) {
  return (
    <div
      className={cn("flex-1 rounded-lg bg-background overflow-clip shadow-lg ring-1 ring-sidebar-border m-0.5 ", className)}>
      {children}
    </div>
  )
}