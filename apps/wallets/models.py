from django.db import models
from django.conf import settings
from django.db import models


class Wallet(models.Model):

    class Currency(models.TextChoices):
        NGN = "NGN", "Nigerian Naira"
        USD = "USD", "US Dollar"
        EUR = "EUR", "Euro"

    class Status(models.TextChoices):
        ACTIVE = "active", "Active"
        FROZEN = "frozen", "Frozen"
        CLOSED = "closed", "Closed"

    user = models.ForeignKey(
        settings.AUTH_USER_MODEL,
        on_delete=models.CASCADE,
        related_name="wallets",
    )

    currency = models.CharField(
        max_length=3,
        choices=Currency.choices,
    )

    balance = models.DecimalField(
        max_digits=18,
        decimal_places=2,
        default=0,
    )

    status = models.CharField(
        max_length=10,
        choices=Status.choices,
        default=Status.ACTIVE,
    )

    created_at = models.DateTimeField(
        auto_now_add=True,
    )

    updated_at = models.DateTimeField(
        auto_now=True,
    )

    class Meta:
        constraints = [
            models.UniqueConstraint(
                fields=["user", "currency"],
                name="unique_user_currency_wallet",
            )
        ]

    def __str__(self):
        return f"{self.user.username} - {self.currency}"








class Transaction(models.Model):

    class TransactionType(models.TextChoices):
        DEPOSIT = "deposit", "Deposit"
        WITHDRAWAL = "withdrawal", "Withdrawal"
        TRANSFER = "transfer", "Transfer"
        PAYMENT = "payment", "Payment"
        REFUND = "refund", "Refund"
        EXCHANGE = "exchange", "Exchange"

    class Status(models.TextChoices):
        PENDING = "pending", "Pending"
        SUCCESS = "success", "Success"
        FAILED = "failed", "Failed"
        REVERSED = "reversed", "Reversed"

    wallet = models.ForeignKey(
        Wallet,
        on_delete=models.PROTECT,
        related_name="transactions",
    )

    transaction_type = models.CharField(
        max_length=20,
        choices=TransactionType.choices,
    )

    amount = models.DecimalField(
        max_digits=18,
        decimal_places=2,
    )

    currency = models.CharField(
        max_length=3,
    )

    status = models.CharField(
        max_length=20,
        choices=Status.choices,
        default=Status.PENDING,
    )

    reference = models.CharField(
        max_length=100,
        unique=True,
    )

    description = models.TextField(
        blank=True,
    )

    created_at = models.DateTimeField(
        auto_now_add=True,
    )

    updated_at = models.DateTimeField(
        auto_now=True,
    )

    def __str__(self):
        return f"{self.transaction_type} - {self.amount} - {self.reference}"



