<?php

namespace App\Http\Controllers\Webhook;

use App\Http\Controllers\Controller;
use App\Models\Subscription;
use App\Models\User;
use App\Models\UserMeta;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class GitHubSponsorsController extends Controller
{
    public function handle(Request $request): JsonResponse
    {
        $this->verifySignature($request);

        $action = $request->input('action');
        $sponsorship = $request->input('sponsorship', []);
        $sponsor = $sponsorship['sponsor'] ?? [];
        $githubId = (string) ($sponsor['id'] ?? '');

        if (! $githubId) {
            return response()->json(['message' => 'No sponsor ID'], 400);
        }

        $meta = UserMeta::where('key', 'github_id')->where('value', $githubId)->first();
        $user = $meta?->user;

        if (! $user) {
            return response()->json(['message' => 'User not found'], 404);
        }

        match ($action) {
            'created' => $this->handleCreated($user, $sponsorship),
            'cancelled' => $this->handleCancelled($user),
            'tier_changed' => $this->handleTierChanged($user, $sponsorship),
            'pending_cancellation' => $this->handlePendingCancellation($user),
            default => null,
        };

        return response()->json(['message' => 'Processed']);
    }

    /** @param array<string, mixed> $sponsorship */
    private function handleCreated(User $user, array $sponsorship): void
    {
        $tier = $sponsorship['tier'] ?? [];
        $amount = (int) (($tier['monthly_price_in_cents'] ?? 0));

        Subscription::updateOrCreate(
            ['user_id' => $user->id],
            [
                'plan' => $this->mapTierToPlan($amount),
                'status' => 'active',
                'start_date' => now(),
                'github_sponsor_id' => (string) ($sponsorship['sponsor']['id'] ?? ''),
                'amount_cents' => $amount,
                'last_payment_at' => now(),
            ],
        );

        $user->assignRole('subscriber');
    }

    private function handleCancelled(User $user): void
    {
        $user->subscription?->update(['status' => 'cancelled']);
    }

    /** @param array<string, mixed> $sponsorship */
    private function handleTierChanged(User $user, array $sponsorship): void
    {
        $tier = $sponsorship['tier'] ?? [];
        $amount = (int) ($tier['monthly_price_in_cents'] ?? 0);

        $user->subscription?->update([
            'plan' => $this->mapTierToPlan($amount),
            'amount_cents' => $amount,
        ]);
    }

    private function handlePendingCancellation(User $user): void
    {
        $user->subscription?->update(['status' => 'past_due']);
    }

    private function mapTierToPlan(int $amountCents): string
    {
        return match (true) {
            $amountCents >= 2500 => 'team_sponsor',
            $amountCents >= 500 => 'sponsor',
            default => 'free',
        };
    }

    private function verifySignature(Request $request): void
    {
        $secret = config('services.github.webhook_secret');

        if (! $secret) {
            return;
        }

        $signature = $request->header('X-Hub-Signature-256');

        if (! $signature) {
            abort(403, 'Missing signature');
        }

        $hash = 'sha256='.hash_hmac('sha256', $request->getContent(), $secret);

        if (! hash_equals($hash, $signature)) {
            abort(403, 'Invalid signature');
        }
    }
}
