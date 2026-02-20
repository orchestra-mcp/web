import { Link, usePage } from '@inertiajs/react';
import { Bell } from 'lucide-react';
import { Button } from '@/components/ui/button';

export function NotificationBell() {
    const { auth } = usePage().props;
    const notifications = auth.notifications ?? [];
    const unreadCount = notifications.filter((n) => !n.read_at).length;

    return (
        <Button variant="ghost" size="icon" className="relative" asChild>
            <Link href="/notifications">
                <Bell className="size-5 opacity-80 group-hover:opacity-100" />
                {unreadCount > 0 && (
                    <span className="absolute top-1 right-1 flex size-4 items-center justify-center rounded-full bg-destructive text-[10px] font-medium text-white">
                        {unreadCount > 9 ? '9+' : unreadCount}
                    </span>
                )}
                <span className="sr-only">
                    Notifications{unreadCount > 0 ? ` (${unreadCount} unread)` : ''}
                </span>
            </Link>
        </Button>
    );
}
