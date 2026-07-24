import Image from "next/image";
import { cn } from "@/lib/utils";

/**
 * A macOS-style window chrome around a product screenshot, with a soft
 * emerald glow behind it. Used for the hero shot and the showcase rows.
 */
export function WindowFrame({
  src,
  alt,
  title,
  priority,
  sizes,
  className,
  glow = true,
}: {
  src: string;
  alt: string;
  title?: string;
  priority?: boolean;
  sizes?: string;
  className?: string;
  glow?: boolean;
}) {
  return (
    <div className={cn("relative", className)}>
      {glow && (
        <div
          aria-hidden
          className="pointer-events-none absolute -inset-8 -z-10 opacity-60 blur-3xl"
          style={{
            background:
              "radial-gradient(60% 60% at 50% 40%, rgba(52,211,153,0.28), transparent 70%)",
          }}
        />
      )}
      <div className="overflow-hidden rounded-xl border border-white/10 bg-[#0b0f16] shadow-2xl shadow-black/60 ring-1 ring-white/5">
        {/* Title bar */}
        <div className="flex items-center gap-2 border-b border-white/8 bg-white/[0.03] px-4 py-3">
          <span className="size-3 rounded-full bg-[#ff5f57]" />
          <span className="size-3 rounded-full bg-[#febc2e]" />
          <span className="size-3 rounded-full bg-[#28c840]" />
          {title && (
            <span className="ml-3 truncate font-mono text-xs text-muted-fg">
              {title}
            </span>
          )}
        </div>
        {/* Screenshot */}
        <div className="relative aspect-[16/9] w-full">
          <Image
            src={src}
            alt={alt}
            fill
            priority={priority}
            sizes={sizes ?? "(max-width: 1024px) 100vw, 60vw"}
            className="object-cover object-left-top"
          />
        </div>
      </div>
    </div>
  );
}
