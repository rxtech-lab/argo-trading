import type { Metadata } from "next";
import { Space_Grotesk, Inter, Geist_Mono } from "next/font/google";
import "./globals.css";
import { Toaster } from "@/components/ui/sonner";

const spaceGrotesk = Space_Grotesk({
  variable: "--font-space-grotesk",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
});

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

const siteUrl =
  process.env.NEXT_PUBLIC_SITE_URL ||
  process.env.AUTH_URL ||
  "http://localhost:3000";

export const metadata: Metadata = {
  metadataBase: new URL(siteUrl),
  title: {
    default: "RxArgo — Algorithmic trading, backtested and deployed",
    template: "%s — RxArgo",
  },
  description:
    "RxArgo is a desktop app for developing, backtesting, and deploying WASM trading strategies on real market data. Join the private beta waitlist.",
  applicationName: "RxArgo",
  keywords: [
    "RxArgo",
    "algorithmic trading",
    "backtesting",
    "trading strategies",
    "WASM",
    "Polygon.io",
    "Binance",
  ],
  openGraph: {
    type: "website",
    siteName: "RxArgo",
    title: "RxArgo — Algorithmic trading, backtested and deployed",
    description:
      "Develop, backtest, and deploy WASM trading strategies on real market data — from historical backtest to live trading, in one desktop app.",
    url: siteUrl,
  },
  twitter: {
    card: "summary_large_image",
    title: "RxArgo — Algorithmic trading, backtested and deployed",
    description:
      "Develop, backtest, and deploy WASM trading strategies on real market data — from historical backtest to live trading, in one desktop app.",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`${spaceGrotesk.variable} ${inter.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col bg-background text-foreground">
        {children}
        <Toaster theme="dark" richColors position="top-center" />
      </body>
    </html>
  );
}
