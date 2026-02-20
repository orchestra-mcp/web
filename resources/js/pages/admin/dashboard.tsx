import { Head } from '@inertiajs/react';
import {
    AlertTriangle,
    Clock,
    CreditCard,
    TrendingDown,
    TrendingUp,
    UserCheck,
    UserX,
    Users,
} from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import AppLayout from '@/layouts/app-layout';
import type { BreadcrumbItem } from '@/types';
import type { LucideIcon } from 'lucide-react';

type AdminStats = {
    total_users: number;
    active_users: number;
    blocked_users: number;
    active_subscriptions: number;
    expiring_soon: number;
    unpaid: number;
};

type StatCardDef = {
    label: string;
    value: number;
    icon: LucideIcon;
    trend?: 'up' | 'down' | 'neutral';
    trendLabel?: string;
    accent?: boolean;
};

const breadcrumbs: BreadcrumbItem[] = [
    { title: 'Admin', href: '/admin' },
    { title: 'Dashboard', href: '/admin' },
];

function TrendIndicator({ trend, label }: { trend: 'up' | 'down' | 'neutral'; label?: string }) {
    if (trend === 'neutral') {
        return (
            <span className="inline-flex items-center gap-1 text-xs text-muted-foreground">
                {label ?? 'No change'}
            </span>
        );
    }

    const isUp = trend === 'up';
    const Icon = isUp ? TrendingUp : TrendingDown;

    return (
        <span
            className={`inline-flex items-center gap-1 text-xs font-medium ${
                isUp
                    ? 'text-emerald-600 dark:text-emerald-400'
                    : 'text-red-500 dark:text-red-400'
            }`}
        >
            <Icon className="size-3.5" />
            {label}
        </span>
    );
}

export default function AdminDashboard({ stats }: { stats: AdminStats }) {
    const cards: StatCardDef[] = [
        {
            label: 'Total Users',
            value: stats.total_users,
            icon: Users,
            trend: 'up',
            trendLabel: 'Growing',
        },
        {
            label: 'Active Users',
            value: stats.active_users,
            icon: UserCheck,
            trend: 'up',
            trendLabel: 'Healthy',
        },
        {
            label: 'Blocked Users',
            value: stats.blocked_users,
            icon: UserX,
            trend: stats.blocked_users > 0 ? 'down' : 'neutral',
            trendLabel: stats.blocked_users > 0 ? `${stats.blocked_users} blocked` : 'None',
            accent: stats.blocked_users > 0,
        },
        {
            label: 'Active Subscriptions',
            value: stats.active_subscriptions,
            icon: CreditCard,
            trend: 'up',
            trendLabel: 'Stable',
        },
        {
            label: 'Expiring Soon',
            value: stats.expiring_soon,
            icon: Clock,
            trend: stats.expiring_soon > 0 ? 'down' : 'neutral',
            trendLabel: stats.expiring_soon > 0 ? 'Needs attention' : 'All good',
            accent: stats.expiring_soon > 0,
        },
        {
            label: 'Unpaid',
            value: stats.unpaid,
            icon: AlertTriangle,
            trend: stats.unpaid > 0 ? 'down' : 'neutral',
            trendLabel: stats.unpaid > 0 ? 'Action required' : 'Clear',
            accent: stats.unpaid > 0,
        },
    ];

    return (
        <AppLayout breadcrumbs={breadcrumbs}>
            <Head title="Admin Dashboard" />

            <div className="flex flex-1 flex-col gap-6 p-4 md:p-6">
                <div className="flex items-center gap-3">
                    <h1 className="text-2xl font-semibold tracking-tight">Admin Dashboard</h1>
                    <span className="brand-gradient-text text-sm font-medium">Overview</span>
                </div>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                    {cards.map((card) => (
                        <Card
                            key={card.label}
                            className="group relative overflow-hidden transition-shadow hover:shadow-md"
                        >
                            {/* Accent border for flagged cards */}
                            {card.accent && (
                                <div className="absolute inset-y-0 left-0 w-[3px] bg-gradient-to-b from-[var(--brand-cyan)] to-[var(--brand-purple)]" />
                            )}

                            <CardHeader className="flex flex-row items-center justify-between pb-2">
                                <CardTitle className="text-sm font-medium text-muted-foreground">
                                    {card.label}
                                </CardTitle>
                                <div
                                    className={`flex size-9 items-center justify-center rounded-lg transition-colors ${
                                        card.accent
                                            ? 'bg-[var(--brand-purple)]/10 dark:bg-[var(--brand-purple)]/20'
                                            : 'bg-muted'
                                    }`}
                                >
                                    <card.icon
                                        className={`size-4 ${
                                            card.accent
                                                ? 'text-[var(--brand-purple)]'
                                                : 'text-muted-foreground/70'
                                        }`}
                                    />
                                </div>
                            </CardHeader>

                            <CardContent className="space-y-1">
                                <p className="text-3xl font-bold tracking-tight">
                                    {card.value.toLocaleString()}
                                </p>
                                {card.trend && (
                                    <TrendIndicator
                                        trend={card.trend}
                                        label={card.trendLabel}
                                    />
                                )}
                            </CardContent>
                        </Card>
                    ))}
                </div>
            </div>
        </AppLayout>
    );
}
