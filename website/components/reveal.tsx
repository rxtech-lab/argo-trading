"use client";

import { motion, type HTMLMotionProps } from "framer-motion";

type RevealProps = HTMLMotionProps<"div"> & {
  /** Stagger index — each step adds 0.08s of delay. */
  index?: number;
  /** Extra delay in seconds. */
  delay?: number;
  y?: number;
};

/**
 * Scroll-triggered fade-up. Mirrors the reference site's staggered
 * `animation-range` reveals, but driven by Framer Motion `whileInView`.
 */
export function Reveal({
  index = 0,
  delay = 0,
  y = 40,
  children,
  ...props
}: RevealProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, margin: "-80px" }}
      transition={{
        duration: 0.6,
        ease: [0.22, 1, 0.36, 1],
        delay: delay + index * 0.08,
      }}
      {...props}
    >
      {children}
    </motion.div>
  );
}
