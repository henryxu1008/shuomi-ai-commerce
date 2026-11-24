import type { Metadata } from "next";
import { GoogleAuthorizationCallback } from "@/components/workbench/stores/google-connection";

export const metadata: Metadata = { referrer: "no-referrer" };

export default function Page() {
  return <GoogleAuthorizationCallback />;
}
