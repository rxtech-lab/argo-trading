import {
  Boxes,
  Gauge,
  ShieldCheck,
  TerminalSquare,
  Waypoints,
  Zap,
} from "lucide-react";
import { SiteHeader } from "@/components/site-header";
import { SignInButton } from "@/components/auth-buttons";
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
    icon: TerminalSquare,
    title: "WASM strategies",
    body: "Write in Go, compile to WebAssembly, run in an isolated plugin sandbox over gRPC.",
  },
  {
    icon: Waypoints,
    title: "Real market data",
    body: "Backtest against real Polygon.io and Binance data, stored locally as fast Parquet.",
  },
  {
    icon: Gauge,
    title: "Rich indicators",
    body: "RSI, MACD, EMA, Bollinger Bands, ATR and more — configured and evaluated by the host.",
  },
  {
    icon: Zap,
    title: "Fast backtests",
    body: "A DuckDB-backed engine crunches years of ticks in seconds so you iterate quickly.",
  },
  {
    icon: ShieldCheck,
    title: "Isolated & safe",
    body: "Strategies are stateless and sandboxed. State lives in the host cache, never the plugin.",
  },
  {
    icon: Boxes,
    title: "Backtest to live",
    body: "Take one strategy from historical simulation to live trading with a single framework.",
  },
];

const STATS = [
  ["Go → WASM", "Sandboxed strategies"],
  ["10+", "Built-in indicators"],
  ["2", "Market-data providers"],
  ["1", "Backtest to live"],
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
                Algorithmic trading,{" "}
                <span className="bg-gradient-to-r from-brand to-brand-2 bg-clip-text text-transparent">
                  backtested
                </span>{" "}
                and deployed.
              </h1>
            </Reveal>

            <Reveal delay={0.1}>
              <p className="mx-auto mt-6 max-w-xl text-lg text-muted-fg">
                Develop, test, and run WASM trading strategies on real market
                data — from historical backtest to live trading, in one desktop
                app.
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

        {/* ── Features ── */}
        <section id="features" className="mx-auto max-w-6xl px-5 py-20 md:px-8 md:py-28">
          <Reveal className="mx-auto max-w-2xl text-center">
            <span className="font-mono text-xs uppercase tracking-wider text-brand">
              The framework
            </span>
            <h2 className="mt-4 font-display text-3xl font-semibold tracking-tight sm:text-4xl md:text-5xl">
              Everything you need to ship a strategy
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
