import { Head, Link, useForm } from '@inertiajs/react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import InputError from '@/components/input-error';
import AppLayout from '@/layouts/app-layout';
import type { BreadcrumbItem, PageData } from '@/types';
import type { FormEventHandler } from 'react';

const breadcrumbs: BreadcrumbItem[] = [
    { title: 'Admin', href: '/admin' },
    { title: 'Pages', href: '/admin/pages' },
    { title: 'Edit Page', href: '#' },
];

type Props = {
    page: PageData;
};

export default function PageEdit({ page }: Props) {
    const { data, setData, post, processing, errors } = useForm({
        _method: 'PUT' as const,
        title: page.title,
        content: page.content,
        meta: page.meta ? JSON.stringify(page.meta, null, 2) : '',
        is_published: page.is_published,
        image: null as File | null,
    });

    const submit: FormEventHandler = (e) => {
        e.preventDefault();
        post(`/admin/pages/${page.id}`, {
            forceFormData: true,
        });
    };

    return (
        <AppLayout breadcrumbs={breadcrumbs}>
            <Head title="Edit Page" />

            <div className="flex h-full flex-1 flex-col gap-4 rounded-xl p-4">
                <Card>
                    <CardHeader>
                        <CardTitle>Edit Page</CardTitle>
                        <CardDescription>Update the page content and settings</CardDescription>
                    </CardHeader>
                    <CardContent>
                        <form onSubmit={submit} className="space-y-6">
                            <div className="space-y-2">
                                <Label htmlFor="title">Title</Label>
                                <Input
                                    id="title"
                                    type="text"
                                    value={data.title}
                                    onChange={(e) => setData('title', e.target.value)}
                                />
                                <InputError message={errors.title} />
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="content">Content</Label>
                                <textarea
                                    id="content"
                                    className="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex min-h-[300px] w-full rounded-md border px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                                    value={data.content}
                                    onChange={(e) => setData('content', e.target.value)}
                                />
                                <InputError message={errors.content} />
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="meta">Meta (JSON)</Label>
                                <textarea
                                    id="meta"
                                    className="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex min-h-[120px] w-full rounded-md border px-3 py-2 font-mono text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                                    value={data.meta}
                                    onChange={(e) => setData('meta', e.target.value)}
                                    placeholder='{"description": "...", "keywords": "..."}'
                                />
                                <InputError message={errors.meta} />
                            </div>

                            <div className="flex items-center gap-2">
                                <Checkbox
                                    id="is_published"
                                    checked={data.is_published}
                                    onCheckedChange={(checked) => setData('is_published', checked === true)}
                                />
                                <Label htmlFor="is_published" className="cursor-pointer">
                                    Published
                                </Label>
                                <InputError message={errors.is_published} />
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="image">Image</Label>
                                <Input
                                    id="image"
                                    type="file"
                                    accept="image/*"
                                    onChange={(e) => setData('image', e.target.files?.[0] ?? null)}
                                />
                                <InputError message={errors.image} />
                            </div>

                            <div className="flex items-center gap-4">
                                <Button type="submit" disabled={processing}>
                                    Save Changes
                                </Button>
                                <Button asChild variant="outline">
                                    <Link href="/admin/pages">Cancel</Link>
                                </Button>
                            </div>
                        </form>
                    </CardContent>
                </Card>
            </div>
        </AppLayout>
    );
}
