"use client";

import { useTransition } from "react";
import { useRouter } from "next/navigation";
import { Check, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { approveAction } from "@/app/actions";

export function ApproveButton({
  userId,
  name,
}: {
  userId: string;
  name: string;
}) {
  const [pending, start] = useTransition();
  const router = useRouter();

  function approve() {
    if (!window.confirm(`Approve ${name} for access?`)) return;
    start(async () => {
      try {
        await approveAction(userId);
        toast.success("Approved");
        router.refresh();
      } catch {
        toast.error("Could not approve. Are you an admin?");
      }
    });
  }

  return (
    <button
      disabled={pending}
      className="inline-flex items-center gap-1.5 rounded-full bg-brand px-4 py-1.5 text-xs font-semibold text-[#05130c] transition-all hover:brightness-110 active:scale-[0.98] disabled:opacity-50 cursor-pointer"
      onClick={approve}
    >
      {pending ? (
        <Loader2 className="size-3.5 animate-spin" />
      ) : (
        <Check className="size-3.5" />
      )}
      Approve
    </button>
  );
}
