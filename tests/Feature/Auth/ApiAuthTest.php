<?php

use App\Models\User;

test('user can login via API and receive token', function () {
    $user = User::factory()->create();

    $response = $this->postJson('/api/auth/login', [
        'email' => $user->email,
        'password' => 'password',
    ]);

    $response->assertOk()
        ->assertJsonStructure(['token']);
});

test('blocked user cannot login via API', function () {
    $user = User::factory()->blocked()->create();

    $response = $this->postJson('/api/auth/login', [
        'email' => $user->email,
        'password' => 'password',
    ]);

    $response->assertUnprocessable();
});

test('user without password cannot login via API', function () {
    $user = User::factory()->socialLogin()->create();

    $response = $this->postJson('/api/auth/login', [
        'email' => $user->email,
        'password' => '',
    ]);

    $response->assertUnprocessable();
});

test('authenticated user can get their info via API', function () {
    $user = User::factory()->create();
    $token = $user->createToken('test', ['ide:access'])->plainTextToken;

    $response = $this->withHeader('Authorization', "Bearer $token")
        ->getJson('/api/auth/user');

    $response->assertOk()
        ->assertJsonPath('user.id', $user->id)
        ->assertJsonPath('user.email', $user->email);
});

test('user can logout via API', function () {
    $user = User::factory()->create();
    $token = $user->createToken('test', ['ide:access'])->plainTextToken;

    $response = $this->withHeader('Authorization', "Bearer $token")
        ->postJson('/api/auth/logout');

    $response->assertOk();
    expect($user->tokens()->count())->toBe(0);
});
