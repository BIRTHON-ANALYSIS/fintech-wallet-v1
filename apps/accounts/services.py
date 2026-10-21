
import secrets
from django.conf import settings
from django.core.mail import send_mail
from django.contrib.auth import get_user_model
from django.utils import timezone
from datetime import timedelta

from .models import EmailOTP
import secrets
from .models import EmailOTP
import redis


User = get_user_model()




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



class OTPService:

    OTP_EXPIRY = 300
    OTP_LENGTH = 6

    @staticmethod
    def get_redis_client():
        return redis.from_url(
            settings.REDIS_URL,
            decode_responses=True,
        )

    @staticmethod
    def generate_otp():
        return str(
            secrets.randbelow(900000) + 100000
        )

    @classmethod
    def create_and_store_otp(cls, user):
        otp = cls.generate_otp()

        redis_client = cls.get_redis_client()

        key = f"otp:user:{user.id}"

        redis_client.setex(
            key,
            cls.OTP_EXPIRY,
            otp,
        )

        return otp

    @classmethod
    def verify_otp(cls, user, submitted_otp):
        redis_client = cls.get_redis_client()

        key = f"otp:user:{user.id}"

        stored_otp = redis_client.get(key)

        if stored_otp is None:
            return False

        if not secrets.compare_digest(
            stored_otp,
            submitted_otp,
        ):
            return False

        redis_client.delete(key)

        return True

    @classmethod
    def send_otp(cls, user, otp):

        send_mail(
            subject="Your verification code",
            message=(
                f"Your OTP is {otp}. "
                f"It expires in 5 minutes."
            ),
            from_email=settings.DEFAULT_FROM_EMAIL,
            recipient_list=[user.email],
            fail_silently=False,
        )