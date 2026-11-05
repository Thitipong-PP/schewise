// SPDX-License-Identifier: Apache-2.0
import localFont from "next/font/local";

export const displayFont = localFont({
  src: "./login/fonts/host-grotesk.ttf",
  variable: "--font-login-display",
  weight: "300 800",
  display: "swap",
});
export const bodyFont = localFont({
  src: "./login/fonts/instrument-sans.ttf",
  variable: "--font-login-body",
  weight: "400 700",
  display: "swap",
});
