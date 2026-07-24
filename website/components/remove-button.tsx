"use client";

import { useTransition } from "react";
import { useRouter } from "next/navigation";
import { Trash2, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { removeAction } from "@/app/actions";

export function RemoveButton({
  userId,
  name,
}: {
  userId: string;
  name: string;
}) {
  const [pending, start] = useTransition();
  const router = useRouter();

  function remove() {
    if (!window.confirm(`Remove ${name} from the waitlist?`)) return;
    start(async () => {
      try {
        await removeAction(userId);
        toast.success("Removed from waitlist");
        router.refresh();
      } catch {
        toast.error("Could not remove. Are you an admin?");
      }
    });
  }

  return (
    <button
      aria-label={`Remove ${name}`}
      disabled={pending}
      onClick={remove}
      className="inline-flex size-8 items-center justify-center rounded-full text-muted-fg transition-colors hover:bg-destructive/10 hover:text-destructive disabled:opacity-50 cursor-pointer"
    >
      {pending ? (
        <Loader2 className="size-4 animate-spin" />
      ) : (
        <Trash2 className="size-4" />
      )}
    </button>
  );
}
