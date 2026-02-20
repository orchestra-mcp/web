import { Head } from '@inertiajs/react';
import {
    AlertTriangle,
    Clock,
    CreditCard,
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

type StatCard = {
    label: string;
    value: number;
    icon: LucideIcon;
};

const breadcrumbs: BreadcrumbItem[] = [
    { title: 'Admin', href: '/admin' },
    { title: 'Dashboard', href: '/admin' },
];

export default function AdminDashboard({ stats }: { stats: AdminStats }) {
    const cards: StatCard[] = [
        { label: 'Total Users', value: stats.total_users, icon: Users },
        { label: 'Active Users', value: stats.active_users, icon: UserCheck },
        { label: 'Blocked Users', value: stats.blocked_users, icon: UserX },
        {
            label: 'Active Subscriptions',
            value: stats.active_subscriptions,
            icon: CreditCard,
        },
        { label: 'Expiring Soon', value: stats.expiring_soon, icon: Clock },
        { label: 'Unpaid', value: stats.unpaid, icon: AlertTriangle },
    ];

    return (
        <AppLayout breadcrumbs={breadcrumbs}>
            <Head title="Admin Dashboard" />

            <div className="flex flex-1 flex-col gap-6 p-4 md:p-6">
                <h1 className="text-2xl font-semibold tracking-tight">
                    Admin Dashboard
                </h1>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                    {cards.map((card) => (
                        <Card key={card.label}>
                            <CardHeader className="flex flex-row items-center justify-between pb-2">
                                <CardTitle className="text-sm font-medium text-muted-foreground">
                                    {card.label}
                                </CardTitle>
                                <card.icon className="size-5 text-muted-foreground/70" />
                            </CardHeader>
                            <CardContent>
                                <p className="text-3xl font-bold tracking-tight">
                                    {card.value.toLocaleString()}
                                </p>
                            </CardContent>
                        </Card>
                    ))}
                </div>
            </div>
        </AppLayout>
    );
}
