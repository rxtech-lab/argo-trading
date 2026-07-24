"use client";

import { motion } from "framer-motion";

// Deterministic pseudo-candles so server/client markup matches (no Math.random).
const CANDLES = Array.from({ length: 28 }, (_, i) => {
  const wave = Math.sin(i * 0.55) * 22 + Math.cos(i * 0.23) * 12;
  const mid = 90 + wave + (i % 5) * 3;
  const bodyH = 14 + ((i * 37) % 26);
  const wickH = bodyH + 10 + ((i * 13) % 18);
  const up = (i * 7) % 3 !== 0;
  return { x: i * 26 + 12, mid, bodyH, wickH, up };
});

export function ChartBackdrop() {
  return (
    <div className="pointer-events-none absolute inset-0 overflow-hidden [mask-image:linear-gradient(to_bottom,transparent,black_18%,black_70%,transparent)]">
      <svg
        className="absolute bottom-0 left-1/2 h-[70%] w-[200%] -translate-x-1/2 opacity-[0.28] dark:opacity-40"
        viewBox="0 0 730 260"
        preserveAspectRatio="xMidYMax slice"
        aria-hidden
      >
        {CANDLES.map((c, i) => {
          const color = c.up ? "var(--up)" : "var(--down)";
          return (
            <motion.g
              key={i}
              initial={{ opacity: 0, y: 24 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{
                delay: i * 0.04,
                duration: 0.6,
                ease: "easeOut",
              }}
              style={
                {
                  "--up": "oklch(0.72 0.17 152)",
                  "--down": "oklch(0.63 0.2 20)",
                } as React.CSSProperties
              }
            >
              <line
                x1={c.x}
                x2={c.x}
                y1={c.mid - c.wickH / 2}
                y2={c.mid + c.wickH / 2}
                stroke={color}
                strokeWidth={1.5}
              />
              <rect
                x={c.x - 6}
                y={c.mid - c.bodyH / 2}
                width={12}
                height={c.bodyH}
                rx={2}
                fill={color}
              />
            </motion.g>
          );
        })}
      </svg>
    </div>
  );
}
