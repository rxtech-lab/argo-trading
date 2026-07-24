"use server";

import { revalidatePath } from "next/cache";
import { signIn, signOut } from "@/lib/auth";
import { currentUser, isAdmin } from "@/lib/session";
import { approveEntry, joinWaitlist, removeEntry } from "@/lib/waitlist";

export async function signInAction() {
  await signIn("rxlab", { redirectTo: "/" });
}

export async function signOutAction() {
  await signOut({ redirectTo: "/" });
}

export async function joinWaitlistAction() {
  const user = await currentUser();
  if (!user) throw new Error("Not authenticated");
  await joinWaitlist(user);
  revalidatePath("/");
}

export async function approveAction(userId: string) {
  const user = await currentUser();
  if (!user || !isAdmin(user)) throw new Error("Forbidden");
  await approveEntry(userId);
  revalidatePath("/admin");
  revalidatePath("/");
}

export async function removeAction(userId: string) {
  const user = await currentUser();
  if (!user || !isAdmin(user)) throw new Error("Forbidden");
  await removeEntry(userId);
  revalidatePath("/admin");
  revalidatePath("/");
}
