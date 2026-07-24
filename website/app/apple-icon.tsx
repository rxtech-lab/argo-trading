import { ImageResponse } from "next/og";

export const size = { width: 180, height: 180 };
export const contentType = "image/png";

export default function AppleIcon() {
  return new ImageResponse(
    (
      <div
        style={{
          display: "flex",
          width: "100%",
          height: "100%",
          alignItems: "center",
          justifyContent: "center",
          background: "#06090e",
        }}
      >
        <div
          style={{
            display: "flex",
            width: 132,
            height: 132,
            alignItems: "center",
            justifyContent: "center",
            background: "#34d399",
            borderRadius: 32,
          }}
        >
          <svg
            width="88"
            height="88"
            viewBox="0 0 24 24"
            fill="none"
            stroke="#05130c"
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M3 12h3.5l2.5-7 4 14 2.5-7H21" />
          </svg>
        </div>
      </div>
    ),
    { ...size },
  );
}
