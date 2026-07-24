import "server-only";
import { Resend } from "resend";

/**
 * Transactional email via Resend. Sends the waitlist join confirmation and the
 * waitlist-approval notification. Sending is best-effort: callers should not let
 * a mail failure roll back the underlying write (see `joinWaitlist` /
 * `approveEntry`).
 */

const DEFAULT_FROM = "RxArgo <onboarding@resend.dev>";

function client(): Resend | null {
  const apiKey = process.env.RESEND_API_KEY;
  if (!apiKey) return null;
  return new Resend(apiKey);
}

function from(): string {
  return process.env.EMAIL_FROM || DEFAULT_FROM;
}

/** Public base URL of this deployment, used to link back into the app. */
function appUrl(): string {
  return (process.env.AUTH_URL || "https://argo.rxlab.app").replace(/\/+$/, "");
}

/**
 * Confirm to a user that they've joined the RxArgo waitlist. Returns `false`
 * (and logs) when Resend isn't configured or the send fails — it never throws,
 * so the join flow stays unaffected.
 */
export async function sendWaitlistJoinedEmail(to: {
  email: string;
  name?: string | null;
}): Promise<boolean> {
  const resend = client();
  if (!resend) {
    console.warn(
      "[email] RESEND_API_KEY not set; skipping waitlist joined email",
    );
    return false;
  }

  const firstName = to.name?.trim().split(/\s+/)[0] || "there";
  const url = appUrl();

  try {
    const { error } = await resend.emails.send({
      from: from(),
      to: to.email,
      subject: "You're on the RxArgo waitlist ✅",
      text: [
        `Hi ${firstName},`,
        "",
        "Thanks for joining the RxArgo waitlist — you're on the list.",
        "We'll email you as soon as your access is approved and the macOS",
        "app is ready to download.",
        "",
        "In the meantime, check us out here:",
        url,
        "",
        "Talk soon,",
        "The RxArgo Team",
      ].join("\n"),
      html: joinedHtml(firstName, url),
    });

    if (error) {
      console.error("[email] failed to send waitlist joined email", error);
      return false;
    }
    return true;
  } catch (err) {
    console.error("[email] failed to send waitlist joined email", err);
    return false;
  }
}

function joinedHtml(firstName: string, url: string): string {
  return `
  <div style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;line-height:1.6;color:#111;max-width:520px;margin:0 auto;padding:24px">
    <h1 style="font-size:20px;margin:0 0 16px">You're on the waitlist ✅</h1>
    <p style="margin:0 0 12px">Hi ${firstName},</p>
    <p style="margin:0 0 12px">
      Thanks for joining the RxArgo waitlist — you're on the list. We'll email
      you as soon as your access is approved and the macOS app is ready to
      download.
    </p>
    <p style="margin:24px 0">
      <a href="${url}"
         style="display:inline-block;background:#111;color:#fff;text-decoration:none;padding:12px 20px;border-radius:8px;font-weight:600">
        Visit RxArgo
      </a>
    </p>
    <p style="margin:0;color:#666;font-size:13px">Talk soon,<br/>The RxArgo Team</p>
  </div>`;
}

/**
 * Notify a user that their waitlist entry was approved and the download is now
 * available. Returns `false` (and logs) when Resend isn't configured or the
 * send fails — it never throws, so approval flows stay unaffected.
 */
export async function sendWaitlistApprovalEmail(to: {
  email: string;
  name?: string | null;
}): Promise<boolean> {
  const resend = client();
  if (!resend) {
    console.warn(
      "[email] RESEND_API_KEY not set; skipping waitlist approval email",
    );
    return false;
  }

  const firstName = to.name?.trim().split(/\s+/)[0] || "there";
  const url = appUrl();

  try {
    const { error } = await resend.emails.send({
      from: from(),
      to: to.email,
      subject: "You're off the RxArgo waitlist 🎉",
      text: [
        `Hi ${firstName},`,
        "",
        "Good news — you've been approved off the RxArgo waitlist.",
        "You can now sign in and download the macOS app:",
        "",
        url,
        "",
        "Happy trading,",
        "The RxArgo Team",
      ].join("\n"),
      html: approvalHtml(firstName, url),
    });

    if (error) {
      console.error("[email] failed to send waitlist approval email", error);
      return false;
    }
    return true;
  } catch (err) {
    console.error("[email] failed to send waitlist approval email", err);
    return false;
  }
}

function approvalHtml(firstName: string, url: string): string {
  return `
  <div style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;line-height:1.6;color:#111;max-width:520px;margin:0 auto;padding:24px">
    <h1 style="font-size:20px;margin:0 0 16px">You're off the waitlist 🎉</h1>
    <p style="margin:0 0 12px">Hi ${firstName},</p>
    <p style="margin:0 0 12px">
      Good news — you've been approved off the RxArgo waitlist.
      You can now sign in and download the macOS app.
    </p>
    <p style="margin:24px 0">
      <a href="${url}"
         style="display:inline-block;background:#111;color:#fff;text-decoration:none;padding:12px 20px;border-radius:8px;font-weight:600">
        Open RxArgo
      </a>
    </p>
    <p style="margin:0;color:#666;font-size:13px">Happy trading,<br/>The RxArgo Team</p>
  </div>`;
}
