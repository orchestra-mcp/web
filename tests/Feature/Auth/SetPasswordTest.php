<?php

use App\Models\User;

test('social login user is redirected to set password page', function () {
    $user = User::factory()->socialLogin()->create([
        'password' => bcrypt('temp'),
        'password_set' => false,
    ]);

    $response = $this->actingAs($user)->get(route('dashboard'));
    $response->assertRedirect(route('set-password'));
});

test('set password page can be rendered', function () {
    $user = User::factory()->socialLogin()->create([
        'password' => bcrypt('temp'),
        'password_set' => false,
    ]);

    $response = $this->actingAs($user)->get(route('set-password'));
    $response->assertOk();
});

test('social user can set password', function () {
    $user = User::factory()->socialLogin()->create([
        'password' => bcrypt('temp'),
        'password_set' => false,
    ]);

    $response = $this->actingAs($user)->post(route('set-password.store'), [
        'password' => 'newpassword123',
        'password_confirmation' => 'newpassword123',
    ]);

    $response->assertRedirect(route('dashboard'));
    expect($user->fresh()->password_set)->toBeTrue();
});

test('user with password set is redirected away from set-password', function () {
    $user = User::factory()->create();

    $response = $this->actingAs($user)->get(route('set-password'));
    $response->assertRedirect(route('dashboard'));
});
