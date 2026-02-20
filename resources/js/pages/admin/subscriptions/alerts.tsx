import { Head, Link } from '@inertiajs/react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import AppLayout from '@/layouts/app-layout';
import type { BreadcrumbItem, Subscription, User } from '@/types';

const breadcrumbs: BreadcrumbItem[] = [
    { title: 'Admin', href: '/admin' },
    { title: 'Subscription Alerts', href: '/admin/subscriptions-alerts' },
];

type SubscriptionWithUser = Subscription & { user: User };

type Props = {
    expiring: SubscriptionWithUser[];
    unpaid: SubscriptionWithUser[];
};

function formatDate(dateString: string | null): string {
    if (!dateString) return '--';
    return new Date(dateString).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
    });
}

function AlertTable({ subscriptions, emptyMessage }: { subscriptions: SubscriptionWithUser[]; emptyMessage: string }) {
    if (subscriptions.length === 0) {
        return (
            <p className="py-6 text-center text-sm text-muted-foreground">{emptyMessage}</p>
        );
    }

    return (
        <div className="overflow-hidden rounded-lg border">
            <table className="w-full text-sm">
                <thead className="border-b bg-muted/50">
                    <tr>
                        <th className="px-4 py-3 text-left font-medium text-muted-foreground">User</th>
                        <th className="px-4 py-3 text-left font-medium text-muted-foreground">Email</th>
                        <th className="px-4 py-3 text-left font-medium text-muted-foreground">Plan</th>
                        <th className="px-4 py-3 text-left font-medium text-muted-foreground">End Date</th>
                        <th className="px-4 py-3 text-left font-medium text-muted-foreground">Actions</th>
                    </tr>
                </thead>
                <tbody className="divide-y">
                    {subscriptions.map((subscription) => (
                        <tr key={subscription.id} className="hover:bg-muted/30">
                            <td className="px-4 py-3 font-medium">{subscription.user.name}</td>
                            <td className="px-4 py-3 text-muted-foreground">{subscription.user.email}</td>
                            <td className="px-4 py-3">
                                <Badge variant="outline">
                                    {subscription.plan.charAt(0).toUpperCase() + subscription.plan.slice(1)}
                                </Badge>
                            </td>
                            <td className="px-4 py-3 text-muted-foreground">{formatDate(subscription.end_date)}</td>
                            <td className="px-4 py-3">
                                <Button asChild variant="ghost" size="sm">
                                    <Link href={`/admin/subscriptions/${subscription.id}/edit`}>Edit</Link>
                                </Button>
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}

export default function SubscriptionAlerts({ expiring, unpaid }: Props) {
    return (
        <AppLayout breadcrumbs={breadcrumbs}>
            <Head title="Subscription Alerts" />

            <div className="flex h-full flex-1 flex-col gap-4 rounded-xl p-4">
                <Card>
                    <CardHeader>
                        <CardTitle>Expiring Soon</CardTitle>
                        <CardDescription>Subscriptions that will expire in the near future</CardDescription>
                    </CardHeader>
                    <CardContent>
                        <AlertTable
                            subscriptions={expiring}
                            emptyMessage="No alerts -- no subscriptions expiring soon."
                        />
                    </CardContent>
                </Card>

                <Card>
                    <CardHeader>
                        <CardTitle>Unpaid / Past Due</CardTitle>
                        <CardDescription>Subscriptions with outstanding payment issues</CardDescription>
                    </CardHeader>
                    <CardContent>
                        <AlertTable
                            subscriptions={unpaid}
                            emptyMessage="No alerts -- no unpaid or past due subscriptions."
                        />
                    </CardContent>
                </Card>
            </div>
        </AppLayout>
    );
}
