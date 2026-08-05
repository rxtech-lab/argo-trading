"use client";

import { useTransition } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { ArrowRight, Check, Clock, Download, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { PRIMARY_CTA } from "@/lib/styles";
import { joinWaitlistAction } from "@/app/actions";

export type WaitlistStatus = "none" | "pending" | "approved";

const panel = {
  initial: { opacity: 0, y: 12 },
  animate: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: -12 },
  transition: { duration: 0.3 },
};

const layout =
  "grid items-center gap-x-12 gap-y-8 md:grid-cols-[1fr_auto]";

function Eyebrow() {
  return (
    <span className="font-mono text-xs uppercase tracking-wider text-brand">
      Get access
    </span>
  );
}

export function WaitlistPanel({
  status,
  downloadUrl,
}: {
  status: WaitlistStatus;
  downloadUrl: string | null;
}) {
  const [pending, start] = useTransition();

  function join() {
    start(async () => {
      try {
        await joinWaitlistAction();
        toast.success("You're on the waitlist!", {
          description: "We'll email you when your spot opens up.",
        });
      } catch {
        toast.error("Something went wrong. Please try again.");
      }
    });
  }

  return (
    <AnimatePresence mode="wait">
      {status === "none" && (
        <motion.div key="none" {...panel} className={layout}>
          <div className="flex flex-col gap-4">
            <Eyebrow />
            <h2 className="font-display text-2xl font-semibold tracking-tight sm:text-3xl md:text-4xl">
              Get early access to RxArgo
            </h2>
            <p className="max-w-xl text-sm text-muted-fg md:text-base">
              You&apos;re signed in. Join the waitlist to get early access to
              the RxArgo desktop app.
            </p>
          </div>
          <button
            className={`${PRIMARY_CTA} w-full md:w-auto`}
            onClick={join}
            disabled={pending}
          >
            Join the waitlist
            {pending ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <ArrowRight className="size-4" />
            )}
          </button>
        </motion.div>
      )}

      {status === "pending" && (
        <motion.div key="pending" {...panel} className={layout}>
          <div className="flex flex-col gap-4">
            <Eyebrow />
            <h2 className="font-display text-2xl font-semibold tracking-tight sm:text-3xl md:text-4xl">
              You&apos;re on the waitlist
            </h2>
            <p className="max-w-xl text-sm text-muted-fg md:text-base">
              We roll out access in batches — the download will appear right
              here the moment your spot opens up.
            </p>
          </div>
          <span className="inline-flex w-fit items-center gap-1.5 rounded-full border border-amber-400/30 bg-amber-400/10 px-4 py-2 text-sm font-medium text-amber-300">
            <Clock className="size-4" />
            On the waitlist
          </span>
        </motion.div>
      )}

      {status === "approved" && (
        <motion.div
          key="approved"
          initial={{ opacity: 0, scale: 0.97 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0, scale: 0.97 }}
          transition={{ type: "spring", stiffness: 220, damping: 20 }}
          className={layout}
        >
          <div className="flex flex-col gap-4">
            <span className="inline-flex w-fit items-center gap-1.5 rounded-full border border-brand/30 bg-brand/10 px-3 py-1 text-xs font-medium text-brand">
              <Check className="size-3.5" />
              You&apos;re in
            </span>
            <h2 className="font-display text-2xl font-semibold tracking-tight sm:text-3xl md:text-4xl">
              Your access is live
            </h2>
            <p className="max-w-xl text-sm text-muted-fg md:text-base">
              Download the RxArgo desktop app for macOS and start building.
            </p>
          </div>
          {downloadUrl ? (
            <a
              className={`${PRIMARY_CTA} w-full md:w-auto`}
              href={downloadUrl}
              download
            >
              <Download className="size-4" />
              Download for macOS
            </a>
          ) : (
            <p className="font-mono text-xs text-destructive">
              Download link is not configured yet. Please check back shortly.
            </p>
          )}
        </motion.div>
      )}
    </AnimatePresence>
  );
}
