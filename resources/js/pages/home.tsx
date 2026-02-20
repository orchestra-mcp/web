import { Head, Link, usePage } from '@inertiajs/react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from '@/components/ui/card';
import { dashboard, login, register } from '@/routes';
import type { PageData } from '@/types';

type HeroSection = {
    title: string;
    subtitle: string;
    cta_text: string;
    cta_url: string;
};

type Feature = {
    title: string;
    description: string;
    icon: string;
};

type Plan = {
    name: string;
    price: string;
    features: string[];
};

type PageContent = {
    hero: HeroSection;
    features: Feature[];
    plans: Plan[];
};

function parsePageContent(page: PageData | null): PageContent | null {
    if (!page?.content) {
        return null;
    }

    try {
        return JSON.parse(page.content) as PageContent;
    } catch {
        return null;
    }
}

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

function HeroBlock({ hero }: { hero: HeroSection }) {
    return (
        <section className="relative overflow-hidden py-24 sm:py-32">
            <div className="mx-auto max-w-7xl px-4 text-center sm:px-6 lg:px-8">
                <Badge variant="secondary" className="mb-4">
                    AI-Powered IDE
                </Badge>
                <h1 className="mx-auto max-w-4xl text-4xl font-bold tracking-tight text-foreground sm:text-5xl lg:text-6xl">
                    {hero.title}
                </h1>
                <p className="mx-auto mt-6 max-w-2xl text-lg leading-8 text-muted-foreground">
                    {hero.subtitle}
                </p>
                {hero.cta_text && (
                    <div className="mt-10 flex items-center justify-center gap-4">
                        <Button asChild size="lg">
                            <a href={hero.cta_url}>{hero.cta_text}</a>
                        </Button>
                        <Button asChild variant="outline" size="lg">
                            <a href="#features">Learn more</a>
                        </Button>
                    </div>
                )}
            </div>
        </section>
    );
}

function FeaturesBlock({ features }: { features: Feature[] }) {
    return (
        <section id="features" className="border-t bg-muted/40 py-24 sm:py-32">
            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="mx-auto max-w-2xl text-center">
                    <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
                        Features
                    </h2>
                    <p className="mt-4 text-lg text-muted-foreground">
                        Everything you need to build, manage, and ship projects faster.
                    </p>
                </div>
                <div className="mx-auto mt-16 grid max-w-5xl gap-6 sm:grid-cols-2 lg:grid-cols-3">
                    {features.map((feature) => (
                        <Card key={feature.title}>
                            <CardHeader>
                                <div className="mb-2 flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-xl">
                                    {feature.icon}
                                </div>
                                <CardTitle className="text-base">{feature.title}</CardTitle>
                            </CardHeader>
                            <CardContent>
                                <CardDescription>{feature.description}</CardDescription>
                            </CardContent>
                        </Card>
                    ))}
                </div>
            </div>
        </section>
    );
}

function PlansBlock({ plans }: { plans: Plan[] }) {
    return (
        <section id="pricing" className="border-t py-24 sm:py-32">
            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="mx-auto max-w-2xl text-center">
                    <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
                        Pricing
                    </h2>
                    <p className="mt-4 text-lg text-muted-foreground">
                        Choose the plan that fits your workflow.
                    </p>
                </div>
                <div className="mx-auto mt-16 grid max-w-5xl gap-6 sm:grid-cols-2 lg:grid-cols-3">
                    {plans.map((plan) => (
                        <Card key={plan.name} className="flex flex-col">
                            <CardHeader>
                                <CardTitle>{plan.name}</CardTitle>
                                <div className="mt-2 text-3xl font-bold tracking-tight text-foreground">
                                    {plan.price}
                                </div>
                            </CardHeader>
                            <CardContent className="flex-1">
                                <ul className="space-y-3 text-sm text-muted-foreground">
                                    {plan.features.map((feature) => (
                                        <li key={feature} className="flex items-start gap-2">
                                            <svg
                                                className="mt-0.5 h-4 w-4 shrink-0 text-primary"
                                                fill="none"
                                                viewBox="0 0 24 24"
                                                strokeWidth={2}
                                                stroke="currentColor"
                                            >
                                                <path
                                                    strokeLinecap="round"
                                                    strokeLinejoin="round"
                                                    d="M4.5 12.75l6 6 9-13.5"
                                                />
                                            </svg>
                                            {feature}
                                        </li>
                                    ))}
                                </ul>
                            </CardContent>
                        </Card>
                    ))}
                </div>
            </div>
        </section>
    );
}

function Footer() {
    return (
        <footer className="border-t py-8">
            <div className="mx-auto max-w-7xl px-4 text-center text-sm text-muted-foreground sm:px-6 lg:px-8">
                &copy; {new Date().getFullYear()} Orchestra MCP. All rights reserved.
            </div>
        </footer>
    );
}

function Fallback() {
    return (
        <div className="flex min-h-[60vh] items-center justify-center">
            <div className="text-center">
                <h1 className="text-4xl font-bold tracking-tight text-foreground">
                    Orchestra MCP
                </h1>
                <p className="mt-4 text-lg text-muted-foreground">
                    The AI-agentic IDE for every platform.
                </p>
                <div className="mt-8 flex items-center justify-center gap-4">
                    <Button asChild>
                        <Link href={register()}>Get Started</Link>
                    </Button>
                    <Button asChild variant="outline">
                        <Link href={login()}>Log in</Link>
                    </Button>
                </div>
            </div>
        </div>
    );
}

export default function Home({ page }: { page: PageData | null }) {
    const content = parsePageContent(page);

    return (
        <>
            <Head title="Orchestra MCP" />
            <div className="min-h-screen bg-background text-foreground">
                <NavHeader />
                {content ? (
                    <main>
                        <HeroBlock hero={content.hero} />
                        {content.features.length > 0 && (
                            <FeaturesBlock features={content.features} />
                        )}
                        {content.plans.length > 0 && (
                            <PlansBlock plans={content.plans} />
                        )}
                    </main>
                ) : (
                    <main>
                        <Fallback />
                    </main>
                )}
                <Footer />
            </div>
        </>
    );
}
