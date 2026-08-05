"use client";

import { useTransition } from "react";
import { ArrowRight, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { PRIMARY_CTA } from "@/lib/styles";
import { signInAction } from "@/app/actions";

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
