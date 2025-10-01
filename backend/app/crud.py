from sqlalchemy.orm import Session
from app import models, utils

def get_user_by_email(db: Session, email: str):
    return db.query(models.User).filter(models.User.email == email).first()

def create_user(db: Session, nome: str, email: str, password: str, role: str = "tecnico"):
    hashed = utils.hash_password(password)
    db_user = models.User(nome=nome, email=email, hashed_password=hashed, role=role)
    db.add(db_user)
    db.commit()
    db.refresh(db_user)
    return db_user

def authenticate_user(db: Session, email: str, password: str):
    user = get_user_by_email(db, email)
    if not user:
        return None
    if not utils.verify_password(password, user.hashed_password):
        return None
    return user

# Projects
def create_project(db: Session, nome: str, localizacao: str = None):
    proj = models.Project(nome=nome, localizacao=localizacao)
    db.add(proj); db.commit(); db.refresh(proj)
    return proj

def list_projects(db: Session, skip: int = 0, limit: int = 100):
    return db.query(models.Project).offset(skip).limit(limit).all()

# MIBs
def create_mib(db: Session, arquivo: str, oid: str = None, descricao: str = None, project_id: int = None):
    mib = models.MIB(arquivo=arquivo, oid=oid, descricao=descricao, project_id=project_id)
    db.add(mib); db.commit(); db.refresh(mib)
    return mib

def list_mibs(db: Session, skip: int = 0, limit: int = 100):
    return db.query(models.MIB).offset(skip).limit(limit).all()

def list_mibs_by_project(db: Session, project_id: int):
    return db.query(models.MIB).filter(models.MIB.project_id == project_id).all()
