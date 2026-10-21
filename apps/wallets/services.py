from decimal import Decimal
from uuid import uuid4
from django.db import transaction
from .models import Wallet, Transaction


class WalletService:
    @staticmethod
    @transaction.atomic     

    def deposit(wallet: wallet_id, amount: Decimal) -> Transaction:

        wallet = Wallet.objects.select_for_update().get(id=wallet_id)
        amount = Decimal(amount)

        if amount <= 0:
            raise ValueError ("Deposit amount must be greater than zero.")

        if wallet.status != Wallet.Status.ACTIVE:
            raise ValueError ("Wallet is not active.")

        wallet.balance += amount
        wallet.save(update_fields=["balance", "updated_at"])

        transaction_record = Transaction.objects.create(
            wallet=wallet,
            transaction_type=Transaction.TransactionType.DEPOSIT,
            amount=amount,
            currency=wallet.currency,
            status=Transaction.Status.SUCCESS,
            reference=f"DEP-{uuid4().hex}",
            description="Wallet deposit",
        )

        return transaction_record