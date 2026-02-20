import { Head, Link, usePage } from '@inertiajs/react';
import {
    Activity,
    ArrowRight,
    Bell,
    BookOpen,
    CreditCard,
    KeyRound,
    Settings,
    Sparkles,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from '@/components/ui/card';
import AppLayout from '@/layouts/app-layout';
import type { BreadcrumbItem } from '@/types';
import { dashboard, subscription } from '@/routes';
import { edit as profileEdit } from '@/routes/profile';

const breadcrumbs: BreadcrumbItem[] = [
    {
        title: 'Dashboard',
        href: dashboard().url,
    },
];

function StatCard({
    icon: Icon,
    label,
    value,
    hint,
}: {
    icon: typeof CreditCard;
    label: string;
    value: string;
    hint?: string;
}) {
    return (
        <Card className="group relative overflow-hidden py-5">
            {/* Gradient top accent line */}
            <div className="brand-gradient absolute inset-x-0 top-0 h-[2px] opacity-60 transition-opacity group-hover:opacity-100" />
            <CardHeader className="flex flex-row items-center gap-3 pb-1">
                <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-[var(--brand-purple)]/10 dark:bg-[var(--brand-purple)]/20">
                    <Icon className="size-4 text-[var(--brand-purple)]" />
                </div>
                <div className="min-w-0">
                    <CardDescription className="text-xs">{label}</CardDescription>
                    <p className="truncate text-lg font-semibold tracking-tight">{value}</p>
                </div>
            </CardHeader>
            {hint && (
                <CardContent className="pt-0">
                    <p className="text-xs text-muted-foreground">{hint}</p>
                </CardContent>
            )}
        </Card>
    );
}

function QuickAction({
    icon: Icon,
    label,
    description,
    href,
}: {
    icon: typeof Settings;
    label: string;
    description: string;
    href: string;
}) {
    return (
        <Link
            href={href}
            className="group flex items-center gap-4 rounded-lg border border-border/50 bg-card/50 p-4 transition-all hover:border-[var(--brand-purple)]/30 hover:bg-accent/50 hover:shadow-sm"
        >
            <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted transition-colors group-hover:bg-[var(--brand-purple)]/10 dark:group-hover:bg-[var(--brand-purple)]/20">
                <Icon className="size-5 text-muted-foreground transition-colors group-hover:text-[var(--brand-purple)]" />
            </div>
            <div className="min-w-0 flex-1">
                <p className="text-sm font-medium">{label}</p>
                <p className="text-xs text-muted-foreground">{description}</p>
            </div>
            <ArrowRight className="size-4 text-muted-foreground/50 transition-transform group-hover:translate-x-0.5 group-hover:text-[var(--brand-purple)]" />
        </Link>
    );
}

export default function Dashboard() {
    const { auth } = usePage().props;
    const user = auth.user;
    const notificationCount = auth.notifications?.length ?? 0;
    const planLabel = (() => {
        const plan = user.subscription?.plan;
        if (!plan || plan === 'free') return 'Free';
        if (plan === 'sponsor') return 'Sponsor';
        if (plan === 'team_sponsor') return 'Team Sponsor';
        return 'Free';
    })();
    const subStatus = user.subscription?.status ?? 'active';

    return (
        <AppLayout breadcrumbs={breadcrumbs}>
            <Head title="Dashboard" />
            <div className="flex h-full flex-1 flex-col gap-6 overflow-x-auto p-4 md:p-6">
                {/* ── Welcome Card ── */}
                <Card className="relative overflow-hidden py-0">
                    {/* Background decorative gradient blob */}
                    <div className="pointer-events-none absolute -right-20 -top-20 size-64 rounded-full bg-[var(--brand-purple)]/5 blur-3xl dark:bg-[var(--brand-purple)]/10" />
                    <div className="pointer-events-none absolute -bottom-16 -left-16 size-48 rounded-full bg-[var(--brand-cyan)]/5 blur-3xl dark:bg-[var(--brand-cyan)]/10" />

                    <CardContent className="relative flex flex-col gap-3 py-6 sm:flex-row sm:items-center sm:justify-between">
                        <div className="space-y-1.5">
                            <div className="flex items-center gap-2">
                                <Sparkles className="size-5 text-[var(--brand-purple)]" />
                                <h2 className="text-xl font-semibold tracking-tight">
                                    Welcome back, {user.name.split(' ')[0]}
                                </h2>
                            </div>
                            <p className="text-sm text-muted-foreground">
                                Here is what is happening with your Orchestra MCP workspace.
                            </p>
                        </div>
                        {/* Brand accent divider (vertical on desktop, horizontal on mobile) */}
                        <div className="brand-gradient hidden h-10 w-[2px] rounded-full opacity-40 sm:block" />
                        <div className="brand-gradient block h-[2px] w-full rounded-full opacity-40 sm:hidden" />
                        <div className="text-right">
                            <p className="brand-gradient-text text-2xl font-bold">{planLabel}</p>
                            <p className="text-xs capitalize text-muted-foreground">{subStatus} plan</p>
                        </div>
                    </CardContent>
                </Card>

                {/* ── Quick Stats ── */}
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                    <StatCard
                        icon={CreditCard}
                        label="Subscription"
                        value={planLabel}
                        hint={subStatus === 'active' ? 'Your plan is active' : `Status: ${subStatus}`}
                    />
                    <StatCard
                        icon={Bell}
                        label="Notifications"
                        value={notificationCount.toString()}
                        hint={notificationCount > 0 ? `${notificationCount} unread` : 'All caught up'}
                    />
                    <StatCard
                        icon={Activity}
                        label="Recent Activity"
                        value="--"
                        hint="Activity tracking coming soon"
                    />
                </div>

                {/* ── Activity Feed + Quick Actions ── */}
                <div className="grid gap-6 lg:grid-cols-5">
                    {/* Recent Activity Feed */}
                    <Card className="lg:col-span-3">
                        <CardHeader>
                            <CardTitle className="text-base">Recent Activity</CardTitle>
                            <CardDescription>Your latest workspace events</CardDescription>
                        </CardHeader>
                        <CardContent>
                            <div className="flex min-h-[200px] flex-col items-center justify-center rounded-lg border border-dashed border-border/60 py-12 text-center">
                                <Activity className="mb-3 size-8 text-muted-foreground/40" />
                                <p className="text-sm font-medium text-muted-foreground">No recent activity</p>
                                <p className="mt-1 text-xs text-muted-foreground/70">
                                    Your activity will appear here as you use Orchestra MCP.
                                </p>
                            </div>
                        </CardContent>
                    </Card>

                    {/* Quick Actions */}
                    <div className="flex flex-col gap-4 lg:col-span-2">
                        <h3 className="text-sm font-medium text-muted-foreground">Quick Actions</h3>
                        <div className="flex flex-col gap-3">
                            <QuickAction
                                icon={Settings}
                                label="Settings"
                                description="Manage your profile and preferences"
                                href={profileEdit().url}
                            />
                            <QuickAction
                                icon={KeyRound}
                                label="Subscription"
                                description="View plan details and billing"
                                href={subscription().url}
                            />
                            <QuickAction
                                icon={BookOpen}
                                label="Documentation"
                                description="Explore guides and API references"
                                href="https://docs.orchestra-mcp.com"
                            />
                        </div>
                    </div>
                </div>
            </div>
        </AppLayout>
    );
}
