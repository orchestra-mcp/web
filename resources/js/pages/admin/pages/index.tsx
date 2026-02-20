import { Head, Link } from '@inertiajs/react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import AppLayout from '@/layouts/app-layout';
import type { BreadcrumbItem, PageData } from '@/types';

const breadcrumbs: BreadcrumbItem[] = [
    { title: 'Admin', href: '/admin' },
    { title: 'Pages', href: '/admin/pages' },
];

type PaginationLink = {
    url: string | null;
    label: string;
    active: boolean;
};

type Props = {
    pages: {
        data: PageData[];
        links: PaginationLink[];
        current_page: number;
        last_page: number;
    };
};

function formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
    });
}

export default function PagesIndex({ pages }: Props) {
    return (
        <AppLayout breadcrumbs={breadcrumbs}>
            <Head title="Pages" />

            <div className="flex h-full flex-1 flex-col gap-4 rounded-xl p-4">
                <div className="flex items-center justify-between">
                    <h1 className="text-lg font-semibold">Pages</h1>
                </div>

                <div className="overflow-hidden rounded-lg border">
                    <table className="w-full text-sm">
                        <thead className="border-b bg-muted/50">
                            <tr>
                                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Title</th>
                                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Slug</th>
                                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Published</th>
                                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Last Updated</th>
                                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y">
                            {pages.data.map((page) => (
                                <tr key={page.id} className="hover:bg-muted/30">
                                    <td className="px-4 py-3 font-medium">{page.title}</td>
                                    <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{page.slug}</td>
                                    <td className="px-4 py-3">
                                        <Badge variant={page.is_published ? 'default' : 'secondary'}>
                                            {page.is_published ? 'Yes' : 'No'}
                                        </Badge>
                                    </td>
                                    <td className="px-4 py-3 text-muted-foreground">{formatDate(page.updated_at)}</td>
                                    <td className="px-4 py-3">
                                        <Button asChild variant="ghost" size="sm">
                                            <Link href={`/admin/pages/${page.id}/edit`}>Edit</Link>
                                        </Button>
                                    </td>
                                </tr>
                            ))}
                            {pages.data.length === 0 && (
                                <tr>
                                    <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                                        No pages found.
                                    </td>
                                </tr>
                            )}
                        </tbody>
                    </table>
                </div>

                {pages.last_page > 1 && (
                    <div className="flex items-center justify-center gap-1">
                        {pages.links.map((link, index) => (
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
