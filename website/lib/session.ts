import { auth } from "./auth";

export type AppUser = {
  id: string;
  email: string;
  name: string | null;
  roles: string[];
};

/** Return the signed-in user, or null. Requires a stable OIDC subject id. */
export async function currentUser(): Promise<AppUser | null> {
  const session = await auth();
  const user = session?.user;
  if (!user?.id || !user.email) return null;
  return {
    id: user.id,
    email: user.email,
    name: user.name ?? null,
    roles: user.roles ?? [],
  };
}

/** Admins can approve waitlist entries. Gated on the OIDC `roles` claim. */
export function isAdmin(user: AppUser): boolean {
  return user.roles.includes("admin");
}
