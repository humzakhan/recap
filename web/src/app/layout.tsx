import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "recap",
  description: "Extract structured data from any URL",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="antialiased">
        <nav className="border-b border-gray-200 px-6 py-3 flex items-center gap-6">
          <span className="font-bold text-lg">recap</span>
          <Link href="/" className="text-sm text-gray-600 hover:text-gray-900">Extract</Link>
          <Link href="/history" className="text-sm text-gray-600 hover:text-gray-900">History</Link>
          <Link href="/templates" className="text-sm text-gray-600 hover:text-gray-900">Templates</Link>
        </nav>
        {children}
      </body>
    </html>
  );
}
