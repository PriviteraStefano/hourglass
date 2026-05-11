import * as React from "react";
import {cn} from "@/lib/utils.ts";

export function Header({children, className}: { children?: React.ReactNode, className?: string }) {
  return (
    <header className={cn("flex h-12 shrink-0 items-center bg-sidebar gap-2 px-4", className)}>
      {children}
    </header>
  )
}