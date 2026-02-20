<?php

use App\Models\Subscription;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;

uses(Tests\TestCase::class, RefreshDatabase::class);

test('subscription can check if active', function () {
    $user = User::factory()->create();
    $sub = Subscription::create([
        'user_id' => $user->id,
        'plan' => 'pro',
        'status' => 'active',
        'start_date' => now(),
        'end_date' => now()->addMonth(),
        'amount_cents' => 500,
    ]);

    expect($sub->isActive())->toBeTrue();
    expect($sub->isExpired())->toBeFalse();
});

test('subscription expiring soon scope works', function () {
    $user = User::factory()->create();
    Subscription::create([
        'user_id' => $user->id,
        'plan' => 'pro',
        'status' => 'active',
        'start_date' => now()->subMonth(),
        'end_date' => now()->addDays(3),
        'amount_cents' => 500,
    ]);

    expect(Subscription::expiringSoon()->count())->toBe(1);
});
