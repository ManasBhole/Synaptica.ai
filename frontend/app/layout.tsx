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
    <html lang="en" className="h-full bg-surface-subtle">
      <body className="h-full text-neutral-700">
        <Providers>
          <div className="flex min-h-screen">
            <Sidebar />
            <main className="flex-1 bg-surface-subtle">
              <Topbar />
              <div className="px-10 pb-16 pt-8">{children}</div>
            </main>
          </div>
        </Providers>
      </body>
    </html>
  );
}
