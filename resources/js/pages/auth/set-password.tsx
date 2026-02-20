import { useForm, Head } from '@inertiajs/react';
import InputError from '@/components/input-error';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Spinner } from '@/components/ui/spinner';
import AuthLayout from '@/layouts/auth-layout';

export default function SetPassword() {
    const { data, setData, post, processing, errors } = useForm({
        password: '',
        password_confirmation: '',
    });

    const submit = (e: React.FormEvent) => {
        e.preventDefault();
        post('/set-password');
    };

    return (
        <AuthLayout
            title="Set your password"
            description="Create a password for IDE access"
        >
            <Head title="Set Password" />

            <p className="text-sm text-muted-foreground">
                You signed in with a social account. Please create a password to use the Orchestra IDE desktop app.
            </p>

            <form onSubmit={submit} className="flex flex-col gap-6">
                <div className="grid gap-6">
                    <div className="grid gap-2">
                        <Label htmlFor="password">Password</Label>
                        <Input
                            id="password"
                            type="password"
                            required
                            minLength={8}
                            autoFocus
                            tabIndex={1}
                            autoComplete="new-password"
                            placeholder="Password"
                            value={data.password}
                            onChange={(e) => setData('password', e.target.value)}
                        />
                        <InputError message={errors.password} />
                    </div>

                    <div className="grid gap-2">
                        <Label htmlFor="password_confirmation">Confirm Password</Label>
                        <Input
                            id="password_confirmation"
                            type="password"
                            required
                            minLength={8}
                            tabIndex={2}
                            autoComplete="new-password"
                            placeholder="Confirm password"
                            value={data.password_confirmation}
                            onChange={(e) => setData('password_confirmation', e.target.value)}
                        />
                        <InputError message={errors.password_confirmation} />
                    </div>

                    <Button
                        type="submit"
                        className="mt-4 w-full"
                        tabIndex={3}
                        disabled={processing}
                    >
                        {processing && <Spinner />}
                        Set Password
                    </Button>
                </div>
            </form>
        </AuthLayout>
    );
}
