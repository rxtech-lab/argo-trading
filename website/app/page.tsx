import {
  Activity,
  ArrowUpRight,
  FileSearch,
  Globe,
  History,
  Layers,
  LineChart,
  MessagesSquare,
  Newspaper,
  Sparkles,
} from "lucide-react";
import { SiteHeader } from "@/components/site-header";
import { SignInButton } from "@/components/auth-buttons";
import { PRIMARY_CTA } from "@/lib/styles";
import { cn } from "@/lib/utils";
import { Reveal } from "@/components/reveal";
import { WindowFrame } from "@/components/window-frame";
import { ScreenshotShowcase } from "@/components/screenshot-showcase";
import {
  WaitlistPanel,
  type WaitlistStatus,
} from "@/components/waitlist-panel";
import { currentUser, isAdmin } from "@/lib/session";
import { getEntryForUser } from "@/lib/waitlist";
import { getLatestMacDownloadUrl } from "@/lib/release";

// Depends on the signed-in RxLab session (cookies) — never prerender.
export const dynamic = "force-dynamic";

const FEATURES = [
  {
    icon: Newspaper,
    title: "Plan before you invest",
    body: "Write equity research reports, search the latest news on any name, and turn it into a clear write-up — so you decide what's worth investing in before committing a dollar.",
  },
  {
    icon: History,
    title: "Backtest with confidence",
    body: "Replay your strategy against years of real market data and see how it would have performed before you ever go live.",
  },
  {
    icon: Activity,
    title: "Go live",
    body: "Take the exact strategy you planned and backtested straight to live trading — no rewrites, one platform end to end.",
  },
  {
    icon: Layers,
    title: "Every asset class",
    body: "Trade equities, crypto, and more from a single platform — bring the assets you care about together.",
  },
  {
    icon: Globe,
    title: "Any market",
    body: "Reach across different markets and venues, so your strategies aren't boxed into one exchange.",
  },
  {
    icon: Sparkles,
    title: "AI-powered strategies",
    body: "Build any strategy with AI — different edges, different profit engines. You think it, you get it.",
  },
];

const RESEARCH_URL = "https://finance.bots.rxlab.app/";

const RESEARCH_POINTS = [
  {
    icon: LineChart,
    title: "Live quotes and odds",
    body: "Equity and crypto prices, historical ranges, and prediction-market odds pulled as the agent works.",
  },
  {
    icon: FileSearch,
    title: "Filings and news, actually read",
    body: "It opens SEC filings and full news articles, then cites the exact lines behind every figure.",
  },
  {
    icon: MessagesSquare,
    title: "Answers in the thread",
    body: "Charts, tables, and probability bars land inline — keep the write-up as a searchable document.",
  },
];

const STATS = [
  ["Plan", "Research before you invest"],
  ["Backtest", "Years of real market data"],
  ["Live", "Trade in one platform"],
  ["AI", "Any strategy you imagine"],
];

export default async function Home() {
  const user = await currentUser();
  const admin = user ? isAdmin(user) : false;
  const entry = user ? await getEntryForUser(user.id) : null;

  const status: WaitlistStatus = !entry
    ? "none"
    : entry.status === "approved"
      ? "approved"
      : "pending";

  const downloadUrl =
    status === "approved" ? await getLatestMacDownloadUrl() : null;

  return (
    <>
      <SiteHeader user={user} isAdmin={admin} />

      <main className="flex-1">
        {/* ── Hero ── */}
        <section className="relative overflow-hidden">
          <div className="grid-dots pointer-events-none absolute inset-0 -z-10 opacity-60 [mask-image:radial-gradient(70%_50%_at_50%_0%,black,transparent)]" />
          <div className="mx-auto max-w-6xl px-5 pt-20 pb-16 text-center md:px-8 md:pt-28">
            <Reveal className="flex justify-center">
              <span className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/[0.04] px-3.5 py-1.5 text-xs text-muted-fg backdrop-blur">
                <span className="relative flex size-2">
                  <span className="absolute inline-flex size-full animate-ping rounded-full bg-brand opacity-70" />
                  <span className="relative inline-flex size-2 rounded-full bg-brand" />
                </span>
                Now in private beta
              </span>
            </Reveal>

            <Reveal delay={0.05}>
              <h1 className="mx-auto mt-7 max-w-4xl font-display text-5xl font-semibold leading-[1.03] tracking-tight sm:text-6xl md:text-7xl">
                Plan, backtest,{" "}
                <span className="bg-gradient-to-r from-brand to-brand-2 bg-clip-text text-transparent">
                  and trade live
                </span>{" "}
                — all in one place.
              </h1>
            </Reveal>

            <Reveal delay={0.1}>
              <p className="mx-auto mt-6 max-w-xl text-lg text-muted-fg">
                Research what to invest in, backtest your strategy on real market
                data, and take it live — across asset classes and markets, in one
                desktop app.
              </p>
            </Reveal>

            <Reveal
              delay={0.15}
              className="mt-9 flex flex-col items-center justify-center gap-3 sm:flex-row"
            >
              {!user ? (
                <SignInButton label="Join the waitlist" />
              ) : (
                <a
                  href="#access"
                  className="inline-flex items-center gap-2 rounded-full bg-brand px-6 py-3 text-sm font-semibold text-[#05130c] shadow-lg shadow-brand/25 transition hover:brightness-110"
                >
                  Go to your access
                </a>
              )}
              <a
                href="#features"
                className="inline-flex items-center gap-2 rounded-full border border-white/12 px-6 py-3 text-sm font-medium text-foreground/90 transition hover:bg-white/5"
              >
                Explore the app
              </a>
            </Reveal>

            <Reveal delay={0.2}>
              <p className="mt-5 text-sm text-muted-fg">
                Don&apos;t want to wait?{" "}
                <a
                  href="#research"
                  className="font-medium text-brand underline-offset-4 hover:underline"
                >
                  Try the research desk in your browser
                </a>{" "}
                today.
              </p>
            </Reveal>
          </div>

          {/* Hero screenshot */}
          <Reveal
            y={60}
            delay={0.2}
            className="mx-auto max-w-5xl px-5 pb-4 md:px-8"
          >
            <WindowFrame
              src="/screens/trade-details.webp"
              alt="RxArgo candlestick chart with a trade detail popover and full trades table"
              title="BTCUSDT · Backtest v1 — Trades"
              priority
              sizes="(max-width: 1024px) 100vw, 1024px"
            />
          </Reveal>

          {/* Stats */}
          <div className="mx-auto max-w-5xl px-5 pb-20 pt-10 md:px-8">
            <Reveal className="grid grid-cols-2 gap-px overflow-hidden rounded-2xl border border-white/8 bg-white/5 md:grid-cols-4">
              {STATS.map(([v, l]) => (
                <div key={l} className="bg-background/60 px-6 py-6 text-center">
                  <div className="font-display text-3xl font-semibold tracking-tight">
                    {v}
                  </div>
                  <div className="mt-1 text-xs text-muted-fg">{l}</div>
                </div>
              ))}
            </Reveal>
          </div>
        </section>

        {/* ── Access (only shown when signed in) ── */}
        {user && (
          <section id="access" className="mx-auto max-w-6xl px-5 py-16 md:px-8 md:py-20">
            <Reveal className="border-t border-white/10 pt-14">
              <WaitlistPanel status={status} downloadUrl={downloadUrl} />
            </Reveal>
          </section>
        )}

        {/* ── Research desk (available now, ahead of desktop access) ── */}
        <section
          id="research"
          className="mx-auto max-w-6xl px-5 py-16 md:px-8 md:py-20"
        >
          <Reveal className="relative overflow-hidden rounded-3xl border border-white/10 bg-white/[0.02] px-6 py-12 md:px-12 md:py-14">
            <div
              aria-hidden
              className="pointer-events-none absolute -right-20 -top-24 -z-10 size-80 opacity-60 blur-3xl"
              style={{
                background:
                  "radial-gradient(50% 50% at 50% 50%, rgba(34,211,238,0.22), transparent 70%)",
              }}
            />

            <div className="grid gap-10 lg:grid-cols-2 lg:items-center lg:gap-14">
              <div>
                <span className="inline-flex items-center gap-2 rounded-full border border-brand/25 bg-brand/10 px-3 py-1 font-mono text-xs uppercase tracking-wider text-brand">
                  Available now
                </span>
                <h2 className="mt-5 font-display text-3xl font-semibold tracking-tight sm:text-4xl">
                  Start with the research desk
                </h2>
                <p className="mt-5 text-muted-fg">
                  Waiting on desktop access? The research half of the platform is
                  already live on the web. Ask the Intelligence Desk anything
                  about a name — it pulls live quotes, reads the filings and the
                  news, and answers with sources you can check.
                </p>
                <p className="mt-4 text-sm text-muted-fg">
                  Same RxLab sign-in you use here. Nothing to install.
                </p>

                <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center">
                  <a
                    href={RESEARCH_URL}
                    target="_blank"
                    rel="noopener noreferrer"
                    className={cn(PRIMARY_CTA, "whitespace-nowrap")}
                  >
                    Try the research desk
                    <ArrowUpRight className="size-4" strokeWidth={2.5} />
                  </a>
                  {!user && (
                    <SignInButton
                      label="Join the waitlist"
                      className="whitespace-nowrap border border-white/12 bg-transparent text-foreground/90 shadow-none hover:bg-white/5 hover:shadow-none"
                    />
                  )}
                </div>
              </div>

              <div className="grid gap-4">
                {RESEARCH_POINTS.map((p, i) => (
                  <Reveal
                    key={p.title}
                    index={i}
                    className="flex gap-4 rounded-2xl border border-white/8 bg-white/[0.02] p-5"
                  >
                    <div className="flex size-10 shrink-0 items-center justify-center rounded-xl border border-brand/20 bg-brand/10 text-brand">
                      <p.icon className="size-5" strokeWidth={1.75} />
                    </div>
                    <div>
                      <h3 className="font-display text-base font-semibold tracking-tight">
                        {p.title}
                      </h3>
                      <p className="mt-1.5 text-sm leading-relaxed text-muted-fg">
                        {p.body}
                      </p>
                    </div>
                  </Reveal>
                ))}
              </div>
            </div>
          </Reveal>
        </section>

        {/* ── Features ── */}
        <section id="features" className="mx-auto max-w-6xl px-5 py-20 md:px-8 md:py-28">
          <Reveal className="mx-auto max-w-2xl text-center">
            <span className="font-mono text-xs uppercase tracking-wider text-brand">
              The platform
            </span>
            <h2 className="mt-4 font-display text-3xl font-semibold tracking-tight sm:text-4xl md:text-5xl">
              From research to live trading
            </h2>
          </Reveal>

          <div className="mt-14 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {FEATURES.map((f, i) => (
              <Reveal
                key={f.title}
                index={i}
                className="group rounded-2xl border border-white/8 bg-white/[0.02] p-6 transition-colors hover:border-brand/30 hover:bg-white/[0.04]"
              >
                <div className="flex size-11 items-center justify-center rounded-xl border border-brand/20 bg-brand/10 text-brand">
                  <f.icon className="size-5" strokeWidth={1.75} />
                </div>
                <h3 className="mt-5 font-display text-lg font-semibold tracking-tight">
                  {f.title}
                </h3>
                <p className="mt-2 text-sm leading-relaxed text-muted-fg">
                  {f.body}
                </p>
              </Reveal>
            ))}
          </div>
        </section>

        {/* ── Showcase ── */}
        <section className="mx-auto max-w-6xl px-5 py-16 md:px-8 md:py-20">
          <Reveal className="mx-auto max-w-2xl text-center">
            <span className="font-mono text-xs uppercase tracking-wider text-brand">
              Inside the app
            </span>
            <h2 className="mt-4 font-display text-3xl font-semibold tracking-tight sm:text-4xl md:text-5xl">
              Built for people who read the tape
            </h2>
          </Reveal>
          <div className="mt-20">
            <ScreenshotShowcase />
          </div>
        </section>

        {/* ── CTA ── */}
        <section className="mx-auto max-w-6xl px-5 pb-24 md:px-8">
          <Reveal className="relative overflow-hidden rounded-3xl border border-white/10 bg-gradient-to-b from-white/[0.06] to-transparent px-6 py-16 text-center md:py-20">
            <div
              aria-hidden
              className="pointer-events-none absolute inset-x-0 -top-20 -z-10 h-56 opacity-70 blur-3xl"
              style={{
                background:
                  "radial-gradient(50% 80% at 50% 50%, rgba(52,211,153,0.25), transparent 70%)",
              }}
            />
            <h2 className="mx-auto max-w-2xl font-display text-3xl font-semibold tracking-tight sm:text-4xl md:text-5xl">
              Ready to trade smarter?
            </h2>
            <p className="mx-auto mt-5 max-w-md text-muted-fg">
              Join the waitlist today. We&apos;re onboarding new traders every
              week — sign in with RxLab to claim your spot.
            </p>
            {!user && (
              <div className="mt-8 flex justify-center">
                <SignInButton label="Join the waitlist" />
              </div>
            )}
          </Reveal>
        </section>
      </main>

      <footer className="border-t border-white/8">
        <div className="mx-auto flex max-w-6xl flex-col items-center justify-between gap-3 px-5 py-8 md:flex-row md:px-8">
          <span className="font-display text-sm font-semibold tracking-tight">
            RxArgo
          </span>
          <span className="font-mono text-xs text-muted-fg">
            © {new Date().getFullYear()} · Built with Next.js · Turso · RxLab
            Auth
          </span>
        </div>
      </footer>
    </>
  );
}
