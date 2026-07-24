import { Reveal } from "@/components/reveal";
import { WindowFrame } from "@/components/window-frame";

type Shot = {
  n: string;
  tag: string;
  title: string;
  body: string;
  src: string;
  alt: string;
  window: string;
};

const SHOTS: Shot[] = [
  {
    n: "01",
    tag: "Backtest",
    title: "Every result, fully broken down",
    body: "Run a strategy across years of ticks and inspect total trades, win rate, and P&L. Full run history is kept so you can compare iterations side by side.",
    src: "/screens/strategy-overview.webp",
    alt: "RxArgo strategy overview with metadata, overview stats and run history",
    window: "Place Order Strategy — Results",
  },
  {
    n: "02",
    tag: "Configure",
    title: "Tune the engine, not the plumbing",
    body: "Set initial capital, commission broker, Sharpe annualization, risk-free rate and cache size. The same config drives backtest, strategy and live trading.",
    src: "/screens/engine-config.webp",
    alt: "RxArgo backtest engine configuration dialog",
    window: "Backtest Engine Configuration",
  },
  {
    n: "03",
    tag: "Market data",
    title: "Pull real data on demand",
    body: "Download historical candles straight from Polygon.io or Binance by ticker, range and interval — stored locally as fast Parquet for instant replays.",
    src: "/screens/data-download.webp",
    alt: "RxArgo market data provider download dialog",
    window: "Download Market Data",
  },
  {
    n: "04",
    tag: "Wallet",
    title: "Fund it and trade for real",
    body: "A built-in wallet tracks buying power, assets and orders — so the strategy you backtested can move from simulation to live trading in the same app.",
    src: "/screens/wallet.webp",
    alt: "RxArgo wallet showing balance, buying power and assets",
    window: "Wallet",
  },
];

export function ScreenshotShowcase() {
  return (
    <div className="flex flex-col gap-20 md:gap-28">
      {SHOTS.map((s, i) => {
        const flip = i % 2 === 1;
        return (
          <div
            key={s.n}
            className="grid grid-cols-1 items-center gap-10 lg:grid-cols-2 lg:gap-16"
          >
            <Reveal
              className={`flex flex-col gap-5 ${flip ? "lg:order-2" : ""}`}
            >
              <span className="inline-flex w-fit items-center gap-2 rounded-full border border-white/10 bg-white/[0.03] px-3 py-1 font-mono text-xs text-brand">
                {s.n} · {s.tag}
              </span>
              <h3 className="font-display text-2xl font-semibold tracking-tight sm:text-3xl">
                {s.title}
              </h3>
              <p className="max-w-md leading-relaxed text-muted-fg">{s.body}</p>
            </Reveal>

            <Reveal
              y={50}
              delay={0.1}
              className={flip ? "lg:order-1" : ""}
            >
              <WindowFrame
                src={s.src}
                alt={s.alt}
                title={s.window}
                sizes="(max-width: 1024px) 100vw, 50vw"
              />
            </Reveal>
          </div>
        );
      })}
    </div>
  );
}
