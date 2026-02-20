import { Head } from '@inertiajs/react';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import AppLayout from '@/layouts/app-layout';
import type { BreadcrumbItem, Subscription } from '@/types';

const breadcrumbs: BreadcrumbItem[] = [
    { title: 'Dashboard', href: '/dashboard' },
    { title: 'Subscription', href: '/subscription' },
];

type Props = {
    subscription: Subscription | null;
};

function planBadgeVariant(plan: Subscription['plan']) {
    switch (plan) {
        case 'free':
            return 'secondary';
        case 'pro':
            return 'default';
        case 'team':
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
    return (
        <AppLayout breadcrumbs={breadcrumbs}>
            <Head title="Subscription" />

            <div className="flex h-full flex-1 flex-col gap-4 rounded-xl p-4">
                {subscription ? (
                    <Card>
                        <CardHeader>
                            <div className="flex items-center gap-3">
                                <CardTitle>Subscription</CardTitle>
                                <Badge variant={planBadgeVariant(subscription.plan)}>
                                    {subscription.plan.charAt(0).toUpperCase() + subscription.plan.slice(1)}
                                </Badge>
                                <Badge variant={statusBadgeVariant(subscription.status)}>
                                    {subscription.status.replace('_', ' ').replace(/\b\w/g, (c) => c.toUpperCase())}
                                </Badge>
                            </div>
                            <CardDescription>Your current subscription details</CardDescription>
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
                                    <dd className="mt-1 text-sm">{formatAmount(subscription.amount_cents)}</dd>
                                </div>
                            </dl>
                        </CardContent>
                    </Card>
                ) : (
                    <Card>
                        <CardHeader>
                            <CardTitle>Subscription</CardTitle>
                            <CardDescription>No active subscription</CardDescription>
                        </CardHeader>
                        <CardContent>
                            <p className="text-sm text-muted-foreground">
                                You are currently on the free tier. Upgrade to unlock additional features and higher limits.
                            </p>
                        </CardContent>
                    </Card>
                )}
            </div>
        </AppLayout>
    );
}
