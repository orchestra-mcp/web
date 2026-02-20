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

const GITHUB_SPONSORS_URL = 'https://github.com/sponsors/fadymondy';

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
        ctaHref: 'register' as const,
        highlighted: false,
    },
    {
        name: 'Sponsor',
        price: '$5',
        period: '/month',
        description: 'Support the project and unlock the full experience.',
        features: [
            'Everything in Free',
            'Cloud AI (Claude + GPT)',
            'Unlimited synced devices',
            'RAG memory system',
            'Priority support',
            'All 26 themes',
        ],
        cta: 'Sponsor on GitHub',
        ctaHref: 'github' as const,
        highlighted: true,
    },
    {
        name: 'Team Sponsor',
        price: '$25',
        period: '/month',
        description: 'For teams building together. Sponsor at a higher tier.',
        features: [
            'Everything in Sponsor',
            'Team collaboration',
            'Admin panel & analytics',
            'Custom extensions',
            'Shared workspaces',
            'SLA & dedicated support',
        ],
        cta: 'Sponsor on GitHub',
        ctaHref: 'github' as const,
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
        <header className="glass sticky top-0 z-50 w-full border-b border-border/50">
            <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
                <Link
                    href="/"
                    className="flex items-center gap-2.5 text-lg font-bold tracking-tight text-foreground"
                >
                    <AppLogoIcon className="size-7" />
                    <span>Orchestra MCP</span>
                </Link>
                <nav className="hidden items-center gap-8 text-sm font-medium text-muted-foreground md:flex">
                    <a
                        href="#features"
                        onClick={handleAnchorClick}
                        className="nav-link-gradient transition-colors hover:text-foreground"
                    >
                        Features
                    </a>
                    <a
                        href="#agents"
                        onClick={handleAnchorClick}
                        className="nav-link-gradient transition-colors hover:text-foreground"
                    >
                        Agents
                    </a>
                    <a
                        href="#tech"
                        onClick={handleAnchorClick}
                        className="nav-link-gradient transition-colors hover:text-foreground"
                    >
                        Tech Stack
                    </a>
                    <a
                        href="#pricing"
                        onClick={handleAnchorClick}
                        className="nav-link-gradient transition-colors hover:text-foreground"
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
                            <Button
                                asChild
                                size="sm"
                                className="brand-gradient border-0 text-white shadow-md hover:opacity-90"
                            >
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
        <section className="relative overflow-hidden py-28 sm:py-36 lg:py-44">
            {/* Animated mesh gradient background */}
            <div className="hero-mesh-bg absolute inset-0 -z-20" />

            {/* Dot grid pattern overlay */}
            <div className="hero-grid-pattern absolute inset-0 -z-10" />

            {/* Floating orbs */}
            <div className="hero-orb hero-orb-cyan absolute -left-32 top-1/4 h-96 w-96" />
            <div className="hero-orb hero-orb-purple absolute -right-32 bottom-1/4 h-80 w-80" />

            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="text-center">
                    <div className="animate-fade-in-up">
                        <Badge
                            variant="secondary"
                            className="mb-8 border border-border/50 px-4 py-1.5 text-sm shadow-sm"
                        >
                            <i className="bx bx-rocket mr-1.5 text-primary" />
                            Now in Public Beta
                        </Badge>
                    </div>
                    <h1 className="animate-fade-in-up-d1 mx-auto max-w-5xl text-5xl font-extrabold tracking-tight text-foreground sm:text-6xl lg:text-7xl xl:text-8xl">
                        The IDE that{' '}
                        <span className="brand-gradient-text">
                            thinks with you
                        </span>
                    </h1>
                    <p className="animate-fade-in-up-d2 mx-auto mt-8 max-w-2xl text-lg leading-relaxed text-muted-foreground sm:text-xl">
                        16 AI agents. 57 project tools. A Rust-powered engine.
                        One unified desktop IDE for macOS, Windows, and Linux.
                    </p>
                    <div className="animate-fade-in-up-d3 mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row">
                        <Button
                            asChild
                            size="lg"
                            className="brand-gradient gap-2 border-0 px-8 text-base text-white shadow-lg hover:opacity-90"
                        >
                            <Link href={register()}>
                                <i className="bx bx-download" />
                                Download for Free
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

                    {/* Platform availability */}
                    <div className="animate-fade-in-up-d4 mt-14">
                        <p className="mb-4 text-xs font-medium tracking-widest text-muted-foreground/70 uppercase">
                            Available on
                        </p>
                        <div className="flex flex-wrap items-center justify-center gap-6">
                            {platforms.map((p) => (
                                <div
                                    key={p.label}
                                    className={`flex flex-col items-center gap-1.5 transition-all duration-300 ${
                                        p.available
                                            ? 'text-foreground'
                                            : 'text-muted-foreground/30'
                                    }`}
                                >
                                    <i className={`${p.icon} text-2xl`} />
                                    <span className="text-xs font-medium">
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

                {/* Terminal mockup */}
                <div className="animate-fade-in-up-d5 mx-auto mt-20 max-w-2xl">
                    <div className="terminal-window brand-glow">
                        <div className="terminal-header">
                            <div className="terminal-dot bg-[#ff5f57]" />
                            <div className="terminal-dot bg-[#febc2e]" />
                            <div className="terminal-dot bg-[#28c840]" />
                            <span className="ml-3 text-xs text-[oklch(0.55_0_0)]">
                                orchestra-mcp
                            </span>
                        </div>
                        <div className="terminal-body">
                            <div>
                                <span className="t-dim">$</span>{' '}
                                <span className="t-cyan">npx</span>{' '}
                                <span className="t-white">orchestra-mcp init</span>
                            </div>
                            <div className="mt-1">
                                <span className="t-green">
                                    {'>'} Workspace initialized
                                </span>
                            </div>
                            <div className="mt-3">
                                <span className="t-dim">$</span>{' '}
                                <span className="t-cyan">orchestra</span>{' '}
                                <span className="t-white">start</span>
                            </div>
                            <div className="mt-1">
                                <span className="t-purple">
                                    {'>'} 57 tools loaded
                                </span>
                            </div>
                            <div>
                                <span className="t-purple">
                                    {'>'} 16 agents ready
                                </span>
                            </div>
                            <div>
                                <span className="t-green">
                                    {'>'} Rust engine connected
                                </span>
                            </div>
                            <div className="mt-3">
                                <span className="t-dim">$</span>{' '}
                                <span className="typing-cursor" />
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </section>
    );
}

function StatsSection() {
    return (
        <section className="relative border-y border-border/50 py-16">
            <div className="absolute inset-0 -z-10 bg-muted/20" />
            <div className="stagger-children mx-auto grid max-w-5xl grid-cols-2 gap-6 px-4 sm:grid-cols-4 sm:px-6 lg:px-8">
                {stats.map((s) => (
                    <div
                        key={s.label}
                        className="glass-card group rounded-2xl border border-border/50 p-6 text-center transition-all duration-300 hover:-translate-y-1 hover:shadow-lg"
                    >
                        <div className="brand-gradient-text text-5xl font-extrabold tracking-tighter">
                            {s.value}
                        </div>
                        <div className="mt-2 text-sm font-medium text-muted-foreground">
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
        <section id="features" className="scroll-mt-20 py-28 sm:py-36">
            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="animate-on-scroll mx-auto max-w-3xl text-center">
                    <Badge
                        variant="outline"
                        className="mb-4 border-primary/20"
                    >
                        <i className="bx bx-star mr-1 text-primary" />
                        Features
                    </Badge>
                    <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl lg:text-5xl">
                        Everything you need to{' '}
                        <span className="brand-gradient-text">ship faster</span>
                    </h2>
                    <p className="mt-5 text-lg leading-relaxed text-muted-foreground">
                        From AI code generation to project management,
                        Orchestra MCP combines the best tools into one
                        unified experience.
                    </p>
                </div>
                <div className="stagger-children mx-auto mt-20 grid max-w-6xl gap-5 sm:grid-cols-2 lg:grid-cols-3">
                    {features.map((f) => (
                        <Card
                            key={f.title}
                            className="group glass-card relative overflow-hidden border-border/50 transition-all duration-300 hover:-translate-y-1.5 hover:border-transparent hover:shadow-xl"
                        >
                            {/* Gradient border on hover */}
                            <div className="absolute inset-0 -z-10 rounded-[inherit] bg-gradient-to-br from-[#00e5ff]/0 via-transparent to-[#a900ff]/0 opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
                            <div className="absolute inset-px -z-10 rounded-[calc(inherit-1px)] bg-card transition-colors group-hover:bg-card/95" />
                            <div className="absolute inset-0 rounded-[inherit] opacity-0 transition-opacity duration-300 group-hover:opacity-100" style={{ background: 'linear-gradient(135deg, #00e5ff, #a900ff)', padding: '1px', WebkitMask: 'linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0)', WebkitMaskComposite: 'xor', maskComposite: 'exclude' }} />

                            <CardHeader className="pb-3">
                                <div className="brand-gradient mb-4 flex h-12 w-12 items-center justify-center rounded-xl text-white shadow-md transition-transform duration-300 group-hover:scale-110">
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
        <section
            id="agents"
            className="relative scroll-mt-20 overflow-hidden py-28 sm:py-36"
        >
            <div className="absolute inset-0 -z-10 bg-muted/30" />
            {/* Subtle gradient accent */}
            <div className="absolute right-0 top-0 -z-10 h-96 w-96 rounded-full bg-[radial-gradient(circle,oklch(0.55_0.27_295/0.04),transparent)]" />

            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="animate-on-scroll mx-auto max-w-3xl text-center">
                    <Badge
                        variant="outline"
                        className="mb-4 border-primary/20"
                    >
                        <i className="bx bx-bot mr-1 text-primary" />
                        AI Agents
                    </Badge>
                    <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl lg:text-5xl">
                        <span className="brand-gradient-text">16 agents</span>{' '}
                        at your command
                    </h2>
                    <p className="mt-5 text-lg leading-relaxed text-muted-foreground">
                        Each agent is an expert in its domain. They work
                        autonomously on complex tasks, then report back with
                        results.
                    </p>
                </div>
                <div className="stagger-children mx-auto mt-20 grid max-w-5xl gap-5 sm:grid-cols-2 lg:grid-cols-4">
                    {agents.map((a) => (
                        <div
                            key={a.name}
                            className="group flex flex-col items-center rounded-2xl border border-border/50 bg-card/80 p-6 text-center backdrop-blur-sm transition-all duration-300 hover:-translate-y-1.5 hover:shadow-xl"
                        >
                            {/* Gradient ring around icon */}
                            <div className="gradient-ring mb-4 rounded-2xl p-[2px]">
                                <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-card transition-colors group-hover:bg-muted/50">
                                    <i
                                        className={`${a.icon} text-2xl brand-gradient-text`}
                                    />
                                </div>
                            </div>
                            <div className="text-sm font-semibold text-foreground">
                                {a.name}
                            </div>
                            <div className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
                                {a.desc}
                            </div>
                        </div>
                    ))}
                </div>
                <p className="animate-on-scroll mt-10 text-center text-sm text-muted-foreground">
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
        <section id="tech" className="scroll-mt-20 border-t border-border/50 py-28 sm:py-36">
            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="animate-on-scroll mx-auto max-w-3xl text-center">
                    <Badge
                        variant="outline"
                        className="mb-4 border-primary/20"
                    >
                        <i className="bx bx-chip mr-1 text-primary" />
                        Tech Stack
                    </Badge>
                    <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl lg:text-5xl">
                        Built on{' '}
                        <span className="brand-gradient-text">
                            battle-tested
                        </span>{' '}
                        technology
                    </h2>
                    <p className="mt-5 text-lg leading-relaxed text-muted-foreground">
                        Go for the API layer. Rust for the engine. React for
                        every frontend. No compromises.
                    </p>
                </div>

                {/* Horizontal scrolling pills */}
                <div className="animate-on-scroll mx-auto mt-16 max-w-4xl">
                    <div className="tech-scroll justify-center px-4 sm:flex-wrap">
                        {techStack.map((t) => (
                            <div
                                key={t.label}
                                className="group flex items-center gap-3 rounded-full border border-border/50 bg-card/80 px-5 py-3 backdrop-blur-sm transition-all duration-300 hover:border-transparent hover:shadow-lg"
                                style={{ minWidth: 'fit-content' }}
                            >
                                {/* Hover gradient border */}
                                <div className="absolute inset-0 rounded-full opacity-0 transition-opacity duration-300 group-hover:opacity-100" style={{ background: 'linear-gradient(135deg, #00e5ff, #a900ff)', padding: '1px', WebkitMask: 'linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0)', WebkitMaskComposite: 'xor', maskComposite: 'exclude' }} />
                                <div className="brand-gradient flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-white shadow-sm">
                                    <i className={`${t.icon} text-lg`} />
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
            </div>
        </section>
    );
}

function WorkflowSection() {
    const steps = [
        {
            icon: 'bx bx-list-plus',
            label: 'Backlog',
        },
        {
            icon: 'bx bx-check-circle',
            label: 'Todo',
        },
        {
            icon: 'bx bx-loader-alt',
            label: 'In Progress',
        },
        {
            icon: 'bx bx-test-tube',
            label: 'Testing',
        },
        {
            icon: 'bx bx-file',
            label: 'Docs',
        },
        {
            icon: 'bx bx-search-alt',
            label: 'Review',
        },
        {
            icon: 'bx bx-check-double',
            label: 'Done',
        },
    ];

    return (
        <section className="relative overflow-hidden py-28 sm:py-36">
            <div className="absolute inset-0 -z-10 bg-muted/30" />

            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="animate-on-scroll mx-auto max-w-3xl text-center">
                    <Badge
                        variant="outline"
                        className="mb-4 border-primary/20"
                    >
                        <i className="bx bx-git-merge mr-1 text-primary" />
                        Workflow
                    </Badge>
                    <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl lg:text-5xl">
                        Quality{' '}
                        <span className="brand-gradient-text">
                            built into
                        </span>{' '}
                        the process
                    </h2>
                    <p className="mt-5 text-lg leading-relaxed text-muted-foreground">
                        Every task flows through testing, documentation, and
                        review gates. 13 states. No shortcuts.
                    </p>
                </div>
                <div className="stagger-children mx-auto mt-16 flex max-w-5xl flex-wrap items-center justify-center gap-3">
                    {steps.map((s, i) => (
                        <div key={s.label} className="flex items-center gap-3">
                            <div className="group relative flex items-center gap-2 rounded-full px-5 py-2.5 text-sm font-medium text-foreground transition-all duration-300 hover:-translate-y-0.5 hover:shadow-md">
                                {/* Gradient border outline */}
                                <div
                                    className="absolute inset-0 rounded-full"
                                    style={{
                                        background: 'linear-gradient(135deg, #00e5ff, #a900ff)',
                                        padding: '1.5px',
                                        WebkitMask: 'linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0)',
                                        WebkitMaskComposite: 'xor',
                                        maskComposite: 'exclude',
                                    }}
                                />
                                <i className={`${s.icon} brand-gradient-text`} />
                                {s.label}
                            </div>
                            {i < steps.length - 1 && (
                                <i className="bx bx-right-arrow-alt brand-gradient-text text-xl" />
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
        <section
            id="pricing"
            className="scroll-mt-20 border-t border-border/50 py-28 sm:py-36"
        >
            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="animate-on-scroll mx-auto max-w-3xl text-center">
                    <Badge
                        variant="outline"
                        className="mb-4 border-primary/20"
                    >
                        <i className="bx bxl-github mr-1 text-primary" />
                        GitHub Sponsors
                    </Badge>
                    <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl lg:text-5xl">
                        Powered by{' '}
                        <span className="brand-gradient-text">sponsors</span>
                    </h2>
                    <p className="mt-5 text-lg leading-relaxed text-muted-foreground">
                        Orchestra MCP is free to use. Sponsor us on GitHub to
                        unlock cloud AI, unlimited sync, and premium features.
                    </p>
                </div>
                <div className="stagger-children mx-auto mt-20 grid max-w-5xl gap-6 sm:grid-cols-2 lg:grid-cols-3">
                    {plans.map((plan) => (
                        <Card
                            key={plan.name}
                            className={`glass-card relative flex flex-col border-border/50 transition-all duration-300 hover:-translate-y-1.5 ${
                                plan.highlighted
                                    ? 'animate-pulse-glow mt-4 border-transparent shadow-xl'
                                    : 'hover:shadow-xl'
                            }`}
                        >
                            {/* Gradient border for highlighted plan */}
                            {plan.highlighted && (
                                <div
                                    className="absolute inset-0 rounded-[inherit]"
                                    style={{
                                        background: 'linear-gradient(135deg, #00e5ff, #a900ff)',
                                        padding: '1.5px',
                                        WebkitMask: 'linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0)',
                                        WebkitMaskComposite: 'xor',
                                        maskComposite: 'exclude',
                                    }}
                                />
                            )}
                            {plan.highlighted && (
                                <div className="absolute -top-3.5 left-1/2 z-10 -translate-x-1/2">
                                    <Badge className="brand-gradient border-0 px-4 text-white shadow-md">
                                        Recommended
                                    </Badge>
                                </div>
                            )}
                            <CardHeader className="relative">
                                <CardTitle className="text-lg">
                                    {plan.name}
                                </CardTitle>
                                <div className="mt-4 flex items-baseline gap-1">
                                    <span className={`text-5xl font-extrabold tracking-tight ${plan.highlighted ? 'brand-gradient-text' : 'text-foreground'}`}>
                                        {plan.price}
                                    </span>
                                    <span className="text-sm text-muted-foreground">
                                        {plan.period}
                                    </span>
                                </div>
                                <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
                                    {plan.description}
                                </p>
                            </CardHeader>
                            <CardContent className="relative flex flex-1 flex-col">
                                <Separator className="mb-6" />
                                <ul className="flex-1 space-y-3.5 text-sm">
                                    {plan.features.map((feature) => (
                                        <li
                                            key={feature}
                                            className="flex items-start gap-3 text-muted-foreground"
                                        >
                                            <i className="bx bx-check mt-0.5 text-lg text-primary" />
                                            {feature}
                                        </li>
                                    ))}
                                </ul>
                                {plan.ctaHref === 'github' ? (
                                    <Button
                                        asChild
                                        className={`mt-8 w-full gap-2 ${
                                            plan.highlighted
                                                ? 'brand-gradient border-0 text-white shadow-md hover:opacity-90'
                                                : ''
                                        }`}
                                        variant={plan.highlighted ? 'default' : 'outline'}
                                    >
                                        <a href={GITHUB_SPONSORS_URL} target="_blank" rel="noopener noreferrer">
                                            <i className="bx bxl-github" />
                                            {plan.cta}
                                        </a>
                                    </Button>
                                ) : (
                                    <Button
                                        asChild
                                        className="mt-8 w-full"
                                        variant="outline"
                                    >
                                        <Link href={register()}>{plan.cta}</Link>
                                    </Button>
                                )}
                            </CardContent>
                        </Card>
                    ))}
                </div>
                <p className="animate-on-scroll mt-10 text-center text-sm text-muted-foreground">
                    After sponsoring, your account is activated manually by an admin.
                    Already a sponsor?{' '}
                    <Link href={login()} className="text-foreground underline underline-offset-4 hover:text-primary">
                        Log in
                    </Link>{' '}
                    to check your status.
                </p>
            </div>
        </section>
    );
}

function CTASection() {
    return (
        <section className="cta-gradient-bg shimmer relative py-24">
            <div className="animate-on-scroll-scale mx-auto max-w-4xl px-4 text-center sm:px-6 lg:px-8">
                <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl lg:text-5xl">
                    Ready to orchestrate your workflow?
                </h2>
                <p className="mx-auto mt-6 max-w-2xl text-lg leading-relaxed text-white/80">
                    Join developers who are shipping faster with AI agents,
                    real-time sync, and a 13-state workflow engine.
                </p>
                <div className="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row">
                    <Button
                        asChild
                        size="lg"
                        variant="secondary"
                        className="gap-2 px-8 text-base font-semibold shadow-lg"
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
                        className="gap-2 border-white/30 bg-white/5 px-8 text-base text-white backdrop-blur-sm hover:bg-white/15 hover:text-white"
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
        <footer className="relative border-t border-border/50 pt-0">
            {/* Brand gradient top line */}
            <div className="brand-gradient h-px w-full" />

            <div className="mx-auto max-w-7xl px-4 pb-12 pt-16 sm:px-6 lg:px-8">
                <div className="grid gap-12 sm:grid-cols-2 lg:grid-cols-4">
                    <div className="lg:col-span-1">
                        <div className="flex items-center gap-2.5 text-lg font-bold text-foreground">
                            <AppLogoIcon className="size-6" />
                            Orchestra MCP
                        </div>
                        <p className="mt-4 max-w-xs text-sm leading-relaxed text-muted-foreground">
                            The AI-Agentic IDE for modern development. Build,
                            manage, and ship across every platform.
                        </p>
                        <div className="mt-6 flex items-center gap-3">
                            <a
                                href="https://github.com/orchestra-mcp"
                                className="flex h-9 w-9 items-center justify-center rounded-lg border border-border/50 text-muted-foreground transition-all hover:border-primary/30 hover:text-foreground"
                            >
                                <i className="bx bxl-github text-lg" />
                            </a>
                            <a
                                href="https://discord.gg/orchestra-mcp"
                                className="flex h-9 w-9 items-center justify-center rounded-lg border border-border/50 text-muted-foreground transition-all hover:border-primary/30 hover:text-foreground"
                            >
                                <i className="bx bxl-discord-alt text-lg" />
                            </a>
                            <a
                                href="https://twitter.com/orchestra_mcp"
                                className="flex h-9 w-9 items-center justify-center rounded-lg border border-border/50 text-muted-foreground transition-all hover:border-primary/30 hover:text-foreground"
                            >
                                <i className="bx bxl-twitter text-lg" />
                            </a>
                        </div>
                    </div>
                    <div>
                        <h3 className="text-sm font-semibold tracking-wide text-foreground">
                            Product
                        </h3>
                        <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                            <li>
                                <a
                                    href="#features"
                                    onClick={handleAnchorClick}
                                    className="transition-colors hover:text-foreground"
                                >
                                    Features
                                </a>
                            </li>
                            <li>
                                <a
                                    href="#pricing"
                                    onClick={handleAnchorClick}
                                    className="transition-colors hover:text-foreground"
                                >
                                    Pricing
                                </a>
                            </li>
                            <li>
                                <a
                                    href="#tech"
                                    onClick={handleAnchorClick}
                                    className="transition-colors hover:text-foreground"
                                >
                                    Tech Stack
                                </a>
                            </li>
                        </ul>
                    </div>
                    <div>
                        <h3 className="text-sm font-semibold tracking-wide text-foreground">
                            Developers
                        </h3>
                        <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                            <li>
                                <a
                                    href="https://github.com/orchestra-mcp"
                                    className="transition-colors hover:text-foreground"
                                >
                                    GitHub
                                </a>
                            </li>
                            <li>
                                <a
                                    href="#"
                                    className="transition-colors hover:text-foreground"
                                >
                                    Documentation
                                </a>
                            </li>
                            <li>
                                <a
                                    href="#"
                                    className="transition-colors hover:text-foreground"
                                >
                                    Extensions
                                </a>
                            </li>
                        </ul>
                    </div>
                    <div>
                        <h3 className="text-sm font-semibold tracking-wide text-foreground">
                            Company
                        </h3>
                        <ul className="mt-4 space-y-3 text-sm text-muted-foreground">
                            <li>
                                <a
                                    href="#"
                                    className="transition-colors hover:text-foreground"
                                >
                                    About
                                </a>
                            </li>
                            <li>
                                <a
                                    href="#"
                                    className="transition-colors hover:text-foreground"
                                >
                                    Privacy
                                </a>
                            </li>
                            <li>
                                <a
                                    href="#"
                                    className="transition-colors hover:text-foreground"
                                >
                                    Terms
                                </a>
                            </li>
                        </ul>
                    </div>
                </div>
                <Separator className="my-10" />
                <div className="flex flex-col items-center justify-between gap-4 text-sm text-muted-foreground sm:flex-row">
                    <p>
                        &copy; {new Date().getFullYear()} Orchestra MCP. All
                        rights reserved.
                    </p>
                    <p className="text-xs text-muted-foreground/60">
                        Built with Go, Rust, and React
                    </p>
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
