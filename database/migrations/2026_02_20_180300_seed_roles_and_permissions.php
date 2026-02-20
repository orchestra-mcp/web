<?php

use Illuminate\Database\Migrations\Migration;
use Spatie\Permission\Models\Permission;
use Spatie\Permission\Models\Role;

return new class extends Migration
{
    public function up(): void
    {
        app()[\Spatie\Permission\PermissionRegistrar::class]->forgetCachedPermissions();

        $permissions = [
            'view-admin-panel',
            'manage-users',
            'manage-roles',
            'manage-subscriptions',
            'manage-pages',
            'access-ide-features',
        ];

        foreach ($permissions as $permission) {
            Permission::create(['name' => $permission]);
        }

        $superAdmin = Role::create(['name' => 'super_admin']);
        $superAdmin->givePermissionTo(Permission::all());

        $admin = Role::create(['name' => 'admin']);
        $admin->givePermissionTo([
            'view-admin-panel',
            'manage-users',
            'manage-roles',
            'manage-subscriptions',
            'manage-pages',
        ]);

        $subscriber = Role::create(['name' => 'subscriber']);
        $subscriber->givePermissionTo(['access-ide-features']);

        Role::create(['name' => 'user']);
    }

    public function down(): void
    {
        app()[\Spatie\Permission\PermissionRegistrar::class]->forgetCachedPermissions();

        Role::whereIn('name', ['super_admin', 'admin', 'subscriber', 'user'])->delete();
        Permission::whereIn('name', [
            'view-admin-panel', 'manage-users', 'manage-roles',
            'manage-subscriptions', 'manage-pages', 'access-ide-features',
        ])->delete();
    }
};
