import { redirect } from "next/navigation";
import { Users, Clock, CircleCheck } from "lucide-react";
import { SiteHeader } from "@/components/site-header";
import { ApproveButton } from "@/components/approve-button";
import { RemoveButton } from "@/components/remove-button";
import { currentUser, isAdmin } from "@/lib/session";
import { listEntries } from "@/lib/waitlist";

export const dynamic = "force-dynamic";

const dateFmt = new Intl.DateTimeFormat("en-US", {
  month: "short",
  day: "numeric",
  year: "numeric",
});

function initials(name: string | null, email: string): string {
  const src = (name || email).trim();
  const parts = src.split(/\s+/).filter(Boolean);
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return src.slice(0, 2).toUpperCase();
}

export default async function AdminPage() {
  const user = await currentUser();
  if (!user) redirect("/");
  if (!isAdmin(user)) redirect("/");

  const entries = await listEntries();
  const pending = entries.filter((e) => e.status === "pending");
  const approved = entries.filter((e) => e.status === "approved");

  const stats = [
    { label: "Total", value: entries.length, Icon: Users, tone: "text-foreground" },
    { label: "Pending", value: pending.length, Icon: Clock, tone: "text-amber-300" },
    { label: "Approved", value: approved.length, Icon: CircleCheck, tone: "text-brand" },
  ];

  return (
    <>
      <SiteHeader user={user} isAdmin />
      <main className="w-full flex-1 px-5 py-12 md:px-8 lg:px-12">
        {/* Heading */}
        <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <span className="font-mono text-xs uppercase tracking-wider text-brand">
              Admin
            </span>
            <h1 className="mt-3 font-display text-3xl font-semibold tracking-tight sm:text-4xl">
              Waitlist
            </h1>
            <p className="mt-2 text-sm text-muted-fg">
              Review everyone who requested access and approve them to unlock the
              download.
            </p>
          </div>
        </div>

        {/* Stats */}
        <div className="mt-8 grid grid-cols-1 gap-4 sm:grid-cols-3">
          {stats.map(({ label, value, Icon, tone }) => (
            <div
              key={label}
              className="glass relative flex items-center gap-4 overflow-hidden rounded-2xl px-6 py-5"
            >
              <span
                className={`flex size-11 shrink-0 items-center justify-center rounded-xl bg-white/5 ${tone}`}
              >
                <Icon className="size-5" strokeWidth={2} />
              </span>
              <div>
                <div className="font-display text-3xl font-semibold tracking-tight tabular-nums">
                  {value}
                </div>
                <div className="mt-0.5 text-xs uppercase tracking-wide text-muted-fg">
                  {label}
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Table */}
        <div className="glass mt-8 overflow-hidden rounded-2xl">
          {entries.length === 0 ? (
            <div className="flex flex-col items-center gap-3 px-6 py-20 text-center">
              <span className="flex size-12 items-center justify-center rounded-full bg-white/5 text-muted-fg">
                <Users className="size-6" strokeWidth={1.75} />
              </span>
              <p className="text-sm text-muted-fg">
                No one has joined the waitlist yet.
              </p>
            </div>
          ) : (
            <>
              {/* Column header — hidden on mobile */}
              <div className="hidden grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)_120px_auto] items-center gap-4 border-b border-white/8 px-6 py-3 text-[11px] font-medium uppercase tracking-wider text-muted-fg md:grid">
                <span>User</span>
                <span>Email</span>
                <span>Joined</span>
                <span className="text-right">Status</span>
              </div>

              <ul className="divide-y divide-white/8">
                {entries.map((e) => (
                  <li
                    key={e.id}
                    className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-4 px-6 py-4 transition-colors hover:bg-white/[0.03] md:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)_120px_auto]"
                  >
                    {/* User */}
                    <div className="flex min-w-0 items-center gap-3">
                      <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-brand/25 to-brand-2/20 font-mono text-[11px] font-semibold text-brand">
                        {initials(e.name, e.email)}
                      </span>
                      <div className="min-w-0">
                        <p className="truncate font-medium">
                          {e.name || e.email}
                        </p>
                        {/* Email inline on mobile only */}
                        <p className="truncate font-mono text-xs text-muted-fg md:hidden">
                          {e.email}
                        </p>
                      </div>
                    </div>

                    {/* Email */}
                    <p className="hidden min-w-0 truncate font-mono text-xs text-muted-fg md:block">
                      {e.email}
                    </p>

                    {/* Joined */}
                    <p className="hidden text-xs text-muted-fg md:block">
                      {e.createdAt ? dateFmt.format(e.createdAt) : "—"}
                    </p>

                    {/* Status + action */}
                    <div className="flex shrink-0 items-center justify-end gap-2">
                      {e.status === "approved" ? (
                        <span className="inline-flex items-center gap-1.5 rounded-full border border-brand/30 bg-brand/10 px-3 py-1 text-xs font-medium text-brand">
                          <CircleCheck className="size-3.5" />
                          Approved
                        </span>
                      ) : (
                        <>
                          <span className="inline-flex items-center gap-1.5 rounded-full border border-amber-400/30 bg-amber-400/10 px-3 py-1 text-xs font-medium text-amber-300">
                            <Clock className="size-3.5" />
                            Pending
                          </span>
                          <ApproveButton
                            userId={e.userId}
                            name={e.name || e.email}
                          />
                        </>
                      )}
                      <RemoveButton userId={e.userId} name={e.name || e.email} />
                    </div>
                  </li>
                ))}
              </ul>
            </>
          )}
        </div>
      </main>
    </>
  );
}
