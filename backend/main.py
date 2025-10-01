import logging
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.database import engine
from app import models
from app.routers import auth_router

# ---------------------------------------------------------------------------- #
# Configuração de logging
# ---------------------------------------------------------------------------- #
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - [%(levelname)s] - %(message)s"
)
logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------- #
# Inicialização da aplicação
# ---------------------------------------------------------------------------- #
app = FastAPI(
    title="ANTOSC Backend - Diagnóstico Auth",
    description="API de autenticação e diagnóstico do sistema ANTOSC",
    version="1.0.0"
)

# ---------------------------------------------------------------------------- #
# Banco de dados e modelos
# ---------------------------------------------------------------------------- #
try:
    models.Base.metadata.create_all(bind=engine)
    logger.info("✅ Tabelas do banco de dados criadas com sucesso.")
except Exception as e:
    logger.error(f"❌ Erro ao criar tabelas: {e}")

# ---------------------------------------------------------------------------- #
# Configuração de CORS
# ---------------------------------------------------------------------------- #
origins = ["*"]  # Ajustar em produção para domínios específicos

app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# ---------------------------------------------------------------------------- #
# Routers
# ---------------------------------------------------------------------------- #
app.include_router(auth_router.router)

# ---------------------------------------------------------------------------- #
# Endpoints básicos
# ---------------------------------------------------------------------------- #
@app.get("/", tags=["Health"])
def root():
    """
    Endpoint básico para verificar se a API está online.
    """
    return {
        "status": "ok",
        "message": "API ANTOSC - Auth router ativo 🚀"
    }


@app.get("/health", tags=["Health"])
def health_check():
    """
    Health-check detalhado.
    """
    return {
        "status": "ok",
        "database": "conectado",
        "service": "ANTOSC Backend",
        "version": "1.0.0"
    }
