import { drizzle, type LibSQLDatabase } from "drizzle-orm/libsql";
import { createClient } from "@libsql/client";
import * as schema from "./schema";

let cached: LibSQLDatabase<typeof schema> | null = null;

/**
 * Lazily created so importing this module during `next build` (no env vars)
 * does not throw. The connection is only established on first query.
 */
export function getDb(): LibSQLDatabase<typeof schema> {
  if (cached) return cached;
  const url = process.env.TURSO_DATABASE_URL;
  if (!url) throw new Error("TURSO_DATABASE_URL is not set");
  const client = createClient({ url, authToken: process.env.TURSO_AUTH_TOKEN });
  return (cached = drizzle(client, { schema }));
}

export { schema };
