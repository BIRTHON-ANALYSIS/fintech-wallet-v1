from .models import User
from django.contrib.auth import get_user_model
from rest_framework import serializers
from rest_framework_simplejwt.serializers import TokenObtainPairSerializer
from django.contrib.auth import get_user_model
from rest_framework import serializers
from .services import OTPService
from django.contrib.auth import authenticate
from rest_framework import serializers
from rest_framework_simplejwt.serializers import TokenObtainPairSerializer
from .models import User


User = get_user_model()

class RegisterSerializer(serializers.ModelSerializer):
    confirm_password = serializers.CharField(
        write_only=True,
    )

    password = serializers.CharField(
        write_only=True,    
        min_length=8,
    )

    class Meta:
        model = User
        fields = [
            "first_name",
            "last_name",
            "email",
            "phone_number",
            "password",
            "confirm_password",
        ]

    def validate(self, attrs):
        password = attrs.get("password")
        confirm_password = attrs.get("confirm_password")

        if password != confirm_password:
            raise serializers.ValidationError(
                {
                    "confirm_password": "Passwords do not match."
                }
            )

        return attrs

    def create(self, validated_data):
        validated_data.pop("confirm_password")

        user = User.objects.create_user(
            username=validated_data["email"],
            first_name=validated_data["first_name"],
            last_name=validated_data["last_name"],
            email=validated_data["email"],
            phone_number=validated_data["phone_number"],
            password=validated_data["password"],
        )

        return user


class UserProfileSerializer(serializers.ModelSerializer):
    class Meta:
        model = User
        fields = [
            "id",
            "username",
            "email",
            "first_name",
            "last_name",
        ]
        read_only_fields = fields

class VerifyEmailOTPSerializer(serializers.Serializer):
    email = serializers.EmailField()

    code = serializers.CharField(
        max_length=6,
        min_length=6,
    )

class ResendEmailOTPSerializer(serializers.Serializer):
    email = serializers.EmailField()



class LoginSerializer(TokenObtainPairSerializer):

    @classmethod
    def get_token(cls, user):
        token = super().get_token(user)

        token["username"] = user.username
        token["email"] = user.email

        return token



class LoginSerializer(TokenObtainPairSerializer):

    def validate(self, attrs):
        username = attrs.get("username")
        password = attrs.get("password")

        user = User.objects.filter(username=username).first()

        if user is None:
            raise serializers.ValidationError(
                "Invalid username or password."
            )

        if user.is_account_locked():
            raise serializers.ValidationError(
                "Account temporarily locked. Please try again later."
            )

        authenticated_user = authenticate(
            username=username,
            password=password
        )

        if authenticated_user is None:
            user.record_failed_login()

            raise serializers.ValidationError(
                "Invalid username or password."
            )

        user.reset_login_attempts()

        refresh = self.get_token(authenticated_user)

        return {
            "refresh": str(refresh),
            "access": str(refresh.access_token),
        }

    @classmethod
    def get_token(cls, user):
        token = super().get_token(user)

        token["username"] = user.username
        token["email"] = user.email

        return token





User = get_user_model()


class RegisterSerializer(serializers.ModelSerializer):

    password = serializers.CharField(
        write_only=True,
        min_length=8,
    )

    class Meta:
        model = User
        fields = [
            "username",
            "email",
            "password",
        ]

    def create(self, validated_data):

        user = User.objects.create_user(
            username=validated_data["username"],
            email=validated_data["email"],
            password=validated_data["password"],
            is_active=False,
        )

        otp = OTPService.create_and_store_otp(user)

        OTPService.send_otp(user, otp)

        return user


class VerifyOTPSerializer(serializers.Serializer):

    username = serializers.CharField()

    otp = serializers.CharField(
        min_length=6,
        max_length=6,
    )

    def validate(self, attrs):

        username = attrs["username"]
        submitted_otp = attrs["otp"]

        try:
            user = User.objects.get(
                username=username
            )
        except User.DoesNotExist:
            raise serializers.ValidationError(
                "Invalid verification request."
            )

        if user.is_active:
            raise serializers.ValidationError(
                "Account is already verified."
            )

        if not OTPService.verify_otp(
            user,
            submitted_otp,
        ):
            raise serializers.ValidationError(
                "Invalid or expired OTP."
            )

        user.is_active = True
        user.save(
            update_fields=["is_active"]
        )

        return attrs


class ResendOTPSerializer(serializers.Serializer):

    username = serializers.CharField()

    def validate(self, attrs):

        username = attrs["username"]

        try:
            user = User.objects.get(
                username=username
            )
        except User.DoesNotExist:
            raise serializers.ValidationError(
                "Invalid request."
            )

        if user.is_active:
            raise serializers.ValidationError(
                "Account is already verified."
            )

        otp = OTPService.create_and_store_otp(user)

        OTPService.send_otp(user, otp)

        return attrs