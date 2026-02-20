import { Head, Link, router } from '@inertiajs/react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import AppLogoIcon from '@/components/app-logo-icon';

const GITHUB_SPONSORS_URL = 'https://github.com/sponsors/fadymondy';

const features = [
    'Cloud AI (Claude + GPT)',
    'Unlimited synced devices',
    'RAG memory system',
    'Priority support',
    'All 26 themes',
    'Team collaboration',
];

export default function Subscribe() {
    return (
        <>
            <Head title="Subscribe — Orchestra MCP" />
            <div className="relative flex min-h-svh flex-col items-center justify-center gap-6 overflow-hidden bg-background p-6 md:p-10">
                {/* Gradient mesh background */}
                <div className="pointer-events-none absolute inset-0 overflow-hidden">
                    <div className="absolute -right-32 -top-32 size-96 rounded-full bg-[var(--brand-cyan)]/8 blur-[100px] dark:bg-[var(--brand-cyan)]/15" />
                    <div className="absolute -bottom-32 -left-32 size-96 rounded-full bg-[var(--brand-purple)]/8 blur-[100px] dark:bg-[var(--brand-purple)]/15" />
                    <div className="absolute left-1/2 top-1/3 size-72 -translate-x-1/2 rounded-full bg-[var(--brand-purple)]/4 blur-[120px] dark:bg-[var(--brand-purple)]/8" />
                </div>

                <div className="relative w-full max-w-lg">
                    <Card className="glass-card border-border/40 shadow-lg dark:border-border/20">
                        <CardHeader className="text-center">
                            <div className="mx-auto mb-4 flex size-16 items-center justify-center">
                                <AppLogoIcon className="size-16" />
                            </div>
                            <CardTitle className="text-2xl">
                                <span className="brand-gradient-text">Subscription Required</span>
                            </CardTitle>
                            <CardDescription className="text-base">
                                Sponsor Orchestra MCP on GitHub to access the full IDE experience.
                            </CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-6">
                            <div className="rounded-xl border border-border/50 bg-muted/30 p-5">
                                <div className="flex items-baseline gap-2">
                                    <span className="brand-gradient-text text-4xl font-extrabold tracking-tight">$5</span>
                                    <span className="text-sm text-muted-foreground">/month</span>
                                </div>
                                <p className="mt-2 text-sm text-muted-foreground">
                                    via GitHub Sponsors
                                </p>
                            </div>

                            <Separator />

                            <ul className="space-y-3 text-sm">
                                {features.map((feature) => (
                                    <li key={feature} className="flex items-center gap-3 text-muted-foreground">
                                        <i className="bx bx-check text-lg text-primary" />
                                        {feature}
                                    </li>
                                ))}
                            </ul>

                            <Button
                                asChild
                                size="lg"
                                className="brand-gradient w-full gap-2 border-0 text-base text-white shadow-lg hover:opacity-90"
                            >
                                <a href={GITHUB_SPONSORS_URL} target="_blank" rel="noopener noreferrer">
                                    <i className="bx bxl-github text-lg" />
                                    Sponsor on GitHub
                                </a>
                            </Button>

                            <p className="text-center text-xs text-muted-foreground">
                                After sponsoring, an admin will activate your account.
                                <br />
                                Already a sponsor?{' '}
                                <Link href="/subscription" className="text-foreground underline underline-offset-4 hover:text-primary">
                                    Check your status
                                </Link>
                            </p>

                            <Separator />

                            <div className="flex items-center justify-between text-xs text-muted-foreground">
                                <Badge variant="secondary">Free tier users</Badge>
                                <button
                                    type="button"
                                    onClick={() => router.post('/logout')}
                                    className="text-muted-foreground underline underline-offset-4 hover:text-foreground"
                                >
                                    Log out
                                </button>
                            </div>
                        </CardContent>
                    </Card>
                </div>
            </div>
        </>
    );
}
