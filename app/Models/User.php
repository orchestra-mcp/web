<?php

namespace App\Models;

use Illuminate\Contracts\Auth\MustVerifyEmail;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Database\Eloquent\Relations\HasOne;
use Illuminate\Database\Eloquent\SoftDeletes;
use Illuminate\Foundation\Auth\User as Authenticatable;
use Illuminate\Notifications\Notifiable;
use Laravel\Fortify\TwoFactorAuthenticatable;
use Laravel\Sanctum\HasApiTokens;
use Spatie\MediaLibrary\HasMedia;
use Spatie\MediaLibrary\InteractsWithMedia;
use Spatie\MediaLibrary\MediaCollections\Models\Media;
use Spatie\Permission\Traits\HasRoles;

class User extends Authenticatable implements HasMedia, MustVerifyEmail
{
    /** @use HasFactory<\Database\Factories\UserFactory> */
    use HasApiTokens, HasFactory, HasRoles, InteractsWithMedia, Notifiable, SoftDeletes, TwoFactorAuthenticatable;

    /** @var list<string> */
    protected $fillable = [
        'name',
        'email',
        'password',
        'status',
        'password_set',
    ];

    /** @var list<string> */
    protected $hidden = [
        'password',
        'two_factor_secret',
        'two_factor_recovery_codes',
        'remember_token',
    ];

    /** @return array<string, string> */
    protected function casts(): array
    {
        return [
            'email_verified_at' => 'datetime',
            'password' => 'hashed',
            'password_set' => 'boolean',
            'two_factor_confirmed_at' => 'datetime',
        ];
    }

    public function registerMediaCollections(): void
    {
        $this->addMediaCollection('avatar')->singleFile();
    }

    public function registerMediaConversions(?Media $media = null): void
    {
        $this->addMediaConversion('thumb')->width(64)->height(64)->sharpen(10);
        $this->addMediaConversion('medium')->width(256)->height(256);
    }

    /** @return HasMany<UserMeta, $this> */
    public function metas(): HasMany
    {
        return $this->hasMany(UserMeta::class);
    }

    /** @return HasOne<Subscription, $this> */
    public function subscription(): HasOne
    {
        return $this->hasOne(Subscription::class)->latestOfMany();
    }

    public function getMeta(string $key): ?string
    {
        return $this->metas()->where('key', $key)->value('value');
    }

    public function setMeta(string $key, ?string $value): void
    {
        $this->metas()->updateOrCreate(['key' => $key], ['value' => $value]);
    }

    public function isBlocked(): bool
    {
        return $this->status === 'blocked';
    }

    public function isAdmin(): bool
    {
        return $this->hasAnyRole(['admin', 'super_admin']);
    }

    public function isSuperAdmin(): bool
    {
        return $this->hasRole('super_admin');
    }

    public function needsPassword(): bool
    {
        return ! $this->password_set;
    }

    public function isSponsor(): bool
    {
        return $this->subscription?->isSponsor() ?? false;
    }

    public function hasActiveSubscription(): bool
    {
        $sub = $this->subscription;

        if (! $sub) {
            return false;
        }

        return in_array($sub->plan, ['sponsor', 'team_sponsor']) && $sub->isActive();
    }

    /** @param Builder<User> $query */
    public function scopeActive(Builder $query): void
    {
        $query->where('status', 'active');
    }

    /** @param Builder<User> $query */
    public function scopeBlocked(Builder $query): void
    {
        $query->where('status', 'blocked');
    }

    public function getAvatarUrlAttribute(): ?string
    {
        return $this->getFirstMediaUrl('avatar', 'thumb') ?: null;
    }
}
