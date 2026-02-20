import { Head, Link, router } from '@inertiajs/react';
import { Bell, Check, Inbox } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import AppLayout from '@/layouts/app-layout';
import type { BreadcrumbItem, Notification } from '@/types';
import { cn } from '@/lib/utils';

type PaginationLink = {
    url: string | null;
    label: string;
    active: boolean;
};

type PaginatedNotifications = {
    data: Notification[];
    links: PaginationLink[];
    current_page: number;
    last_page: number;
};

const breadcrumbs: BreadcrumbItem[] = [
    {
        title: 'Dashboard',
        href: '/dashboard',
    },
    {
        title: 'Notifications',
        href: '/notifications',
    },
];

function relativeTime(dateString: string): string {
    const now = new Date();
    const date = new Date(dateString);
    const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

    if (seconds < 60) {
        return 'just now';
    }

    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) {
        return `${minutes} minute${minutes !== 1 ? 's' : ''} ago`;
    }

    const hours = Math.floor(minutes / 60);
    if (hours < 24) {
        return `${hours} hour${hours !== 1 ? 's' : ''} ago`;
    }

    const days = Math.floor(hours / 24);
    if (days < 30) {
        return `${days} day${days !== 1 ? 's' : ''} ago`;
    }

    const months = Math.floor(days / 30);
    if (months < 12) {
        return `${months} month${months !== 1 ? 's' : ''} ago`;
    }

    const years = Math.floor(months / 12);
    return `${years} year${years !== 1 ? 's' : ''} ago`;
}

function markAsRead(id: string) {
    router.post(`/notifications/${id}/read`, {}, {
        preserveScroll: true,
    });
}

export default function Notifications({
    notifications,
}: {
    notifications: PaginatedNotifications;
}) {
    return (
        <AppLayout breadcrumbs={breadcrumbs}>
            <Head title="Notifications" />

            <div className="mx-auto w-full max-w-4xl space-y-6 p-4">
                <div className="flex items-center gap-3">
                    <Bell className="size-5 text-muted-foreground" />
                    <h1 className="text-xl font-semibold">Notifications</h1>
                </div>

                {notifications.data.length === 0 ? (
                    <Card>
                        <CardContent className="flex flex-col items-center justify-center py-12">
                            <Inbox className="size-12 text-muted-foreground/50" />
                            <p className="mt-4 text-sm text-muted-foreground">
                                You have no notifications.
                            </p>
                        </CardContent>
                    </Card>
                ) : (
                    <div className="space-y-2">
                        {notifications.data.map((notification) => (
                            <Card
                                key={notification.id}
                                className={cn(
                                    'transition-colors',
                                    !notification.read_at &&
                                        'border-blue-200 bg-blue-50/50 dark:border-blue-900 dark:bg-blue-950/30',
                                )}
                            >
                                <CardContent className="flex items-center gap-4 py-0">
                                    <div className="flex shrink-0 items-center justify-center">
                                        {!notification.read_at ? (
                                            <span className="size-2.5 rounded-full bg-blue-500" />
                                        ) : (
                                            <span className="size-2.5" />
                                        )}
                                    </div>

                                    <div className="min-w-0 flex-1">
                                        <p className="text-sm">
                                            {notification.data.message ??
                                                notification.data.title ??
                                                'Notification'}
                                        </p>
                                        <p className="mt-0.5 text-xs text-muted-foreground">
                                            {relativeTime(
                                                notification.created_at,
                                            )}
                                        </p>
                                    </div>

                                    {!notification.read_at && (
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={() =>
                                                markAsRead(notification.id)
                                            }
                                            className="shrink-0 text-xs text-muted-foreground"
                                        >
                                            <Check className="size-4" />
                                            Mark as read
                                        </Button>
                                    )}
                                </CardContent>
                            </Card>
                        ))}
                    </div>
                )}

                {notifications.last_page > 1 && (
                    <nav className="flex items-center justify-center gap-1">
                        {notifications.links.map((link, index) => (
                            <div key={index}>
                                {link.url ? (
                                    <Link
                                        href={link.url}
                                        preserveScroll
                                        className={cn(
                                            'inline-flex h-9 min-w-9 items-center justify-center rounded-md px-3 text-sm transition-colors',
                                            link.active
                                                ? 'bg-primary text-primary-foreground'
                                                : 'hover:bg-accent hover:text-accent-foreground',
                                        )}
                                        dangerouslySetInnerHTML={{
                                            __html: link.label,
                                        }}
                                    />
                                ) : (
                                    <span
                                        className="inline-flex h-9 min-w-9 items-center justify-center rounded-md px-3 text-sm text-muted-foreground opacity-50"
                                        dangerouslySetInnerHTML={{
                                            __html: link.label,
                                        }}
                                    />
                                )}
                            </div>
                        ))}
                    </nav>
                )}
            </div>
        </AppLayout>
    );
}
