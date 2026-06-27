import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import Link from "next/link";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "Realtor Agent Scraper Dashboard",
  description: "Internal job dashboard and data analytics panel for Realtor.com agent scraper.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="h-full bg-slate-900 text-slate-100 antialiased">
      <body className={`${inter.className} h-full flex`}>
        {/* Sidebar Container */}
        <aside className="w-64 bg-slate-950 border-r border-slate-800 flex flex-col flex-shrink-0">
          {/* Logo / Header */}
          <div className="p-6 border-b border-slate-800">
            <h1 className="text-xl font-bold bg-gradient-to-r from-blue-400 to-indigo-400 bg-clip-text text-transparent">
              Realtor Scraper
            </h1>
            <p className="text-xs text-slate-500 mt-1 font-mono">v1.1.0-release</p>
          </div>

          {/* Navigation Links */}
          <nav className="flex-1 p-4 space-y-1.5">
            <Link
              href="/"
              className="flex items-center gap-3 px-4 py-3 rounded-lg text-slate-300 hover:bg-slate-900 hover:text-white transition-all font-medium text-sm"
            >
              <span>📊</span> Overview
            </Link>
            <Link
              href="/jobs/new"
              className="flex items-center gap-3 px-4 py-3 rounded-lg text-slate-300 hover:bg-slate-900 hover:text-white transition-all font-medium text-sm"
            >
              <span>➕</span> New Scrape Job
            </Link>
            <Link
              href="/jobs"
              className="flex items-center gap-3 px-4 py-3 rounded-lg text-slate-300 hover:bg-slate-900 hover:text-white transition-all font-medium text-sm"
            >
              <span>⚙️</span> Jobs List
            </Link>
            <Link
              href="/agents"
              className="flex items-center gap-3 px-4 py-3 rounded-lg text-slate-300 hover:bg-slate-900 hover:text-white transition-all font-medium text-sm"
            >
              <span>👥</span> Agents List
            </Link>
            <Link
              href="/export"
              className="flex items-center gap-3 px-4 py-3 rounded-lg text-slate-300 hover:bg-slate-900 hover:text-white transition-all font-medium text-sm"
            >
              <span>📥</span> Export Panel
            </Link>
            <Link
              href="/proxies"
              className="flex items-center gap-3 px-4 py-3 rounded-lg text-slate-300 hover:bg-slate-900 hover:text-white transition-all font-medium text-sm"
            >
              <span>🔑</span> Proxies Panel
            </Link>
          </nav>

          {/* Compliance Notice Footer */}
          <div className="p-4 border-t border-slate-800 bg-slate-950">
            <div className="text-[10px] text-slate-500 leading-relaxed font-sans space-y-1.5">
              <p className="font-semibold text-slate-400">🛡️ COMPLIANCE NOTICE</p>
              <p>This software is for internal use only where authorized by legal rights.</p>
              <p>
                Phone, email, and MLS data are subject to state privacy, TCPA, CAN-SPAM laws, and platform terms of service.
              </p>
            </div>
          </div>
        </aside>

        {/* Main Content Area */}
        <main className="flex-1 flex flex-col min-w-0 overflow-y-auto bg-slate-900">
          {children}
        </main>
      </body>
    </html>
  );
}
