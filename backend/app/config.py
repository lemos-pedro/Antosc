from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    # PostgreSQL
    POSTGRES_USER: str = "postgres"
    POSTGRES_PASSWORD: str = "postgres"
    POSTGRES_SERVER: str = "127.0.0.1"
    POSTGRES_PORT: str = "5432"
    POSTGRES_DB: str = "antosc"

    # App
    SECRET_KEY: str = "uma_chave_muito_segura"
    UPLOAD_DIR: str = "uploads"

    # Master user (seed)
    MASTER_NAME: str = "Administrador"
    MASTER_EMAIL: str = "admin@antosc.com"
    MASTER_PASSWORD: str = "admin123"
    MASTER_ROLE: str = "admin"

    class Config:
        env_file = ".env"

settings = Settings()
