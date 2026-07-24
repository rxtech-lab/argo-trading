import { ImageResponse } from "next/og";

export const alt = "RxArgo — Algorithmic trading, backtested and deployed";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

export default function OpengraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          width: "100%",
          height: "100%",
          padding: 80,
          justifyContent: "space-between",
          color: "#e8eef6",
          background: "#06090e",
          backgroundImage:
            "radial-gradient(900px 500px at 15% -10%, rgba(52,211,153,0.30), transparent 60%), radial-gradient(800px 500px at 100% 110%, rgba(34,211,238,0.18), transparent 60%)",
          fontFamily: "sans-serif",
        }}
      >
        {/* Brand */}
        <div style={{ display: "flex", alignItems: "center", gap: 20 }}>
          <div
            style={{
              display: "flex",
              width: 64,
              height: 64,
              alignItems: "center",
              justifyContent: "center",
              background: "#34d399",
              borderRadius: 16,
            }}
          >
            <svg
              width="42"
              height="42"
              viewBox="0 0 24 24"
              fill="none"
              stroke="#05130c"
              strokeWidth="2.6"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M3 12h3.5l2.5-7 4 14 2.5-7H21" />
            </svg>
          </div>
          <div style={{ fontSize: 38, fontWeight: 700, letterSpacing: -1 }}>
            RxArgo
          </div>
        </div>

        {/* Headline */}
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            fontSize: 82,
            fontWeight: 700,
            lineHeight: 1.05,
            letterSpacing: -2.5,
          }}
        >
          <div style={{ display: "flex" }}>Algorithmic trading,</div>
          <div style={{ display: "flex", gap: 22 }}>
            <span style={{ color: "#34d399" }}>backtested</span>
            <span>and deployed.</span>
          </div>
        </div>

        {/* Footer */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
          }}
        >
          <div style={{ display: "flex", fontSize: 26, color: "#8b98a9" }}>
            WASM strategies · real market data · backtest to live
          </div>
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 12,
              padding: "10px 20px",
              borderRadius: 999,
              border: "1px solid rgba(255,255,255,0.12)",
              fontSize: 24,
              color: "#e8eef6",
            }}
          >
            <div
              style={{
                display: "flex",
                width: 12,
                height: 12,
                borderRadius: 999,
                background: "#34d399",
              }}
            />
            Now in private beta
          </div>
        </div>
      </div>
    ),
    { ...size },
  );
}
