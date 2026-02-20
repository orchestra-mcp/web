<?php

use App\Models\User;
use Spatie\Permission\Models\Role;

beforeEach(function () {
    $this->admin = User::factory()->create();
    $this->admin->assignRole('super_admin');
});

test('admin can view roles list', function () {
    $response = $this->actingAs($this->admin)->get(route('admin.roles.index'));
    $response->assertOk();
});

test('admin can create role', function () {
    $response = $this->actingAs($this->admin)->post(route('admin.roles.store'), [
        'name' => 'moderator',
        'permissions' => ['manage-users'],
    ]);

    $response->assertRedirect(route('admin.roles.index'));
    expect(Role::where('name', 'moderator')->exists())->toBeTrue();
});

test('admin cannot delete protected roles', function () {
    $role = Role::where('name', 'admin')->first();

    $response = $this->actingAs($this->admin)->delete(route('admin.roles.destroy', $role));
    $response->assertSessionHasErrors('error');
    expect(Role::where('name', 'admin')->exists())->toBeTrue();
});
