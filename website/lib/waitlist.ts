import "server-only";
import { eq, desc } from "drizzle-orm";
import { getDb } from "./db";
import { waitlist, type WaitlistEntry } from "./db/schema";
import { sendWaitlistApprovalEmail, sendWaitlistJoinedEmail } from "./email";

export async function getEntryForUser(
  userId: string,
): Promise<WaitlistEntry | null> {
  const db = getDb();
  const rows = await db
    .select()
    .from(waitlist)
    .where(eq(waitlist.userId, userId))
    .limit(1);
  return rows[0] ?? null;
}

export async function joinWaitlist(user: {
  id: string;
  email: string;
  name?: string | null;
}): Promise<WaitlistEntry> {
  const db = getDb();
  const existing = await getEntryForUser(user.id);
  if (existing) return existing;

  const rows = await db
    .insert(waitlist)
    .values({ userId: user.id, email: user.email, name: user.name ?? null })
    .onConflictDoNothing({ target: waitlist.userId })
    .returning();

  const inserted = rows[0];
  if (inserted) {
    // Fresh join — confirm by email. Best-effort; a mail failure must not fail
    // the join. A concurrent insert (no returned row) already sent this.
    await sendWaitlistJoinedEmail({
      email: inserted.email,
      name: inserted.name,
    });
    return inserted;
  }

  // If a concurrent insert won the race, fall back to the persisted row.
  return (await getEntryForUser(user.id))!;
}

export async function approveEntry(userId: string): Promise<void> {
  const db = getDb();
  const rows = await db
    .update(waitlist)
    .set({ status: "approved", approvedAt: new Date() })
    .where(eq(waitlist.userId, userId))
    .returning();

  const entry = rows[0];
  if (entry) {
    // Best-effort notification; a mail failure must not fail the approval.
    await sendWaitlistApprovalEmail({ email: entry.email, name: entry.name });
  }
}

export async function removeEntry(userId: string): Promise<void> {
  const db = getDb();
  await db.delete(waitlist).where(eq(waitlist.userId, userId));
}

export async function listEntries(): Promise<WaitlistEntry[]> {
  const db = getDb();
  return db.select().from(waitlist).orderBy(desc(waitlist.createdAt));
}
