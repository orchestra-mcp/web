<?php

use App\Models\User;

beforeEach(function () {
    $this->admin = User::factory()->create();
    $this->admin->assignRole('super_admin');
});

test('admin can view users list', function () {
    $response = $this->actingAs($this->admin)->get(route('admin.users.index'));
    $response->assertOk();
});

test('non-admin cannot access admin panel', function () {
    $user = User::factory()->create();
    $user->assignRole('user');

    $response = $this->actingAs($user)->get(route('admin.users.index'));
    $response->assertForbidden();
});

test('admin can create user', function () {
    $response = $this->actingAs($this->admin)->post(route('admin.users.store'), [
        'name' => 'New User',
        'email' => 'new@example.com',
        'password' => 'password123',
        'role' => 'user',
    ]);

    $response->assertRedirect(route('admin.users.index'));
    expect(User::where('email', 'new@example.com')->exists())->toBeTrue();
});

test('admin can toggle user status', function () {
    $user = User::factory()->create();

    $this->actingAs($this->admin)->post(route('admin.users.toggle-status', $user));

    expect($user->fresh()->status)->toBe('blocked');

    $this->actingAs($this->admin)->post(route('admin.users.toggle-status', $user));

    expect($user->fresh()->status)->toBe('active');
});

test('admin cannot delete super admin', function () {
    $superAdmin = User::factory()->create();
    $superAdmin->assignRole('super_admin');

    $response = $this->actingAs($this->admin)->delete(route('admin.users.destroy', $superAdmin));
    $response->assertSessionHasErrors('error');
    expect(User::find($superAdmin->id))->not->toBeNull();
});
