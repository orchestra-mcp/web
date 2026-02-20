import { Head, Link, router } from '@inertiajs/react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import AppLayout from '@/layouts/app-layout';
import type { BreadcrumbItem, Role } from '@/types';

const breadcrumbs: BreadcrumbItem[] = [
    { title: 'Admin', href: '/admin' },
    { title: 'Roles', href: '/admin/roles' },
];

const PROTECTED_ROLES = ['super_admin', 'admin', 'user'];

type Props = {
    roles: Role[];
};

export default function RolesIndex({ roles }: Props) {
    function handleDelete(role: Role) {
        if (!confirm(`Are you sure you want to delete the "${role.name}" role?`)) {
            return;
        }

        router.post(`/admin/roles/${role.id}`, {
            _method: 'DELETE',
        });
    }

    return (
        <AppLayout breadcrumbs={breadcrumbs}>
            <Head title="Roles" />

            <div className="flex h-full flex-1 flex-col gap-4 rounded-xl p-4">
                <Card>
                    <CardHeader>
                        <div className="flex items-center justify-between">
                            <div>
                                <CardTitle>Roles</CardTitle>
                                <CardDescription>
                                    Manage user roles and their permissions
                                </CardDescription>
                            </div>
                            <Button asChild>
                                <Link href="/admin/roles/create">Create Role</Link>
                            </Button>
                        </div>
                    </CardHeader>
                    <CardContent>
                        <div className="overflow-x-auto">
                            <table className="w-full text-sm">
                                <thead>
                                    <tr className="border-b text-left">
                                        <th className="pb-3 pr-4 font-medium text-muted-foreground">
                                            Name
                                        </th>
                                        <th className="pb-3 pr-4 font-medium text-muted-foreground">
                                            Permissions
                                        </th>
                                        <th className="pb-3 pr-4 font-medium text-muted-foreground">
                                            Users
                                        </th>
                                        <th className="pb-3 text-right font-medium text-muted-foreground">
                                            Actions
                                        </th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {roles.map((role) => (
                                        <tr
                                            key={role.id}
                                            className="border-b last:border-0"
                                        >
                                            <td className="py-3 pr-4">
                                                <div className="flex items-center gap-2">
                                                    <span className="font-medium">
                                                        {role.name}
                                                    </span>
                                                    {PROTECTED_ROLES.includes(role.name) && (
                                                        <Badge variant="secondary">
                                                            Protected
                                                        </Badge>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="py-3 pr-4">
                                                <Badge variant="outline">
                                                    {role.permissions_count ?? 0}
                                                </Badge>
                                            </td>
                                            <td className="py-3 pr-4">
                                                <Badge variant="outline">
                                                    {role.users_count ?? 0}
                                                </Badge>
                                            </td>
                                            <td className="py-3 text-right">
                                                <div className="flex items-center justify-end gap-2">
                                                    <Button
                                                        variant="outline"
                                                        size="sm"
                                                        asChild
                                                    >
                                                        <Link
                                                            href={`/admin/roles/${role.id}/edit`}
                                                        >
                                                            Edit
                                                        </Link>
                                                    </Button>
                                                    <Button
                                                        variant="destructive"
                                                        size="sm"
                                                        disabled={PROTECTED_ROLES.includes(
                                                            role.name,
                                                        )}
                                                        onClick={() =>
                                                            handleDelete(role)
                                                        }
                                                    >
                                                        Delete
                                                    </Button>
                                                </div>
                                            </td>
                                        </tr>
                                    ))}
                                    {roles.length === 0 && (
                                        <tr>
                                            <td
                                                colSpan={4}
                                                className="py-6 text-center text-muted-foreground"
                                            >
                                                No roles found. Create one to get
                                                started.
                                            </td>
                                        </tr>
                                    )}
                                </tbody>
                            </table>
                        </div>
                    </CardContent>
                </Card>
            </div>
        </AppLayout>
    );
}
