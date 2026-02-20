<?php

namespace Database\Seeders;

use App\Models\User;
use Illuminate\Database\Seeder;

class TestUserSeeder extends Seeder
{
    public function run(): void
    {
        if (app()->environment('production')) {
            return;
        }

        $subscriber = User::firstOrCreate(
            ['email' => 'subscriber@example.com'],
            [
                'name' => 'Test Subscriber',
                'password' => 'password',
                'password_set' => true,
                'email_verified_at' => now(),
            ],
        );
        $subscriber->assignRole('subscriber');
        $subscriber->subscription()->firstOrCreate([], [
            'plan' => 'sponsor',
            'status' => 'active',
            'start_date' => now(),
            'end_date' => now()->addYear(),
            'amount_cents' => 500,
        ]);

        $user = User::firstOrCreate(
            ['email' => 'user@example.com'],
            [
                'name' => 'Test User',
                'password' => 'password',
                'password_set' => true,
                'email_verified_at' => now(),
            ],
        );
        $user->assignRole('user');
    }
}
