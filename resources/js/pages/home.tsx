import { Head, Link, usePage } from '@inertiajs/react';
import { useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { dashboard, login, register } from '@/routes';
import AppLogoIcon from '@/components/app-logo-icon';
import type { PageData } from '@/types';

/* ── Intersection Observer hook for scroll animations ── */
function useScrollAnimation() {
    useEffect(() => {
        let observer: IntersectionObserver | null = null;

        // Wait one frame so the DOM is fully painted after React render
        const rafId = requestAnimationFrame(() => {
            const targets = document.querySelectorAll(
                '.animate-on-scroll, .animate-on-scroll-scale, .stagger-children',
            );
            if (!targets.length) return;

            observer = new IntersectionObserver(
                (entries) => {
                    entries.forEach((entry) => {
                        if (entry.isIntersecting) {
                            entry.target.classList.add('is-visible');
                            observer?.unobserve(entry.target);
                        }
                    });
                },
                { threshold: 0.1, rootMargin: '0px 0px -40px 0px' },
            );

            targets.forEach((el) => observer!.observe(el));
        });

        return () => {
            cancelAnimationFrame(rafId);
            observer?.disconnect();
        };
    }, []);
}

/* ── Smooth scroll handler for anchor links ── */
function handleAnchorClick(e: React.MouseEvent<HTMLAnchorElement>) {
    const href = e.currentTarget.getAttribute('href');
    if (href?.startsWith('#')) {
        e.preventDefault();
        document.querySelector(href)?.scrollIntoView({ behavior: 'smooth' });
    }
}

/* ── Data ── */

const platforms = [
    { icon: 'bx bxl-apple', label: 'macOS', available: true },
    { icon: 'bx bxl-windows', label: 'Windows', available: true },
    { icon: 'bx bxs-terminal', label: 'Linux', available: true },
    { icon: 'bx bxl-chrome', label: 'Chrome', available: false },
    { icon: 'bx bxl-apple', label: 'iOS', available: false },
    { icon: 'bx bxl-android', label: 'Android', available: false },
    { icon: 'bx bx-globe', label: 'Web', available: false },
];

const features = [
    {
        icon: 'bx bx-brain',
        title: 'AI-Powered Code Generation',
        description:
            'Built-in AI chat with Claude and GPT. Generate code, refactor, debug, and get intelligent suggestions directly in your editor.',
    },
    {
        icon: 'bx bx-bot',
        title: 'Agentic Workflows',
        description:
            '16 specialized AI agents — Go architect, Rust engineer, frontend dev, DBA, DevOps, QA, and more — that autonomously handle complex tasks.',
    },
    {
        icon: 'bx bx-command',
        title: '57 MCP Tools',
        description:
            'Full project management via Model Context Protocol: epics, stories, tasks, bugs, PRD sessions, memory search, and workflow automation.',
    },
    {
        icon: 'bx bx-devices',
        title: 'Desktop IDE',
        description:
            'Native desktop app for macOS, Windows, and Linux built with Wails. Chrome extension, mobile apps, and web dashboard coming soon.',
    },
    {
        icon: 'bx bx-extension',
        title: 'Extension Ecosystem',
        description:
            'Native extension API plus Raycast compatibility (~95%) and VS Code compatibility (~85%). Build or migrate extensions effortlessly.',
    },
    {
        icon: 'bx bx-sync',
        title: 'Real-Time Sync',
        description:
            'UUID-based sync with version vectors, Redis pub/sub, and last-write-wins conflict resolution. Your work follows you across every device.',
    },
    {
        icon: 'bx bx-code-block',
        title: 'Rust-Powered Engine',
        description:
            'Tree-sitter parsing, Tantivy full-text search, LSP integration, file diffing, and AES-256-GCM encryption — all in a blazing-fast Rust engine.',
    },
    {
        icon: 'bx bx-palette',
        title: '26 Color Themes',
        description:
            'From minimal to Matrix-inspired. Switch themes on the fly with 3 component layout variants (compact, modern, default).',
    },
    {
        icon: 'bx bx-memory-card',
        title: 'RAG Memory System',
        description:
            'Automatic session tracking with hybrid keyword + vector search. Your AI remembers context across conversations.',
    },
    {
        icon: 'bx bx-git-branch',
        title: '13-State Workflow',
        description:
            'From backlog to done with gated transitions: testing, documentation, and code review. Never skip a quality step.',
    },
    {
        icon: 'bx bx-store',
        title: 'Extension Marketplace',
        description:
            'Publish, discover, and install extensions. Full CLI tooling, versioning, reviews, ratings, and auto-updates.',
    },
    {
        icon: 'bx bx-shield-quarter',
        title: 'Enterprise Security',
        description:
            'End-to-end encryption, Sanctum API tokens, role-based permissions, macOS Keychain integration, and sandboxed extensions.',
    },
];

const techStack = [
    { icon: 'bx bxl-go-lang', label: 'Go', desc: 'Fiber v3 + GORM backend' },
    { icon: 'bx bxs-bolt', label: 'Rust', desc: 'Tonic gRPC engine' },
    { icon: 'bx bxl-react', label: 'React', desc: 'TypeScript + Zustand' },
    {
        icon: 'bx bxl-postgresql',
        label: 'PostgreSQL',
        desc: 'pgvector + full-text search',
    },
    { icon: 'bx bxs-data', label: 'SQLite', desc: 'Offline-first local DB' },
    { icon: 'bx bxs-bolt-circle', label: 'Redis', desc: 'Real-time pub/sub' },
    { icon: 'bx bxl-docker', label: 'Docker', desc: 'Containerized deploys' },
    { icon: 'bx bxl-google-cloud', label: 'GCP', desc: 'Cloud Run + CDN' },
];

const plans = [
    {
        name: 'Free',
        price: '$0',
        period: 'forever',
        description: 'For individual developers getting started.',
        features: [
            'Core IDE features',
            'Community extensions',
            'Local AI assistance',
            '1 synced device',
            'SQLite local storage',
        ],
        cta: 'Get Started',
        highlighted: false,
    },
    {
        name: 'Pro',
        price: '$5',
        period: '/month',
        description: 'For power users who want the full experience.',
        features: [
            'Everything in Free',
            'Cloud AI (Claude + GPT)',
            'Unlimited synced devices',
            'RAG memory system',
            'Priority support',
            'All 26 themes',
        ],
        cta: 'Start Pro',
        highlighted: true,
    },
    {
        name: 'Team',
        price: '$25',
        period: '/month',
        description: 'For teams building together.',
        features: [
            'Everything in Pro',
            'Team collaboration',
            'Admin panel & analytics',
            'Custom extensions',
            'Shared workspaces',
            'SLA & dedicated support',
        ],
        cta: 'Start Team',
        highlighted: false,
    },
];

const stats = [
    { value: '57', label: 'MCP Tools' },
    { value: '16', label: 'AI Agents' },
    { value: '26', label: 'Themes' },
    { value: '3', label: 'Desktop OSes' },
];

/* ── Sections ── */

function NavHeader() {
    const { auth } = usePage().props;

    return (
        <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
            <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
                <Link
                    href="/"
                    className="flex items-center gap-2 text-lg font-bold tracking-tight text-foreground"
                >
                    <AppLogoIcon className="size-7" />
                    Orchestra MCP
                </Link>
                <nav className="hidden items-center gap-6 text-sm font-medium text-muted-foreground md:flex">
                    <a
                        href="#features"
                        onClick={handleAnchorClick}
                        className="transition-colors hover:text-foreground"
                    >
                        Features
                    </a>
                    <a
                        href="#tech"
                        onClick={handleAnchorClick}
                        className="transition-colors hover:text-foreground"
                    >
                        Tech Stack
                    </a>
                    <a
                        href="#pricing"
                        onClick={handleAnchorClick}
                        className="transition-colors hover:text-foreground"
                    >
                        Pricing
                    </a>
                </nav>
                <div className="flex items-center gap-2">
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
                                <Link href={register()}>Get Started</Link>
                            </Button>
                        </>
                    )}
                </div>
            </div>
        </header>
    );
}

function HeroSection() {
    return (
        <section className="relative overflow-hidden py-24 sm:py-32 lg:py-40">
            <div className="absolute inset-0 -z-10 bg-[radial-gradient(ellipse_at_top,oklch(0.55_0.27_295/0.08),oklch(0.82_0.19_195/0.04),transparent)]" />
            <div className="mx-auto max-w-7xl px-4 text-center sm:px-6 lg:px-8">
                <div className="animate-fade-in-up">
                    <Badge
                        variant="secondary"
                        className="mb-6 px-4 py-1.5 text-sm"
                    >
                        <i className="bx bx-rocket mr-1.5" />
                        AI-Agentic IDE for Every Platform
                    </Badge>
                </div>
                <h1 className="animate-fade-in-up-d1 mx-auto max-w-5xl text-5xl font-extrabold tracking-tight text-foreground sm:text-6xl lg:text-7xl">
                    Build Smarter with{' '}
                    <span className="brand-gradient-text">
                        Orchestra MCP
                    </span>
                </h1>
                <p className="animate-fade-in-up-d2 mx-auto mt-6 max-w-3xl text-lg leading-8 text-muted-foreground sm:text-xl">
                    The AI-powered desktop IDE that brings 16 specialized
                    agents, 57 project management tools, and a blazing-fast
                    Rust engine to macOS, Windows, and Linux.
                </p>
                <div className="animate-fade-in-up-d3 mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row">
                    <Button asChild size="lg" className="gap-2 px-8 text-base">
                        <Link href={register()}>
                            <i className="bx bx-download" />
                            Get Started Free
                        </Link>
                    </Button>
                    <Button
                        asChild
                        variant="outline"
                        size="lg"
                        className="gap-2 px-8 text-base"
                    >
                        <a href="#features" onClick={handleAnchorClick}>
                            <i className="bx bx-play-circle" />
                            See Features
                        </a>
                    </Button>
                </div>

                {/* Platform icons */}
                <div className="animate-fade-in-up-d4 mt-16">
                    <p className="mb-4 text-xs font-medium tracking-widest text-muted-foreground uppercase">
                        Available on Desktop
                    </p>
                    <div className="flex flex-wrap items-center justify-center gap-6">
                        {platforms.map((p) => (
                            <div
                                key={p.label}
                                className={`flex flex-col items-center gap-1.5 transition-colors ${p.available ? 'text-foreground' : 'text-muted-foreground/40'}`}
                            >
                                <i className={`${p.icon} text-2xl`} />
                                <span className="text-xs">
                                    {p.label}
                                    {!p.available && (
                                        <span className="ml-1 text-[10px] opacity-60">
                                            soon
                                        </span>
                                    )}
                                </span>
                            </div>
                        ))}
                    </div>
                </div>
            </div>
        </section>
    );
}

function StatsSection() {
    return (
        <section className="border-y bg-muted/30 py-12">
            <div className="stagger-children mx-auto grid max-w-5xl grid-cols-2 gap-8 px-4 sm:grid-cols-4 sm:px-6 lg:px-8">
                {stats.map((s) => (
                    <div key={s.label} className="text-center">
                        <div className="brand-gradient-text text-4xl font-extrabold tracking-tight">
                            {s.value}
                        </div>
                        <div className="mt-1 text-sm font-medium text-muted-foreground">
                            {s.label}
                        </div>
                    </div>
                ))}
            </div>
        </section>
    );
}

function FeaturesSection() {
    return (
        <section id="features" className="scroll-mt-20 py-24 sm:py-32">
            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="animate-on-scroll mx-auto max-w-3xl text-center">
                    <Badge variant="outline" className="mb-4">
                        <i className="bx bx-star mr-1" />
                        Features
                    </Badge>
                    <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
                        Everything you need to ship faster
                    </h2>
                    <p className="mt-4 text-lg text-muted-foreground">
                        From AI code generation to project management,
                        Orchestra MCP combines the best tools into one
                        unified experience.
                    </p>
                </div>
                <div className="stagger-children mx-auto mt-16 grid max-w-6xl gap-5 sm:grid-cols-2 lg:grid-cols-3">
                    {features.map((f) => (
                        <Card
                            key={f.title}
                            className="group transition-all duration-300 hover:-translate-y-1 hover:shadow-lg"
                        >
                            <CardHeader className="pb-3">
                                <div className="mb-3 flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10 text-primary transition-colors group-hover:bg-primary group-hover:text-primary-foreground">
                                    <i className={`${f.icon} text-2xl`} />
                                </div>
                                <CardTitle className="text-base leading-snug">
                                    {f.title}
                                </CardTitle>
                            </CardHeader>
                            <CardContent>
                                <CardDescription className="leading-relaxed">
                                    {f.description}
                                </CardDescription>
                            </CardContent>
                        </Card>
                    ))}
                </div>
            </div>
        </section>
    );
}

function AgentsSection() {
    const agents = [
        {
            icon: 'bx bxl-go-lang',
            name: 'Go Architect',
            desc: 'Fiber v3, GORM, services, routes',
        },
        {
            icon: 'bx bxs-bolt',
            name: 'Rust Engineer',
            desc: 'Tonic, Tree-sitter, Tantivy',
        },
        {
            icon: 'bx bxl-react',
            name: 'Frontend Dev',
            desc: 'React, TypeScript, 5 platforms',
        },
        {
            icon: 'bx bx-palette',
            name: 'UI/UX Designer',
            desc: 'shadcn/ui, Tailwind, a11y',
        },
        {
            icon: 'bx bx-data',
            name: 'DBA',
            desc: 'PostgreSQL, SQLite, Redis, sync',
        },
        {
            icon: 'bx bx-mobile',
            name: 'Mobile Dev',
            desc: 'React Native, WatermelonDB',
        },
        {
            icon: 'bx bx-brain',
            name: 'AI Engineer',
            desc: 'LLM, RAG, embeddings, agents',
        },
        {
            icon: 'bx bx-server',
            name: 'DevOps',
            desc: 'Docker, GCP, CI/CD, monitoring',
        },
    ];

    return (
        <section className="border-t bg-muted/40 py-24 sm:py-32">
            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="animate-on-scroll mx-auto max-w-3xl text-center">
                    <Badge variant="outline" className="mb-4">
                        <i className="bx bx-bot mr-1" />
                        AI Agents
                    </Badge>
                    <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
                        16 specialized agents at your command
                    </h2>
                    <p className="mt-4 text-lg text-muted-foreground">
                        Each agent is an expert in its domain — from backend
                        architecture to mobile development. They work
                        autonomously, then report back.
                    </p>
                </div>
                <div className="stagger-children mx-auto mt-16 grid max-w-4xl gap-4 sm:grid-cols-2 lg:grid-cols-4">
                    {agents.map((a) => (
                        <div
                            key={a.name}
                            className="flex flex-col items-center rounded-xl border bg-card p-5 text-center transition-all duration-300 hover:-translate-y-1 hover:shadow-md"
                        >
                            <div className="mb-3 flex h-11 w-11 items-center justify-center rounded-lg bg-primary/10 text-primary">
                                <i className={`${a.icon} text-xl`} />
                            </div>
                            <div className="text-sm font-semibold text-foreground">
                                {a.name}
                            </div>
                            <div className="mt-1 text-xs text-muted-foreground">
                                {a.desc}
                            </div>
                        </div>
                    ))}
                </div>
                <p className="animate-on-scroll mt-8 text-center text-sm text-muted-foreground">
                    Plus 8 more: Scrum Master, Widget Engineer, Platform
                    Engineer, Extension Architect, QA Go, QA Rust, QA Node, QA
                    Playwright
                </p>
            </div>
        </section>
    );
}

function TechSection() {
    return (
        <section id="tech" className="scroll-mt-20 border-t py-24 sm:py-32">
            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="animate-on-scroll mx-auto max-w-3xl text-center">
                    <Badge variant="outline" className="mb-4">
                        <i className="bx bx-chip mr-1" />
                        Tech Stack
                    </Badge>
                    <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
                        Built on battle-tested technology
                    </h2>
                    <p className="mt-4 text-lg text-muted-foreground">
                        Go for the API layer, Rust for the engine, React for
                        every frontend. No compromises.
                    </p>
                </div>
                <div className="stagger-children mx-auto mt-16 grid max-w-4xl gap-4 sm:grid-cols-2 lg:grid-cols-4">
                    {techStack.map((t) => (
                        <div
                            key={t.label}
                            className="flex items-center gap-3 rounded-xl border bg-card p-4 transition-all duration-300 hover:-translate-y-1 hover:shadow-md"
                        >
                            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                                <i className={`${t.icon} text-xl`} />
                            </div>
                            <div>
                                <div className="text-sm font-semibold text-foreground">
                                    {t.label}
                                </div>
                                <div className="text-xs text-muted-foreground">
                                    {t.desc}
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        </section>
    );
}

function WorkflowSection() {
    const steps = [
        {
            icon: 'bx bx-list-plus',
            label: 'Backlog',
            color: 'bg-muted text-muted-foreground',
        },
        {
            icon: 'bx bx-check-circle',
            label: 'Todo',
            color: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
        },
        {
            icon: 'bx bx-loader-alt',
            label: 'In Progress',
            color: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400',
        },
        {
            icon: 'bx bx-test-tube',
            label: 'Testing',
            color: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
        },
        {
            icon: 'bx bx-file',
            label: 'Docs',
            color: 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400',
        },
        {
            icon: 'bx bx-search-alt',
            label: 'Review',
            color: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
        },
        {
            icon: 'bx bx-check-double',
            label: 'Done',
            color: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
        },
    ];

    return (
        <section className="border-t bg-muted/40 py-24 sm:py-32">
            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="animate-on-scroll mx-auto max-w-3xl text-center">
                    <Badge variant="outline" className="mb-4">
                        <i className="bx bx-git-merge mr-1" />
                        Workflow
                    </Badge>
                    <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
                        13-state lifecycle with gated transitions
                    </h2>
                    <p className="mt-4 text-lg text-muted-foreground">
                        Every task flows through testing, documentation, and
                        review gates. Quality is built into the process, not
                        bolted on.
                    </p>
                </div>
                <div className="stagger-children mx-auto mt-12 flex max-w-4xl flex-wrap items-center justify-center gap-3">
                    {steps.map((s, i) => (
                        <div key={s.label} className="flex items-center gap-3">
                            <div
                                className={`flex items-center gap-2 rounded-full px-4 py-2 text-sm font-medium ${s.color}`}
                            >
                                <i className={s.icon} />
                                {s.label}
                            </div>
                            {i < steps.length - 1 && (
                                <i className="bx bx-right-arrow-alt text-lg text-muted-foreground" />
                            )}
                        </div>
                    ))}
                </div>
            </div>
        </section>
    );
}

function PricingSection() {
    return (
        <section id="pricing" className="scroll-mt-20 border-t py-24 sm:py-32">
            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="animate-on-scroll mx-auto max-w-3xl text-center">
                    <Badge variant="outline" className="mb-4">
                        <i className="bx bx-purchase-tag mr-1" />
                        Pricing
                    </Badge>
                    <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
                        Simple, transparent pricing
                    </h2>
                    <p className="mt-4 text-lg text-muted-foreground">
                        Start free. Upgrade when you need cloud AI and unlimited
                        sync.
                    </p>
                </div>
                <div className="stagger-children mx-auto mt-16 grid max-w-5xl gap-6 sm:grid-cols-2 lg:grid-cols-3">
                    {plans.map((plan) => (
                        <Card
                            key={plan.name}
                            className={`relative flex flex-col transition-all duration-300 hover:-translate-y-1 ${plan.highlighted ? 'border-primary shadow-lg ring-1 ring-primary/20' : ''}`}
                        >
                            {plan.highlighted && (
                                <div className="absolute -top-3 left-1/2 -translate-x-1/2">
                                    <Badge className="px-3">
                                        Most Popular
                                    </Badge>
                                </div>
                            )}
                            <CardHeader>
                                <CardTitle className="text-lg">
                                    {plan.name}
                                </CardTitle>
                                <div className="mt-3 flex items-baseline gap-1">
                                    <span className="text-4xl font-extrabold tracking-tight text-foreground">
                                        {plan.price}
                                    </span>
                                    <span className="text-sm text-muted-foreground">
                                        {plan.period}
                                    </span>
                                </div>
                                <p className="mt-2 text-sm text-muted-foreground">
                                    {plan.description}
                                </p>
                            </CardHeader>
                            <CardContent className="flex flex-1 flex-col">
                                <Separator className="mb-5" />
                                <ul className="flex-1 space-y-3 text-sm">
                                    {plan.features.map((feature) => (
                                        <li
                                            key={feature}
                                            className="flex items-start gap-2.5 text-muted-foreground"
                                        >
                                            <i className="bx bx-check mt-0.5 text-lg text-primary" />
                                            {feature}
                                        </li>
                                    ))}
                                </ul>
                                <Button
                                    asChild
                                    className="mt-8 w-full"
                                    variant={
                                        plan.highlighted
                                            ? 'default'
                                            : 'outline'
                                    }
                                >
                                    <Link href={register()}>{plan.cta}</Link>
                                </Button>
                            </CardContent>
                        </Card>
                    ))}
                </div>
            </div>
        </section>
    );
}

function CTASection() {
    return (
        <section className="brand-gradient border-t py-20">
            <div className="animate-on-scroll-scale mx-auto max-w-4xl px-4 text-center sm:px-6 lg:px-8">
                <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">
                    Ready to orchestrate your workflow?
                </h2>
                <p className="mx-auto mt-4 max-w-2xl text-lg text-white/80">
                    Join developers who are shipping faster with AI agents,
                    real-time sync, and a 13-state workflow engine.
                </p>
                <div className="mt-8 flex flex-col items-center justify-center gap-4 sm:flex-row">
                    <Button
                        asChild
                        size="lg"
                        variant="secondary"
                        className="gap-2 px-8 text-base"
                    >
                        <Link href={register()}>
                            <i className="bx bx-rocket" />
                            Get Started Free
                        </Link>
                    </Button>
                    <Button
                        asChild
                        size="lg"
                        variant="outline"
                        className="gap-2 border-white/30 px-8 text-base text-white hover:bg-white/10 hover:text-white"
                    >
                        <a href="https://github.com/orchestra-mcp">
                            <i className="bx bxl-github" />
                            View on GitHub
                        </a>
                    </Button>
                </div>
            </div>
        </section>
    );
}

function Footer() {
    return (
        <footer className="border-t py-12">
            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-4">
                    <div>
                        <div className="flex items-center gap-2 text-lg font-bold text-foreground">
                            <AppLogoIcon className="size-6" />
                            Orchestra MCP
                        </div>
                        <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
                            The AI-Agentic IDE for modern development. Build,
                            manage, and ship across every platform.
                        </p>
                    </div>
                    <div>
                        <h3 className="text-sm font-semibold text-foreground">
                            Product
                        </h3>
                        <ul className="mt-3 space-y-2 text-sm text-muted-foreground">
                            <li>
                                <a
                                    href="#features"
                                    onClick={handleAnchorClick}
                                    className="hover:text-foreground"
                                >
                                    Features
                                </a>
                            </li>
                            <li>
                                <a
                                    href="#pricing"
                                    onClick={handleAnchorClick}
                                    className="hover:text-foreground"
                                >
                                    Pricing
                                </a>
                            </li>
                            <li>
                                <a
                                    href="#tech"
                                    onClick={handleAnchorClick}
                                    className="hover:text-foreground"
                                >
                                    Tech Stack
                                </a>
                            </li>
                        </ul>
                    </div>
                    <div>
                        <h3 className="text-sm font-semibold text-foreground">
                            Developers
                        </h3>
                        <ul className="mt-3 space-y-2 text-sm text-muted-foreground">
                            <li>
                                <a
                                    href="https://github.com/orchestra-mcp"
                                    className="hover:text-foreground"
                                >
                                    GitHub
                                </a>
                            </li>
                            <li>
                                <a href="#" className="hover:text-foreground">
                                    Documentation
                                </a>
                            </li>
                            <li>
                                <a href="#" className="hover:text-foreground">
                                    Extensions
                                </a>
                            </li>
                        </ul>
                    </div>
                    <div>
                        <h3 className="text-sm font-semibold text-foreground">
                            Company
                        </h3>
                        <ul className="mt-3 space-y-2 text-sm text-muted-foreground">
                            <li>
                                <a href="#" className="hover:text-foreground">
                                    About
                                </a>
                            </li>
                            <li>
                                <a href="#" className="hover:text-foreground">
                                    Privacy
                                </a>
                            </li>
                            <li>
                                <a href="#" className="hover:text-foreground">
                                    Terms
                                </a>
                            </li>
                        </ul>
                    </div>
                </div>
                <Separator className="my-8" />
                <div className="flex flex-col items-center justify-between gap-4 text-sm text-muted-foreground sm:flex-row">
                    <p>
                        &copy; {new Date().getFullYear()} Orchestra MCP. All
                        rights reserved.
                    </p>
                    <div className="flex items-center gap-4">
                        <a
                            href="https://github.com/orchestra-mcp"
                            className="transition-colors hover:text-foreground"
                        >
                            <i className="bx bxl-github text-xl" />
                        </a>
                        <a
                            href="https://discord.gg/orchestra-mcp"
                            className="transition-colors hover:text-foreground"
                        >
                            <i className="bx bxl-discord-alt text-xl" />
                        </a>
                        <a
                            href="https://twitter.com/orchestra_mcp"
                            className="transition-colors hover:text-foreground"
                        >
                            <i className="bx bxl-twitter text-xl" />
                        </a>
                    </div>
                </div>
            </div>
        </footer>
    );
}

export default function Home({ page: _page }: { page: PageData | null }) {
    useScrollAnimation();

    return (
        <>
            <Head title="Orchestra MCP — AI-Agentic IDE for Every Platform" />
            <div className="min-h-screen bg-background text-foreground">
                <NavHeader />
                <main>
                    <HeroSection />
                    <StatsSection />
                    <FeaturesSection />
                    <AgentsSection />
                    <WorkflowSection />
                    <TechSection />
                    <PricingSection />
                    <CTASection />
                </main>
                <Footer />
            </div>
        </>
    );
}
