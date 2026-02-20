import { Head, Link, usePage } from '@inertiajs/react';
import { Button } from '@/components/ui/button';
import { dashboard, login, register } from '@/routes';
import type { PageData } from '@/types';

function NavHeader() {
    const { auth } = usePage().props;

    return (
        <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
            <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
                <Link href="/" className="text-lg font-semibold tracking-tight text-foreground">
                    Orchestra MCP
                </Link>
                <nav className="flex items-center gap-2">
                    {auth.user ? (
                        <Button asChild variant="outline" size="sm">
                            <Link href={dashboard()}>Dashboard</Link>
                        </Button>
                    ) : (
                        <>
                            <Button asChild variant="ghost" size="sm">
                                <Link href={login()}>Log in</Link>
                            </Button>
                            <Button asChild size="sm">
                                <Link href={register()}>Register</Link>
                            </Button>
                        </>
                    )}
                </nav>
            </div>
        </header>
    );
}

function isJsonContent(content: string): boolean {
    const trimmed = content.trim();
    return (trimmed.startsWith('{') && trimmed.endsWith('}')) ||
        (trimmed.startsWith('[') && trimmed.endsWith(']'));
}

function JsonContent({ content }: { content: string }) {
    let parsed: Record<string, unknown>;

    try {
        parsed = JSON.parse(content);
    } catch {
        return <HtmlContent content={content} />;
    }

    return (
        <div className="space-y-8">
            {Object.entries(parsed).map(([key, value]) => (
                <section key={key}>
                    <h2 className="mb-3 text-xl font-semibold capitalize text-foreground">
                        {key.replace(/_/g, ' ')}
                    </h2>
                    {typeof value === 'string' ? (
                        <p className="text-muted-foreground">{value}</p>
                    ) : (
                        <pre className="overflow-x-auto rounded-lg border bg-muted p-4 text-sm text-muted-foreground">
                            {JSON.stringify(value, null, 2)}
                        </pre>
                    )}
                </section>
            ))}
        </div>
    );
}

function HtmlContent({ content }: { content: string }) {
    return (
        <div
            className="prose prose-neutral max-w-none dark:prose-invert"
            dangerouslySetInnerHTML={{ __html: content }}
        />
    );
}

export default function Page({ page }: { page: PageData }) {
    return (
        <>
            <Head title={page.title} />
            <div className="min-h-screen bg-background text-foreground">
                <NavHeader />
                <main className="mx-auto max-w-4xl px-4 py-16 sm:px-6 lg:px-8">
                    <h1 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
                        {page.title}
                    </h1>
                    <div className="mt-8">
                        {isJsonContent(page.content) ? (
                            <JsonContent content={page.content} />
                        ) : (
                            <HtmlContent content={page.content} />
                        )}
                    </div>
                </main>
                <footer className="border-t py-8">
                    <div className="mx-auto max-w-7xl px-4 text-center text-sm text-muted-foreground sm:px-6 lg:px-8">
                        &copy; {new Date().getFullYear()} Orchestra MCP. All rights reserved.
                    </div>
                </footer>
            </div>
        </>
    );
}
