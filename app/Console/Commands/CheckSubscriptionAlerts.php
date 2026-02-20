<?php

namespace App\Console\Commands;

use App\Models\Subscription;
use App\Models\User;
use App\Notifications\SubscriptionExpiredNotification;
use App\Notifications\SubscriptionExpiringNotification;
use Illuminate\Console\Command;

class CheckSubscriptionAlerts extends Command
{
    protected $signature = 'subscriptions:check-alerts';

    protected $description = 'Check for expired and expiring subscriptions and notify admins';

    public function handle(): int
    {
        $admins = User::role(['admin', 'super_admin'])->get();

        $expiring = Subscription::with('user')
            ->expiringSoon()
            ->whereNull('alert_sent_at')
            ->get();

        foreach ($expiring as $subscription) {
            foreach ($admins as $admin) {
                $admin->notify(new SubscriptionExpiringNotification($subscription));
            }
            $subscription->update(['alert_sent_at' => now()]);
        }

        $expired = Subscription::with('user')
            ->where('status', 'active')
            ->whereNotNull('end_date')
            ->where('end_date', '<', now())
            ->get();

        foreach ($expired as $subscription) {
            $subscription->update(['status' => 'expired']);
            foreach ($admins as $admin) {
                $admin->notify(new SubscriptionExpiredNotification($subscription));
            }
        }

        $this->info("Processed {$expiring->count()} expiring, {$expired->count()} expired.");

        return self::SUCCESS;
    }
}
