<?php

namespace App\Notifications;

use App\Models\Subscription;
use Illuminate\Bus\Queueable;
use Illuminate\Notifications\Notification;

class SubscriptionExpiringNotification extends Notification
{
    use Queueable;

    public function __construct(
        private Subscription $subscription,
    ) {}

    /** @return list<string> */
    public function via(object $notifiable): array
    {
        return ['database'];
    }

    /** @return array<string, mixed> */
    public function toArray(object $notifiable): array
    {
        return [
            'type' => 'subscription_expiring',
            'title' => 'Subscription Expiring Soon',
            'message' => "Subscription for {$this->subscription->user->name} expires on {$this->subscription->end_date->format('M d, Y')}.",
            'subscription_id' => $this->subscription->id,
            'user_id' => $this->subscription->user_id,
        ];
    }
}
