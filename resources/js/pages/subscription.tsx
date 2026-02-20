import { Head } from '@inertiajs/react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import AppLayout from '@/layouts/app-layout';
import type { BreadcrumbItem, Subscription } from '@/types';

const GITHUB_SPONSORS_URL = 'https://github.com/sponsors/fadymondy';

const breadcrumbs: BreadcrumbItem[] = [
    { title: 'Dashboard', href: '/dashboard' },
    { title: 'Subscription', href: '/subscription' },
];

type Props = {
    subscription: Subscription | null;
};

function formatPlanName(plan: Subscription['plan']): string {
    switch (plan) {
        case 'free':
            return 'Free';
        case 'sponsor':
            return 'Sponsor';
        case 'team_sponsor':
            return 'Team Sponsor';
    }
}

function planBadgeVariant(plan: Subscription['plan']) {
    switch (plan) {
        case 'free':
            return 'secondary';
        case 'sponsor':
            return 'default';
        case 'team_sponsor':
            return 'outline';
    }
}

function statusBadgeVariant(status: Subscription['status']) {
    switch (status) {
        case 'active':
            return 'default';
        case 'expired':
        case 'past_due':
            return 'destructive';
        case 'cancelled':
            return 'secondary';
    }
}

function formatDate(dateString: string | null): string {
    if (!dateString) return '--';
    return new Date(dateString).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
    });
}

function formatAmount(cents: number): string {
    return `$${(cents / 100).toFixed(2)}`;
}

export default function SubscriptionPage({ subscription }: Props) {
    const isSponsor = subscription && ['sponsor', 'team_sponsor'].includes(subscription.plan) && subscription.status === 'active';

    return (
        <AppLayout breadcrumbs={breadcrumbs}>
            <Head title="Subscription" />

            <div className="flex h-full flex-1 flex-col gap-4 rounded-xl p-4">
                {subscription && subscription.plan !== 'free' ? (
                    <Card>
                        <CardHeader>
                            <div className="flex items-center gap-3">
                                <CardTitle>Subscription</CardTitle>
                                <Badge variant={planBadgeVariant(subscription.plan)}>
                                    {formatPlanName(subscription.plan)}
                                </Badge>
                                <Badge variant={statusBadgeVariant(subscription.status)}>
                                    {subscription.status.replace('_', ' ').replace(/\b\w/g, (c) => c.toUpperCase())}
                                </Badge>
                            </div>
                            <CardDescription>
                                {isSponsor
                                    ? 'Thank you for sponsoring Orchestra MCP!'
                                    : 'Your current subscription details'}
                            </CardDescription>
                        </CardHeader>
                        <CardContent>
                            <dl className="grid gap-4 sm:grid-cols-2">
                                <div>
                                    <dt className="text-sm font-medium text-muted-foreground">Start Date</dt>
                                    <dd className="mt-1 text-sm">{formatDate(subscription.start_date)}</dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-muted-foreground">End Date</dt>
                                    <dd className="mt-1 text-sm">{formatDate(subscription.end_date)}</dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-muted-foreground">Amount</dt>
                                    <dd className="mt-1 text-sm">{formatAmount(subscription.amount_cents)}/month</dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-muted-foreground">Via</dt>
                                    <dd className="mt-1 text-sm">GitHub Sponsors</dd>
                                </div>
                            </dl>
                        </CardContent>
                    </Card>
                ) : (
                    <Card>
                        <CardHeader>
                            <CardTitle>Subscription</CardTitle>
                            <CardDescription>You are on the Free plan</CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            <p className="text-sm text-muted-foreground">
                                Sponsor Orchestra MCP on GitHub to unlock cloud AI, unlimited synced devices, RAG memory, and all 26 themes.
                                After sponsoring, an admin will activate your account.
                            </p>
                            <Button asChild className="brand-gradient gap-2 border-0 text-white hover:opacity-90">
                                <a href={GITHUB_SPONSORS_URL} target="_blank" rel="noopener noreferrer">
                                    <i className="bx bxl-github" />
                                    Sponsor on GitHub
                                </a>
                            </Button>
                        </CardContent>
                    </Card>
                )}
            </div>
        </AppLayout>
    );
}
