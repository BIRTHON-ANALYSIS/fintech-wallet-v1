from django.db import models
from django.contrib.auth.models import AbstractUser
from django.conf import settings
from django.utils import timezone
from datetime import timedelta


class User(AbstractUser):
    class Role(models.TextChoices):
        CUSTOMER = "customer", "Customer"
        ADMIN = "admin", "Admin"
        SUPPORT = "support", "Support"

    role = models.CharField(
        max_length=20,
        choices=Role.choices,
        default=Role.CUSTOMER,
    )

    phone_number = models.CharField(max_length=20,unique=True,null=True,blank=True,)
    is_email_verified = models.BooleanField(default=False,)
    created_at = models.DateTimeField(auto_now_add=True,)
    updated_at = models.DateTimeField(
    auto_now=True,)

    def __str__(self):
        full_name = f"{self.first_name} {self.last_name}".strip()
        return full_name or self.email

    failed_login_attempts = models.PositiveIntegerField(default=0)

    account_locked_until = models.DateTimeField(
        null=True,
        blank=True
    )

    def is_account_locked(self):
        if self.account_locked_until is None:
            return False

        if timezone.now() >= self.account_locked_until:
            self.failed_login_attempts = 0
            self.account_locked_until = None
            self.save(
                update_fields=[
                    "failed_login_attempts",
                    "account_locked_until",
                ]
             )
            return False

        return True

    def record_failed_login(self):
        self.failed_login_attempts += 1

        if self.failed_login_attempts >= 3:
            self.account_locked_until = (
                timezone.now() + timedelta(minutes=15)
            )

        self.save(
            update_fields=[
                "failed_login_attempts",
                "account_locked_until",
            ]
        )

    def reset_login_attempts(self):
        self.failed_login_attempts = 0
        self.account_locked_until = None

        self.save(
            update_fields=[
                "failed_login_attempts",
                "account_locked_until",
            ]
        )

class EmailOTP(models.Model):
    user = models.ForeignKey(
        User,
        on_delete=models.CASCADE,
        related_name="email_otps",
    )

    code = models.CharField(max_length=6)

    expires_at = models.DateTimeField()

    is_used = models.BooleanField(default=False)

    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        ordering = ["-created_at"]

    def __str__(self):
        return f"OTP for {self.user.email}"


