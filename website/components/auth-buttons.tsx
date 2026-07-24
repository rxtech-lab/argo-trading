"use client";

import { useTransition } from "react";
import { ArrowRight, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { signInAction } from "@/app/actions";

export const PRIMARY_CTA =
  "inline-flex items-center justify-center gap-2 rounded-full bg-brand px-6 py-3 text-sm font-semibold text-[#05130c] shadow-lg shadow-brand/25 transition-all hover:brightness-110 hover:shadow-brand/40 active:scale-[0.98] disabled:pointer-events-none disabled:opacity-60 cursor-pointer";

export function SignInButton({
  className,
  label = "Sign in with RxLab",
  compact = false,
}: {
  className?: string;
  label?: string;
  compact?: boolean;
}) {
  const [pending, start] = useTransition();
  return (
    <button
      className={cn(PRIMARY_CTA, compact && "px-4 py-2 text-xs", className)}
      disabled={pending}
      onClick={() => start(() => signInAction())}
    >
      {label}
      {pending ? (
        <Loader2 className="size-4 animate-spin" />
      ) : (
        <ArrowRight className="size-4" />
      )}
    </button>
  );
}
