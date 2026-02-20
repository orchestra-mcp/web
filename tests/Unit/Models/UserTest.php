<?php

use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;

uses(Tests\TestCase::class, RefreshDatabase::class);

test('user can check if blocked', function () {
    $user = User::factory()->blocked()->make();
    expect($user->isBlocked())->toBeTrue();

    $active = User::factory()->make();
    expect($active->isBlocked())->toBeFalse();
});

test('user can check if needs password', function () {
    $socialUser = User::factory()->socialLogin()->make();
    expect($socialUser->needsPassword())->toBeTrue();

    $normalUser = User::factory()->make();
    expect($normalUser->needsPassword())->toBeFalse();
});

test('user active scope works', function () {
    User::factory()->create(['status' => 'active']);
    User::factory()->blocked()->create();

    expect(User::active()->count())->toBe(1);
    expect(User::blocked()->count())->toBe(1);
});
