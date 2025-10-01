from app.database import SessionLocal
from app import models
from app.config import settings
from app.utils import hash_password

def init_master_user():
    db = SessionLocal()
    master = db.query(models.User).filter(models.User.email == settings.MASTER_EMAIL).first()
    if not master:
        user = models.User(
            name=settings.MASTER_NAME,
            email=settings.MASTER_EMAIL,
            password_hash=hash_password(settings.MASTER_PASSWORD),
            role=settings.MASTER_ROLE
        )
        db.add(user)
        db.commit()
        print("✅ Master user criado com sucesso!")
    else:
        print("ℹ️ Master user já existe.")
    db.close()
