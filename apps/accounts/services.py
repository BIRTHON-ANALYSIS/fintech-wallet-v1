
import secrets

from django.utils import timezone
from datetime import timedelta
from apps.accounts import serializers
from .models import EmailOTP
import secrets
from .models import EmailOTP


def generate_email_otp(user):
    EmailOTP.objects.filter(
        user=user,
        is_used=False,
    ).update(
        is_used=True
    )

    code = str(
        secrets.randbelow(900000) + 100000
    )

    expires_at = (
        timezone.now()
        + timedelta(minutes=10)
    )

    otp = EmailOTP.objects.create(
        user=user,
        code=code,
        expires_at=expires_at,
    )

    return otp

