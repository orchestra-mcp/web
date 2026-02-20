import { Head, Link, router } from '@inertiajs/react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import AppLayout from '@/layouts/app-layout';
import type { BreadcrumbItem, Subscription, User } from '@/types';

const breadcrumbs: BreadcrumbItem[] = [
    { title: 'Admin', href: '/admin' },
    { title: 'Subscriptions', href: '/admin/subscriptions' },
];

type PaginationLink = {
    url: string | null;
    label: string;
    active: boolean;
};

type Props = {
    subscriptions: {
        data: (Subscription & { user: User })[];
        links: PaginationLink[];
        current_page: number;
        last_page: number;
    };
    filters: {
        status?: string;
    };
};

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
        month: 'short',
        day: 'numeric',
    });
}

function formatStatus(status: string): string {
    return status.replace('_', ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

export default function SubscriptionsIndex({ subscriptions, filters }: Props) {
    function handleFilterChange(status: string) {
        router.get(
            '/admin/subscriptions',
            { status: status === 'all' ? undefined : status },
            { preserveState: true },
        );
    }

    return (
        <AppLayout breadcrumbs={breadcrumbs}>
            <Head title="Subscriptions" />

            <div className="flex h-full flex-1 flex-col gap-4 rounded-xl p-4">
                <div className="flex items-center justify-between">
                    <h1 className="text-lg font-semibold">Subscriptions</h1>

                    <div className="w-48">
                        <Select
                            value={filters.status ?? 'all'}
                            onValueChange={handleFilterChange}
                        >
                            <SelectTrigger>
                                <SelectValue placeholder="Filter by status" />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="all">All Statuses</SelectItem>
                                <SelectItem value="active">Active</SelectItem>
                                <SelectItem value="expired">Expired</SelectItem>
                                <SelectItem value="cancelled">Cancelled</SelectItem>
                                <SelectItem value="past_due">Past Due</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>
                </div>

                <div className="overflow-hidden rounded-lg border">
                    <table className="w-full text-sm">
                        <thead className="border-b bg-muted/50">
                            <tr>
                                <th className="px-4 py-3 text-left font-medium text-muted-foreground">User</th>
                                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Email</th>
                                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Plan</th>
                                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Status</th>
                                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Start Date</th>
                                <th className="px-4 py-3 text-left font-medium text-muted-foreground">End Date</th>
                                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y">
                            {subscriptions.data.map((subscription) => (
                                <tr key={subscription.id} className="hover:bg-muted/30">
                                    <td className="px-4 py-3 font-medium">{subscription.user.name}</td>
                                    <td className="px-4 py-3 text-muted-foreground">{subscription.user.email}</td>
                                    <td className="px-4 py-3">
                                        <Badge variant="outline">
                                            {subscription.plan.charAt(0).toUpperCase() + subscription.plan.slice(1)}
                                        </Badge>
                                    </td>
                                    <td className="px-4 py-3">
                                        <Badge variant={statusBadgeVariant(subscription.status)}>
                                            {formatStatus(subscription.status)}
                                        </Badge>
                                    </td>
                                    <td className="px-4 py-3 text-muted-foreground">{formatDate(subscription.start_date)}</td>
                                    <td className="px-4 py-3 text-muted-foreground">{formatDate(subscription.end_date)}</td>
                                    <td className="px-4 py-3">
                                        <Button asChild variant="ghost" size="sm">
                                            <Link href={`/admin/subscriptions/${subscription.id}/edit`}>Edit</Link>
                                        </Button>
                                    </td>
                                </tr>
                            ))}
                            {subscriptions.data.length === 0 && (
                                <tr>
                                    <td colSpan={7} className="px-4 py-8 text-center text-muted-foreground">
                                        No subscriptions found.
                                    </td>
                                </tr>
                            )}
                        </tbody>
                    </table>
                </div>

                {subscriptions.last_page > 1 && (
                    <div className="flex items-center justify-center gap-1">
                        {subscriptions.links.map((link, index) => (
                            <Button
                                key={index}
                                variant={link.active ? 'default' : 'outline'}
                                size="sm"
                                disabled={!link.url}
                                asChild={!!link.url}
                            >
                                {link.url ? (
                                    <Link
                                        href={link.url}
                                        preserveState
                                        dangerouslySetInnerHTML={{ __html: link.label }}
                                    />
                                ) : (
                                    <span dangerouslySetInnerHTML={{ __html: link.label }} />
                                )}
                            </Button>
                        ))}
                    </div>
                )}
            </div>
        </AppLayout>
    );
}
