import { Head, router, useForm } from '@inertiajs/react';
import type { FormEventHandler } from 'react';
import InputError from '@/components/input-error';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import AppLayout from '@/layouts/app-layout';
import type { BreadcrumbItem, Role, User } from '@/types';

const breadcrumbs: BreadcrumbItem[] = [
    { title: 'Admin', href: '/admin' },
    { title: 'Users', href: '/admin/users' },
    { title: 'Edit User', href: '#' },
];

export default function EditUser({
    user,
    roles,
    avatar_url,
}: {
    user: User;
    roles: Role[];
    avatar_url: string | null;
}) {
    const currentRoleId = user.roles?.[0]?.id ?? '';

    const { data, setData, processing, errors } = useForm<{
        name: string;
        email: string;
        role: string;
        avatar: File | null;
        _method: string;
    }>({
        name: user.name,
        email: user.email,
        role: String(currentRoleId),
        avatar: null,
        _method: 'PUT',
    });

    const submit: FormEventHandler = (e) => {
        e.preventDefault();

        router.post(`/admin/users/${user.id}`, {
            ...data,
            _method: 'PUT',
        }, {
            forceFormData: true,
        });
    };

    function toggleStatus() {
        router.post(`/admin/users/${user.id}/toggle-status`, {}, {
            preserveState: true,
        });
    }

    return (
        <AppLayout breadcrumbs={breadcrumbs}>
            <Head title="Edit User" />

            <div className="flex flex-1 flex-col gap-6 p-4 md:p-6">
                <div className="flex items-center justify-between">
                    <h1 className="text-2xl font-semibold tracking-tight">
                        Edit User
                    </h1>
                    <Button
                        variant={
                            user.status === 'active'
                                ? 'destructive'
                                : 'default'
                        }
                        onClick={toggleStatus}
                    >
                        {user.status === 'active'
                            ? 'Block User'
                            : 'Activate User'}
                    </Button>
                </div>

                {/* Edit Form */}
                <form
                    onSubmit={submit}
                    className="max-w-2xl space-y-6"
                    encType="multipart/form-data"
                >
                    <div className="grid gap-2">
                        <Label htmlFor="name">Name</Label>
                        <Input
                            id="name"
                            value={data.name}
                            onChange={(e) => setData('name', e.target.value)}
                            required
                            autoComplete="name"
                            placeholder="Full name"
                        />
                        <InputError message={errors.name} />
                    </div>

                    <div className="grid gap-2">
                        <Label htmlFor="email">Email</Label>
                        <Input
                            id="email"
                            type="email"
                            value={data.email}
                            onChange={(e) => setData('email', e.target.value)}
                            required
                            autoComplete="email"
                            placeholder="Email address"
                        />
                        <InputError message={errors.email} />
                    </div>

                    <div className="grid gap-2">
                        <Label htmlFor="role">Role</Label>
                        <Select
                            value={data.role}
                            onValueChange={(value) => setData('role', value)}
                        >
                            <SelectTrigger id="role">
                                <SelectValue placeholder="Select a role" />
                            </SelectTrigger>
                            <SelectContent>
                                {roles.map((role) => (
                                    <SelectItem
                                        key={role.id}
                                        value={String(role.id)}
                                    >
                                        {role.name}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        <InputError message={errors.role} />
                    </div>

                    <div className="grid gap-2">
                        <Label htmlFor="avatar">Avatar</Label>
                        {avatar_url && (
                            <div className="mb-2">
                                <img
                                    src={avatar_url}
                                    alt={`${user.name}'s avatar`}
                                    className="size-20 rounded-full border object-cover"
                                />
                            </div>
                        )}
                        <Input
                            id="avatar"
                            type="file"
                            accept="image/*"
                            onChange={(e) =>
                                setData(
                                    'avatar',
                                    e.target.files?.[0] ?? null,
                                )
                            }
                        />
                        <InputError message={errors.avatar} />
                    </div>

                    <div className="flex items-center gap-4">
                        <Button type="submit" disabled={processing}>
                            Save Changes
                        </Button>
                    </div>
                </form>

                <Separator className="my-2 max-w-2xl" />

                {/* Subscription Info */}
                {user.subscription && (
                    <Card className="max-w-2xl">
                        <CardHeader>
                            <CardTitle>Subscription</CardTitle>
                        </CardHeader>
                        <CardContent>
                            <dl className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm">
                                <dt className="font-medium text-muted-foreground">
                                    Plan
                                </dt>
                                <dd className="capitalize">
                                    {user.subscription.plan}
                                </dd>

                                <dt className="font-medium text-muted-foreground">
                                    Status
                                </dt>
                                <dd>
                                    <Badge
                                        variant={
                                            user.subscription.status ===
                                            'active'
                                                ? 'default'
                                                : 'destructive'
                                        }
                                        className={
                                            user.subscription.status ===
                                            'active'
                                                ? 'bg-green-600 hover:bg-green-600/90'
                                                : ''
                                        }
                                    >
                                        {user.subscription.status}
                                    </Badge>
                                </dd>

                                <dt className="font-medium text-muted-foreground">
                                    Start Date
                                </dt>
                                <dd>
                                    {user.subscription.start_date
                                        ? new Date(
                                              user.subscription.start_date,
                                          ).toLocaleDateString()
                                        : '--'}
                                </dd>

                                <dt className="font-medium text-muted-foreground">
                                    End Date
                                </dt>
                                <dd>
                                    {user.subscription.end_date
                                        ? new Date(
                                              user.subscription.end_date,
                                          ).toLocaleDateString()
                                        : '--'}
                                </dd>

                                <dt className="font-medium text-muted-foreground">
                                    Amount
                                </dt>
                                <dd>
                                    $
                                    {(
                                        user.subscription.amount_cents / 100
                                    ).toFixed(2)}
                                </dd>
                            </dl>
                        </CardContent>
                    </Card>
                )}

                {/* User Metas */}
                {user.metas && user.metas.length > 0 && (
                    <Card className="max-w-2xl">
                        <CardHeader>
                            <CardTitle>User Metadata</CardTitle>
                        </CardHeader>
                        <CardContent>
                            <dl className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm">
                                {user.metas.map((meta) => (
                                    <div
                                        key={meta.id}
                                        className="contents"
                                    >
                                        <dt className="font-medium text-muted-foreground">
                                            {meta.key}
                                        </dt>
                                        <dd className="break-all">
                                            {meta.value ?? '--'}
                                        </dd>
                                    </div>
                                ))}
                            </dl>
                        </CardContent>
                    </Card>
                )}
            </div>
        </AppLayout>
    );
}
