import "./globals.css";

import type { Metadata } from "next";
import { ReactNode } from "react";
import { Providers } from "../components/providers";
import { Sidebar } from "../components/sidebar";
import { Topbar } from "../components/topbar";

export const metadata: Metadata = {
  title: "Synaptica Platform",
  description: "Unified health data intelligence platform"
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" className="h-full bg-slate-950">
      <body className="h-full text-slate-100">
        <Providers>
          <div className="flex min-h-screen">
            <Sidebar />
            <main className="flex-1 bg-gradient-to-b from-surface-raised/40 via-transparent to-transparent">
              <Topbar />
              <div className="px-8 pb-16 pt-8">{children}</div>
            </main>
          </div>
        </Providers>
      </body>
    </html>
  );
}
