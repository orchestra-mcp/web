<?php

namespace App\Console\Commands;

use App\Models\User;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\Hash;

class MakeAdmin extends Command
{
    protected $signature = 'make:admin';

    protected $description = 'Create a new super_admin user or promote an existing user';

    public function handle(): int
    {
        $email = $this->ask('Enter email address');
        $user = User::where('email', $email)->first();

        if ($user) {
            $user->assignRole('super_admin');
            $this->info("User [{$user->name}] promoted to super_admin.");

            return self::SUCCESS;
        }

        $name = $this->ask('Enter name');
        $password = $this->secret('Enter password');

        $user = User::create([
            'name' => $name,
            'email' => $email,
            'password' => $password,
            'password_set' => true,
            'email_verified_at' => now(),
        ]);

        $user->assignRole('super_admin');
        $this->info("Super admin [{$name}] created successfully.");

        return self::SUCCESS;
    }
}
