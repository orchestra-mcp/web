<?php

namespace App\Notifications;

use Illuminate\Bus\Queueable;
use Illuminate\Notifications\Messages\MailMessage;
use Illuminate\Notifications\Notification;

class WelcomeNotification extends Notification
{
    use Queueable;

    /** @return list<string> */
    public function via(object $notifiable): array
    {
        return ['mail', 'database'];
    }

    public function toMail(object $notifiable): MailMessage
    {
        return (new MailMessage)
            ->subject('Welcome to Orchestra MCP')
            ->greeting('Welcome!')
            ->line('Thank you for joining Orchestra MCP.')
            ->line('Please set a password to use the IDE.')
            ->action('Set Password', url('/set-password'));
    }

    /** @return array<string, mixed> */
    public function toArray(object $notifiable): array
    {
        return [
            'type' => 'welcome',
            'title' => 'Welcome to Orchestra MCP',
            'message' => 'Your account has been created. Please set a password to use the IDE.',
        ];
    }
}
