import { Link } from '@inertiajs/react';
import AppLogoIcon from '@/components/app-logo-icon';
import type { AuthLayoutProps } from '@/types';
import { home } from '@/routes';

export default function AuthSimpleLayout({
    children,
    title,
    description,
}: AuthLayoutProps) {
    return (
        <div className="relative flex min-h-svh flex-col items-center justify-center gap-6 overflow-hidden bg-background p-6 md:p-10">
            {/* ── Gradient mesh background ── */}
            <div className="pointer-events-none absolute inset-0 overflow-hidden">
                {/* Top-right cyan blob */}
                <div className="absolute -right-32 -top-32 size-96 rounded-full bg-[var(--brand-cyan)]/8 blur-[100px] dark:bg-[var(--brand-cyan)]/15" />
                {/* Bottom-left purple blob */}
                <div className="absolute -bottom-32 -left-32 size-96 rounded-full bg-[var(--brand-purple)]/8 blur-[100px] dark:bg-[var(--brand-purple)]/15" />
                {/* Center subtle mix */}
                <div className="absolute left-1/2 top-1/3 size-72 -translate-x-1/2 rounded-full bg-[var(--brand-purple)]/4 blur-[120px] dark:bg-[var(--brand-purple)]/8" />
            </div>

            {/* ── Form container with glass effect ── */}
            <div className="relative w-full max-w-sm">
                <div className="glass-card flex flex-col gap-8 rounded-2xl border border-border/40 p-8 shadow-lg dark:border-border/20">
                    {/* ── Logo + Brand Name ── */}
                    <div className="flex flex-col items-center gap-4">
                        <Link
                            href={home()}
                            className="flex flex-col items-center gap-3 font-medium"
                        >
                            <div className="flex size-12 items-center justify-center">
                                <AppLogoIcon className="size-12" />
                            </div>
                            <span className="brand-gradient-text text-lg font-semibold tracking-tight">
                                Orchestra MCP
                            </span>
                        </Link>

                        {/* ── Title + Description ── */}
                        <div className="space-y-2 text-center">
                            <h1 className="text-xl font-medium">{title}</h1>
                            <p className="text-center text-sm text-muted-foreground">
                                {description}
                            </p>
                        </div>
                    </div>

                    {/* ── Gradient divider ── */}
                    <div className="brand-gradient mx-auto h-[1px] w-16 rounded-full opacity-40" />

                    {/* ── Form content ── */}
                    {children}
                </div>
            </div>
        </div>
    );
}
