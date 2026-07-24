import { sql } from "drizzle-orm";
import { sqliteTable, text, integer } from "drizzle-orm/sqlite-core";

/**
 * Waitlist entries. One row per RxLab user (keyed by their OIDC subject id).
 * `status` gates access to the download link:
 *   - "pending":  on the waiting list, no download yet
 *   - "approved": out of the waiting list, download link is shown
 */
export const waitlist = sqliteTable("waitlist", {
  id: text("id")
    .primaryKey()
    .$defaultFn(() => crypto.randomUUID()),
  /** RxLab OIDC subject id (session.user.id) — unique per user. */
  userId: text("user_id").notNull().unique(),
  email: text("email").notNull(),
  name: text("name"),
  status: text("status", { enum: ["pending", "approved"] })
    .notNull()
    .default("pending"),
  createdAt: integer("created_at", { mode: "timestamp" })
    .notNull()
    .default(sql`(unixepoch())`),
  approvedAt: integer("approved_at", { mode: "timestamp" }),
});

export type WaitlistEntry = typeof waitlist.$inferSelect;
export type NewWaitlistEntry = typeof waitlist.$inferInsert;
