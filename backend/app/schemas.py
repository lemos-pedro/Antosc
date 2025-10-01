from pydantic import BaseModel, EmailStr
from typing import Optional, List
from datetime import datetime

class UserCreate(BaseModel):
    nome: str
    email: EmailStr
    password: str
    role: Optional[str] = "tecnico"

class UserOut(BaseModel):
    id: int
    nome: str
    email: EmailStr
    role: str
    created_at: datetime
    
    class Config:
        from_attributes = True   # <-- atualizado para Pydantic v2

class Token(BaseModel):
    access_token: str
    token_type: str

class ProjectCreate(BaseModel):
    nome: str
    localizacao: Optional[str] = None

class ProjectOut(BaseModel):
    id: int
    nome: str
    localizacao: Optional[str]
    created_at: datetime
    
    class Config:
        from_attributes = True   # <-- atualizado

class MIBOut(BaseModel):
    id: int
    arquivo: str
    oid: Optional[str]
    descricao: Optional[str]
    project_id: Optional[int]
    created_at: datetime
    
    class Config:
        from_attributes = True   # <-- atualizado
