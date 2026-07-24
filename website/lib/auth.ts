import { createRxLabAuth } from "@rxtech-lab/authjs-rxlab";
import type { NextAuthResult } from "next-auth";

let cached: NextAuthResult | null = null;

/**
 * Lazily initialized so that `next build` (which has no env vars) can collect
 * route data without createRxLabAuth throwing on a missing clientId.
 */
function instance(): NextAuthResult {
  return (cached ??= createRxLabAuth({
    issuer: process.env.AUTH_ISSUER || "https://auth.rxlab.app",
    clientId: process.env.AUTH_CLIENT_ID ?? "",
    clientSecret: process.env.AUTH_CLIENT_SECRET ?? "",
    signInPage: "/",
  }));
}

export const handlers: NextAuthResult["handlers"] = {
  GET: (req) => instance().handlers.GET(req),
  POST: (req) => instance().handlers.POST(req),
};

export const auth: NextAuthResult["auth"] = ((...args: unknown[]) =>
  // @ts-expect-error - forward the overloaded auth() signatures verbatim
  instance().auth(...args)) as NextAuthResult["auth"];

export const signIn: NextAuthResult["signIn"] = (...args) =>
  instance().signIn(...args);

export const signOut: NextAuthResult["signOut"] = (...args) =>
  instance().signOut(...args);
