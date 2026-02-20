<?php

use App\Models\User;

test('blocked user cannot login', function () {
    $user = User::factory()->blocked()->create();

    $response = $this->post(route('login.store'), [
        'email' => $user->email,
        'password' => 'password',
    ]);

    $this->assertGuest();
    $response->assertSessionHasErrors('email');
});

test('blocked user is logged out on next request', function () {
    $user = User::factory()->create();
    $this->actingAs($user);

    $user->update(['status' => 'blocked']);

    $response = $this->get(route('dashboard'));
    $response->assertRedirect(route('login'));
    $this->assertGuest();
});
