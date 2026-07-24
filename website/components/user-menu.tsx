"use client";

import { useTransition } from "react";
import { Menu } from "@base-ui/react/menu";
import { LogOut, Loader2 } from "lucide-react";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { signOutAction } from "@/app/actions";
import type { AppUser } from "@/lib/session";

function initials(name: string | null, email: string): string {
  const src = (name || email).trim();
  const parts = src.split(/\s+/).filter(Boolean);
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return src.slice(0, 2).toUpperCase();
}

export function UserMenu({ user }: { user: AppUser }) {
  const [pending, start] = useTransition();

  return (
    <Menu.Root>
      <Menu.Trigger
        className="flex items-center rounded-full outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-brand/60 cursor-pointer"
        aria-label="Account menu"
      >
        <Avatar>
          <AvatarFallback className="bg-gradient-to-br from-brand/25 to-brand-2/20 font-mono text-[11px] font-semibold text-brand">
            {initials(user.name, user.email)}
          </AvatarFallback>
        </Avatar>
      </Menu.Trigger>
      <Menu.Portal>
        <Menu.Positioner sideOffset={8} align="end" className="z-50">
          <Menu.Popup className="glass min-w-56 origin-[var(--transform-origin)] overflow-hidden rounded-2xl p-1.5 shadow-xl outline-none">
            <div className="flex items-center gap-3 px-3 py-2.5">
              <Avatar>
                <AvatarFallback className="bg-gradient-to-br from-brand/25 to-brand-2/20 font-mono text-[11px] font-semibold text-brand">
                  {initials(user.name, user.email)}
                </AvatarFallback>
              </Avatar>
              <div className="min-w-0">
                {user.name && (
                  <p className="truncate text-sm font-medium">{user.name}</p>
                )}
                <p className="truncate font-mono text-xs text-muted-fg">
                  {user.email}
                </p>
              </div>
            </div>
            <div className="my-1 h-px bg-white/8" />
            <Menu.Item
              closeOnClick={false}
              disabled={pending}
              onClick={() => start(() => signOutAction())}
              className="flex cursor-pointer items-center gap-2.5 rounded-xl px-3 py-2 text-sm text-muted-fg outline-none transition-colors data-[highlighted]:bg-white/5 data-[highlighted]:text-foreground data-[disabled]:opacity-50"
            >
              {pending ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <LogOut className="size-4" />
              )}
              Sign out
            </Menu.Item>
          </Menu.Popup>
        </Menu.Positioner>
      </Menu.Portal>
    </Menu.Root>
  );
}
