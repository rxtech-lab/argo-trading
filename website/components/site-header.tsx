import Link from "next/link";
import { Activity, ArrowUpRight } from "lucide-react";
import { SignInButton } from "@/components/auth-buttons";
import { UserMenu } from "@/components/user-menu";
import type { AppUser } from "@/lib/session";

export function SiteHeader({
  user,
  isAdmin,
}: {
  user: AppUser | null;
  isAdmin: boolean;
}) {
  return (
    <header className="sticky top-0 z-50 w-full border-b border-white/8 bg-background/70 backdrop-blur-xl">
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-5 md:px-8">
        <Link href="/" className="flex items-center gap-2.5">
          <span className="flex size-8 items-center justify-center rounded-lg bg-brand text-[#05130c]">
            <Activity className="size-4.5" strokeWidth={2.75} />
          </span>
          <span className="font-display text-base font-semibold tracking-tight">
            RxArgo
          </span>
        </Link>

        <div className="flex items-center gap-5">
          <a
            href="https://finance.bots.rxlab.app/"
            target="_blank"
            rel="noopener noreferrer"
            className="hidden items-center gap-1 text-sm text-muted-fg transition-colors hover:text-foreground sm:inline-flex"
          >
            Research desk
            <ArrowUpRight className="size-3.5" strokeWidth={2.25} />
          </a>
          {user ? (
            <>
              {isAdmin && (
                <Link
                  href="/admin"
                  className="text-sm text-muted-fg transition-colors hover:text-foreground"
                >
                  Admin
                </Link>
              )}
              <UserMenu user={user} />
            </>
          ) : (
            <SignInButton compact label="Sign in" />
          )}
        </div>
      </div>
    </header>
  );
}
