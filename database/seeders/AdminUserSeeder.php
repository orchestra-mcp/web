<?php

namespace Database\Seeders;

use App\Models\User;
use Illuminate\Database\Seeder;

class AdminUserSeeder extends Seeder
{
    public function run(): void
    {
        $admin = User::firstOrCreate(
            ['email' => 'admin@orchestra-mcp.com'],
            [
                'name' => 'Admin',
                'password' => 'password',
                'password_set' => true,
                'email_verified_at' => now(),
            ],
        );

        $admin->assignRole('super_admin');
    }
}
