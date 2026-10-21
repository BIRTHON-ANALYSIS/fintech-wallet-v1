from django.urls import path

from .views import (
    RegisterView,
    LoginView,
    LogoutView,
    UserProfileView,
    VerifyEmailOTPView,
    ResendEmailOTPView
    #    PasswordResetRequestView,    PasswordResetConfirmView,   
   
)

urlpatterns = [
    path("register/", RegisterView.as_view(), name="register"),

    # JWT
    path("logout/", LogoutView.as_view(), name="logout"),

    # User
    path("me/", UserProfileView.as_view(), name="profile"),

    # Email Verification
    path("verify-email/", VerifyEmailOTPView.as_view(), name="verify-email"),
    path("resend-otp/", ResendEmailOTPView.as_view(), name="resend-otp"),

    # Password Reset
    path("login/", LoginView.as_view(), name="login"),

    #path(
    #    "password-reset/confirm/",
     #   PasswordResetConfirmView.as_view(),
      #  name="password-reset-confirm", ),
]