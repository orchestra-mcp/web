import { Head, Link, router, useForm } from '@inertiajs/react';
import type { FormEventHandler } from 'react';
import InputError from '@/components/input-error';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import AppLayout from '@/layouts/app-layout';
import type { BreadcrumbItem, Permission, Role } from '@/types';

const breadcrumbs: BreadcrumbItem[] = [
    { title: 'Admin', href: '/admin' },
    { title: 'Roles', href: '/admin/roles' },
    { title: 'Edit Role', href: '#' },
];

type Props = {
    role: Role;
    permissions: Permission[];
};

export default function RolesEdit({ role, permissions }: Props) {
    const { data, setData, processing, errors } = useForm({
        name: role.name,
        permissions: (role.permissions ?? []).map((p) => p.id),
    });

    function togglePermission(permissionId: number) {
        setData(
            'permissions',
            data.permissions.includes(permissionId)
                ? data.permissions.filter((id) => id !== permissionId)
                : [...data.permissions, permissionId],
        );
    }

    const submit: FormEventHandler = (e) => {
        e.preventDefault();
        router.post(`/admin/roles/${role.id}`, {
            _method: 'PUT',
            ...data,
        });
    };

    return (
        <AppLayout breadcrumbs={breadcrumbs}>
            <Head title={`Edit Role: ${role.name}`} />

            <div className="flex h-full flex-1 flex-col gap-4 rounded-xl p-4">
                <form onSubmit={submit} className="space-y-6">
                    <Card>
                        <CardHeader>
                            <CardTitle>Edit Role</CardTitle>
                            <CardDescription>
                                Update the role name and assigned permissions
                            </CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-6">
                            <div className="grid gap-2">
                                <Label htmlFor="name">Role Name</Label>
                                <Input
                                    id="name"
                                    value={data.name}
                                    onChange={(e) =>
                                        setData('name', e.target.value)
                                    }
                                    placeholder="e.g. editor, moderator"
                                    required
                                    className="max-w-md"
                                />
                                <InputError message={errors.name} />
                            </div>

                            <div className="grid gap-3">
                                <Label>Permissions</Label>
                                <InputError message={errors.permissions} />
                                {permissions.length > 0 ? (
                                    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                                        {permissions.map((permission) => (
                                            <label
                                                key={permission.id}
                                                className="flex cursor-pointer items-center gap-2 rounded-md border p-3 transition-colors hover:bg-accent"
                                            >
                                                <Checkbox
                                                    checked={data.permissions.includes(
                                                        permission.id,
                                                    )}
                                                    onCheckedChange={() =>
                                                        togglePermission(
                                                            permission.id,
                                                        )
                                                    }
                                                />
                                                <span className="text-sm">
                                                    {permission.name}
                                                </span>
                                            </label>
                                        ))}
                                    </div>
                                ) : (
                                    <p className="text-sm text-muted-foreground">
                                        No permissions available.
                                    </p>
                                )}
                            </div>
                        </CardContent>
                    </Card>

                    <div className="flex items-center gap-4">
                        <Button disabled={processing}>Update Role</Button>
                        <Button variant="outline" asChild>
                            <Link href="/admin/roles">Cancel</Link>
                        </Button>
                    </div>
                </form>
            </div>
        </AppLayout>
    );
}
