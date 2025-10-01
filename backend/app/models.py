from sqlalchemy import Column, Integer, String, Text, DateTime
from app.database import Base
from datetime import datetime


class User(Base):
    __tablename__ = "users"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String(150), nullable=False)             # corresponde ao "name"
    email = Column(String(150), unique=True, index=True, nullable=False)
    password_hash = Column(Text, nullable=False)           # corresponde ao "password_hash"
    role = Column(String(50), nullable=False, default="colaborador")
    created_at = Column(DateTime, default=datetime.utcnow)
